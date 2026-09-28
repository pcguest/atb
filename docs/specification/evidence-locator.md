# ATB semantic evidence locator

## 1. Purpose

A **semantic evidence locator** is a stable, versioned reference to a single
evidence record inside one ATB bundle. It lets another system (for example a
Mortise governance workspace, or an agent that cites evidence) name a specific
record without depending on ATB's file paths, viewer routes, or UI structure.

> An ATB evidence locator identifies evidence. It does not grant authority,
> establish factual truth, or imply governance approval.

## 2. Non-goals

A locator is a reference layer only. It does not:

- verify integrity (resolving a locator is **not** a hash-chain verification);
- discover bundles globally (a locator carries no filesystem path);
- grant access, authority, or approval;
- establish that evidence is true, complete, or trustworthy;
- change canonicalisation, the hash chain, the bundle format, or any event
  field.

## 3. Grammar

Grammar version **1**:

```
atb://evidence/1/<bundle_head_hash>?seq=<event_sequence>[&record=<record_hash>]
```

- `evidence` is the locator class authority. It names the *kind* of reference,
  not a network host.
- `/1` is the grammar version segment. Unknown versions fail closed.
- `<bundle_head_hash>` is the bundle's terminal record hash (64 lowercase hex).
- `?seq=` is required and is the target record's `seq` value.
- `&record=` is optional and is an integrity cross-check of the referenced
  record's stored hash.

Query parameters must be exactly `seq` and (optionally) `record`. Unknown
parameters, duplicate parameters, empty query components (a trailing, leading,
or doubled `&`), fragments (including a bare `#`), userinfo, ports, and
percent-encoding are rejected.

## 4. Canonical form

There is exactly one canonical serialised form:

- scheme `atb`, authority `evidence`, version `1` (all lowercase);
- the version segment has no leading zeros (`1`, never `01`);
- one path segment: the head hash;
- query in the order `seq` then (if present) `record`;
- decimal `seq` with no leading zeros (`0` is permitted);
- no fragment.

`Parse` accepts the canonical form. It also accepts supported equivalent query
parameter orderings (for example `record` before `seq`); accepted textual
variants parse to the same semantic `Locator`. `Locator.String()` always emits
the one canonical serialisation (`?seq=<n>` and, if present, `&record=<hash>`),
so `Parse(String(Parse(x))) == Parse(x)` holds for every accepted locator.

