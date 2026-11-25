# Canonicalisation profiles

**Status: specification plus non-production strict implementation.** This
document defines two canonicalisation profiles and their wire/versioning
contract. A strict canonicaliser for `atb-jcs-rfc8785-1` now exists and its
fixtures are executable, but no production writer, reader, verifier, hashing
path, or manifest parser dispatches to it. In this revision ATB still produces
and verifies bundles with the existing legacy serialisers only. See [Status and
next steps](#status-and-next-steps).

---

## 1. Scope and non-goals

This document is normative for:

- the identity and meaning of the `atb-jcs-legacy-1` and
  `atb-jcs-rfc8785-1` canonicalisation profiles;
- the byte-level behaviour that each profile guarantees;
- the rejection taxonomy enforced by the strict-profile parser;
- the manifest-version and selection contract that will govern strict-profile
  bundles.

This document deliberately does **not**:

- switch any production writer, reader, verifier, hashing path, or manifest
  parser to the strict profile;
- issue manifest version 4, or change `ManifestVersionMax` (still `3` in this
  revision);
- change the hash formula, genesis hash, or any existing golden vector or
  fixture;
- relax the existing legacy behaviour in any language SDK.

The fixtures under [`test/jcs/`](../../test/jcs/README.md) are part of this
specification. The strict positive and negative fixtures are now executed by
conformance runners against the non-production strict implementation; they
remain the independent oracle and are not derived from implementation output.

---

## 2. Profile identifiers

| Profile id             | Meaning                                                       | Current status |
|------------------------|---------------------------------------------------------------|----------------|
| `atb-jcs-legacy-1`     | Byte-frozen behaviour of the serialisers shipped at `a7837e2`. | In production  |
| `atb-jcs-rfc8785-1`    | Strict RFC 8785 profile with the ATB restrictions in §5.      | Non-production implementation |

The legacy profile id is descriptive: it names the behaviour that existing
bundles already depend on. It does not imply that the legacy serialisers are
fully RFC 8785 conformant. The strict profile id names the target behaviour for
future manifest-v4 bundles.

Profile selection is **by declaration only** (see §7). A reader must never infer
the strict profile from the presence of a field, a version, or a payload shape.

---

## 3. Legacy profile: `atb-jcs-legacy-1`

The legacy profile is the **byte-frozen** behaviour of the canonicalisers that
ship with ATB at commit `a7837e2` (Go runtime, Python SDK, TypeScript SDK). Its
purpose is to keep every existing bundle verifiable, not to be a clean RFC 8785
implementation.

Frozen properties:

- **Value-lossy JSON decode.** Input is decoded through each language's generic
  JSON loader before re-serialisation. Integers outside the exactly
  representable binary64 range may be rounded (Go, TypeScript) or preserved
  (Python), depending on the serialiser. This is why the legacy profile is
  described per implementation rather than as one cross-language byte function.
- **Duplicate names: last value wins.** Duplicate members are not rejected.
- **Invalid UTF-8 and lone surrogates are replaced.** Malformed UTF-8 byte
  sequences and unpaired surrogate escapes become U+FFFD (`EF BF BD`). There is
  no rejection.
- **Go number formatting.** Go zero-pads exponents (`1e-07`) and preserves
  negative zero (`-0`). Python zero-pads exponents; TypeScript follows ES6
  (`1e-7`, `0` for `-0`).
- **UTF-16 key ordering.** Object keys are ordered by UTF-16 code units. The
  Go and Python serialisers order a NUL-containing key correctly; the
  TypeScript legacy comparator treats U+0000 as `-1` and can preserve insertion
  order instead. This divergence is frozen, not fixed.
- **Hash formula and genesis unchanged.** Hashes remain
  `SHA-256(UTF-8(hex(prev_hash)) || canonical(event))` with the 64-zero genesis
  hash. The legacy profile does not alter the hash pre-image or chain rules.

Frozen references:

- [`internal/hash/testdata/canonical_vectors.json`](../../internal/hash/testdata/canonical_vectors.json)
  — the cross-language canonical-hash corpus that existing bundles are pinned
  to.
- [`test/jcs/legacy_freeze_vectors.json`](../../test/jcs/legacy_freeze_vectors.json)
  — raw and value cases captured from the implementations at `a7837e2`.

These expectations must **not** be regenerated. Changing any legacy
expectation is a breaking canonicalisation change and requires the migration and
version-bump process in `VERSIONING.md`.

---

## 4. Strict profile: `atb-jcs-rfc8785-1`

The strict profile is defined by the following normative sources, applied
together:

- [RFC 8785 — JSON Canonicalization Scheme (JCS)](https://www.rfc-editor.org/rfc/rfc8785)
- [RFC 7493 — The I-JSON Message Format](https://www.rfc-editor.org/rfc/rfc7493)
  (for the interoperability constraints that JCS assumes)
- ECMA-262 `Number::toString` and `JSON.stringify`, as referenced by RFC 8785
  for number serialisation

### 4.1 Encoding and strings

- The input and output are **UTF-8**. Malformed UTF-8 is rejected, never
  repaired.
- Lone or unpaired surrogate escapes (`\uD800`–`\uDFFF` without a valid pair)
  are rejected. There is **no U+FFFD substitution**.
- Non-ASCII characters are emitted **literally** as UTF-8, not `\u`-escaped.
- The short escapes `\"`, `\\`, `\b`, `\f`, `\n`, `\r`, `\t` are used. Other
  C0 control characters (U+0000–U+001F) are emitted as lowercase `\u00xx`
  (for example `\u0001`).
- The solidus `/` is not escaped.

### 4.2 Objects and names

- Member names are sorted by their **UTF-16 code-unit sequence**, matching
  RFC 8785 §3.2.3. Suffix ordering is by code unit, with correct handling of
  U+0000 (a NUL is a code unit, not a terminator).
- Duplicate member names are rejected **after unescaping**, at every depth,
  including names that are escaped-equivalent to a literal name (for example
  `"a"` and `"\u0061"` in the same object).

### 4.3 Numbers

- Input numbers must be finite binary64 values. `NaN`, `Infinity`, and
  `-Infinity` are rejected. A numeric literal that overflows binary64 is
  rejected as unrepresentable.
- Finite numbers are serialised per ECMA-262 `Number::toString`, including:
  - `1e-7` (not `1e-07`);
  - `0` for negative zero (not `-0`);
  - fixed notation below the `1e21` threshold and exponent notation at and
    above it (for example `1e+21`);
  - the shortest round-tripping representation.
- Trailing data after the top-level JSON value is rejected.

### 4.4 Nesting depth

Array, object, struct, map, and pointer nesting is bounded to a maximum depth of
512. Input that nests more deeply — including through pointer chains in
value-mode input — is rejected with `depth_limit` rather than risking stack
exhaustion.

---

## 5. Conservative safe-integer policy

The strict profile imposes an ATB interoperability restriction on plain integer
tokens: reject a numeric token written without a fraction or exponent (an
optional minus sign followed only by digits) whose exact integer magnitude is
greater than 2^53 - 1 with `number_out_of_policy`; reject `NaN`, `Infinity`,
and `-Infinity` with `non_finite_number`; reject numeric tokens that parse to a
non-finite binary64 with `number_unrepresentable`. Numbers written with a
fraction or exponent are binary64 values governed by §4.3 and are not subject to
that token restriction; for example, `1e20` is accepted, while the plain integer
token `9007199254740993` is rejected. This is an ATB interoperability
restriction for new strict-profile input, not a claim that RFC 8785 universally
rejects larger numbers. Because accepted fraction/exponent literals are not
subject to the token restriction, an accepted exponent form such as `1e20` can
canonicalise to a plain integer token outside the policy
(`100000000000000000000`), which a fresh strict parse of that output would
reject as `number_out_of_policy`; strict output is therefore not guaranteed to
be re-accepted as fresh raw input. This is a known v1 limitation; resolving it
requires a versioned policy decision, not a silent tightening of the current
profile.

Value-mode input has parallel distinctions. Dynamic JSON fragments (`json.Number`
in Go, `json.RawMessage`, and custom `json.Marshaler` or
`encoding.TextMarshaler` implementations) are strictly validated through the
same parser, so they cannot smuggle non-finite, out-of-policy, or invalid-UTF-8
values past the value API. Host-language type differences still remain: integer
and float types differ between Go, Python, and TypeScript, and a value accepted
by one language's value API is not automatically accepted by another's.

---

## 6. Error taxonomy

A strict-profile parser reports exactly one of the following stable classes:

| Error class             | Condition                                                                 |
|-------------------------|---------------------------------------------------------------------------|
| `duplicate_key`         | Duplicate member name after unescaping, at any depth.                     |
| `invalid_utf8`          | Input bytes are not valid UTF-8.                                          |
| `lone_surrogate`        | An escape produces an unpaired UTF-16 surrogate.                          |
| `trailing_data`         | Non-whitespace data follows the top-level JSON value.                     |
| `non_finite_number`     | The input denotes `NaN` or an infinity.                                   |
| `number_out_of_policy`  | Plain integer token outside the safe-integer policy in §5.                |
| `number_unrepresentable`| Numeric literal that cannot be represented as a finite binary64 value.    |
| `invalid_json`          | The input is not syntactically valid JSON for another reason.             |
| `depth_limit`           | Array, object, struct, map, or pointer nesting exceeds 512 levels.        |
| `missing_profile`       | Manifest v4 does not declare `canonical_profile`.                         |
| `unsupported_profile`   | Manifest v4 declares an unknown or unsupported profile id.                |

`missing_profile` and `unsupported_profile` are manifest-selection errors; the
remaining classes are payload validation errors. Both sets are fail-closed.

Negative fixtures for the payload classes are authored in
[`test/jcs/strict_negative_vectors.json`](../../test/jcs/strict_negative_vectors.json)
and are executed by the strict conformance runners in Go, Python, and
TypeScript. The declared classes are authored independently of the
implementation and are not derived from implementation output.

### 6.1 Error precedence and cross-language caveats

- Object member-value errors are reported before a duplicate-key rejection for
  that same member; earlier errors win by left-to-right order.
- Go and TypeScript generally report the first left-to-right error. Python's
  stdlib-based parser pre-scans depth and non-finite token boundaries and
  post-checks lone surrogates, so an input containing multiple independent
  defects can receive a different class; all such inputs are still rejected
  fail-closed. Shared vectors pin single-defect cases and the object-member
  precedence cases.
- Raw APIs differ by input representation: invalid UTF-8 bytes are
  `invalid_utf8`; a host string that already contains a lone surrogate can be
  `lone_surrogate` (TypeScript) or `invalid_json` (Python) depending on the
  decoder. Cross-language byte comparisons should use byte inputs.
- Value-mode depth counts pointer/struct levels as well as arrays/objects/maps;
  raw mode counts only JSON containers. A value exactly at the limit can
  therefore be rejected in value mode while its marshalled bytes are accepted by
  raw mode. This is a documented API asymmetry, not a canonical-byte difference
  for accepted inputs.
- Custom `json.Marshaler`/`TextMarshaler` implementations are invoked once for
  validation and again by the host serialiser, so they must be deterministic; a
  non-deterministic marshaler can otherwise pass validation and emit different
  bytes. If a custom marshaler cannot be interfaced for validation, the Go value
  API fails closed with `invalid_json`.

---

## 7. Wire and versioning contract

Strict-profile bundles will use **manifest version 4** with a new
`canonical_profile` field in the structured manifest payload:

```json
{
  "version": 4,
  "created_at": "2026-01-02T03:04:05Z",
  "bundle_id": "…",
  "canonical_profile": "atb-jcs-rfc8785-1"
}
```

Rules:

- **Selection is by declaration only.** A bundle is strict-profile only when
  its manifest declares version `4` **and** `canonical_profile` equals
  `atb-jcs-rfc8785-1`.
- **Manifest v1, v2, and v3 remain legacy.** Manifest-less legacy bundles also
  remain legacy (implicit version 1).
- **A v3 manifest carrying `canonical_profile` is not strict.** On v3 the field
  has no strict semantics; strict issuance must use v4. A v3 bundle with that
  field still reads as legacy.
- **v4 without `canonical_profile` fails closed** with `missing_profile`.
- **v4 with an unknown `canonical_profile` fails closed** with
  `unsupported_profile`; it is never downgraded to legacy.
- **Old readers reject v4.** A reader whose `ManifestVersionMax` is below 4
  rejects the bundle with an error wrapping `ErrMalformed` ("requires a newer
  version of atb"). In this revision `ManifestVersionMax` is still `3`.
- **No in-place rehash.** Introducing the strict profile does not rewrite or
  re-sign any existing bundle, and does not change the hash of any existing
  record.
- **The hash formula and genesis are unchanged.** Strict profile changes the
  canonicalisation of new input only; hashing remains
  `SHA-256(UTF-8(hex(prev_hash)) || canonical(event))` over the same genesis.
- **No automatic fallback.** A declared strict bundle that fails strict
  validation is never re-read under the legacy profile.

These cases are pinned by
[`test/jcs/profile_selection_cases.json`](../../test/jcs/profile_selection_cases.json)
and the current-reader acceptance test in
`internal/bundle/canonical_profile_test.go`.

---

## 8. Migration acceptance status

Each item below must land as an explicit, separately reviewed change. Items 1,
5, and 7 are complete for the isolated non-production increment; items 2, 3, 4,
and 6 are still required before any production switching.

1. **Complete (isolated increment).** A strict canonicaliser implementing §4–§6,
   with a conformance runner that executes the positive and negative fixtures in
   `test/jcs/`.
2. **Still required before production switching.** A manifest-v4 writer path
   that is opt-in and emits `canonical_profile`.
3. **Still required before production switching.** Reader/verifier dispatch that
   selects the profile by declaration only and fails closed for missing/unknown
   profiles.
4. **Still required before production switching.** A deliberate
   `ManifestVersionMax` bump to `4`, with the forward-compatibility guard
   updated in the same change.
5. **Complete (isolated increment).** Cross-language strict-profile parity for
   Go, Python, and TypeScript across the shared strict vectors and supported byte
   inputs, with new strict vectors authored independently of any ATB
   canonicaliser. This is not universal parity: the multi-error and host-string
   caveats in §6.1 remain.
6. **Still required before production switching.** A `CHANGELOG.md` entry and
   the version-bump treatment required by `VERSIONING.md`.
7. **Complete (isolated increment).** Confirmation that no existing golden
   vector, fixture, legacy hash, or signature pre-image changes.

---

## 9. Status and next steps

In this revision:

- The legacy profile is the only implemented and dispatched production profile.
- A strict canonicaliser now exists in `internal/canonicalize/strict`, exposed
  behind the new, non-production entry points `jcs.MarshalRFC8785`,
  `jcs.MarshalRawRFC8785`, and `jcs.StrictErrorClass`.
- The strict profile is **not** wired into hashing, manifest-v4 issuance or
  acceptance, or verifier dispatch.
- The strict implementation bounds nesting depth (including pointer traversal),
  detects cyclic values, strictly validates dynamic JSON values (`json.Number`,
  `json.RawMessage`, and custom marshalers), and defines duplicate-key versus
  member-value error precedence.
- `ManifestVersionMax` is `3`; manifest v4 is not issued or accepted.
- No production canonicaliser, hasher, manifest writer/parser, or verifier
  dispatch has changed.
- The conformance tests execute the positive and negative vectors in
  [`test/jcs/`](../../test/jcs/README.md) against the strict entry points.

The next step is to add the manifest-v4 writer and declaration-only reader
dispatch, then bump `ManifestVersionMax`, without changing legacy behaviour.
Production switching remains blocked on the remaining items in §8.