`Parse` rejects forms that are invalid or non-canonical rather than normalising
them. The grammar-level rejections are catalogued in [§3](#3-grammar);
additionally, `Parse` rejects a non-lowercase scheme or authority, a
non-canonical or unsupported version segment (for example the non-canonical
`01`), malformed hashes, and a `seq` with leading zeros.

## 5. Field semantics

### `bundle_head_hash`

- Algorithm: SHA-256.
- Representation: 64 lowercase hexadecimal characters, no `0x` prefix.
- Meaning: the hash of the **terminal record** in the bundle, i.e. the
  `hash` field of the last record. It is the same value the viewer reports as
  `head_hash` in `GET /api/v1/verification`.
- Empty bundle: an empty bundle has no terminal record, so no bundle head
  exists and resolution fails with `BUNDLE_NOT_AVAILABLE`.
- Case: uppercase is rejected; the canonical form is lowercase.

### `event_sequence`

- Meaning: the value of the referenced record's `seq` field.
- Base: bundles that contain a manifest record reserve `seq = 0` for the
  manifest; all subsequent records are numbered `1..N` by their `seq` field
  (which equals the record's 0-based index for manifest bundles). Legacy
  manifest-less bundles number their first record `seq = 1`. The locator
  references the stored `seq` verbatim; it is **not** an array offset.
- Negative, malformed, out-of-range, or overflowing values are rejected. The
  resolver never clamps.

### `record_hash`

- Optional. Algorithm SHA-256, 64 lowercase hex, the referenced record's
  stored `hash`.
- When present, a mismatch is reported as `RECORD_HASH_MISMATCH`. This is an
  integrity cross-check of the **record hashes as stored**, not a full
  hash-chain verification of the bundle.

## 6. Versioning

The grammar version lives in the locator path (`/1/`), never in a hashed event
field. A locator is a reference layer over evidence identity, not evidence
itself. A locator that declares an unknown version fails with
`LOCATOR_VERSION_UNSUPPORTED` rather than being guessed.

## 7. Resolution model

A locator identifies; it does not acquire. There is no global registry, daemon,
database, or network lookup. The smallest truthful model is used:

1. **The referenced bundle is supplied explicitly.** `atb view` loads a bundle
   from `--bundle`/positional path; resolution runs against that loaded bundle.
2. **The head hash is matched** against the bundle's terminal record hash. A
   mismatch (or no bundle) yields `BUNDLE_NOT_AVAILABLE`. A hash is not a path.
3. **The sequence is matched** against the records. Absence yields
   `EVENT_NOT_FOUND`. If a (corrupt or tampered) bundle contains duplicate
   `seq` values, the **first matching record in bundle order** is returned;
   sequence uniqueness is an integrity property checked by bundle
   verification, not by the locator.
4. **The optional record hash is matched** against the referenced record's
   stored hash. Mismatch yields `RECORD_HASH_MISMATCH`.

Resolution performs no filesystem search and no cryptographic verification.
The caller is responsible for supplying the bundle and for any integrity check.

## 8. Failure semantics

Errors are typed sentinels (matchable via `errors.Is`) in package
`internal/locator`, mirroring the `internal/bundle` sentinel style:

| Sentinel / code | Meaning |
| --- | --- |
| `ErrMalformed` / `LOCATOR_MALFORMED` | Syntactically invalid locator |
| `ErrUnsupportedVersion` / `LOCATOR_VERSION_UNSUPPORTED` | Grammar version not understood |
| `ErrBundleUnavailable` / `BUNDLE_NOT_AVAILABLE` | Head mismatch, or no bundle |
| `ErrEventNotFound` / `EVENT_NOT_FOUND` | Sequence absent from the bundle |
| `ErrRecordHashMismatch` / `RECORD_HASH_MISMATCH` | Optional record hash did not match |

> Failure to locate evidence is not evidence that the record is invalid.
> Successfully locating evidence is not proof that the evidence is
> trustworthy, complete, or factually correct.

## 9. Security considerations

- A locator is never treated as a filesystem path. Bundle acquisition stays a
  separate, explicit operation (`--bundle`), so a crafted locator cannot cause
  unintended file access.
- Parsing is strict and bounded: percent-encoding, fragments, duplicate
  parameters, unknown parameters, non-canonical hashes, and out-of-range
  sequences are rejected. There is no unbounded filesystem traversal.
- The viewer displays a locator result as a **location** outcome, never as a
  trust or integrity verdict. A location failure must not be presented as
  tampering.
- The viewer `focus` query parameter is a browser transport detail, not the
  canonical evidence identity; the canonical identity remains the locator.

## 10. Examples

```
atb://evidence/1/2f8a…(64 hex)…?seq=4
atb://evidence/1/2f8a…(64 hex)…?seq=4&record=9c31…(64 hex)…
```

Round-trip:

```
l, _ := locator.Parse(s)
locator.Parse(l.String()) // == l
```

## 11. ATB ↔ Mortise use

Mortise (or an agent citing evidence) may store or display a locator so a human
can open the referenced record in the local ATB viewer:

```
atb view --bundle path/to/session.atb --focus 'atb://evidence/1/<head>?seq=4'
```

The viewer focuses the referenced record and continues to show the bundle
context. Opening evidence does not imply that the evidence has been verified:
integrity remains a separate, explicitly performed step, and governance
approval remains a Mortise concern.

## 12. Location is not integrity

Three questions stay separate:

1. **Location** — which record does this reference name? (this spec)
2. **Integrity** — does the bundle hash-chain verify? (`atb verify`, viewer gate)
3. **Interpretation / governance** — what does the evidence mean, and who
   approved what? (Mortise)

A locator answers only the first.

A locator is intentionally orthogonal to attribution
([evidence attribution](./attribution.md)): the locator answers *which record*,
while attribution describes *what that record reports as its producer and
context*. Resolving a locator neither verifies integrity nor asserts
authorship or authority.
