# Canonicalisation profiles

**Status: specification-only.** This document defines two canonicalisation
profiles and their wire/versioning contract. It does **not** describe a shipped
strict canonicaliser. In this revision ATB still produces and verifies bundles
with the existing legacy serialisers only. See [Status and next
steps](#status-and-next-steps).

---

## 1. Scope and non-goals

This document is normative for:

- the identity and meaning of the `atb-jcs-legacy-1` and
  `atb-jcs-rfc8785-1` canonicalisation profiles;
- the byte-level behaviour that each profile guarantees;
- the rejection taxonomy used when a strict-profile parser is introduced;
- the manifest-version and selection contract that will govern strict-profile
  bundles.

This document deliberately does **not**:

- add a strict canonicaliser implementation;
- switch any production writer, reader, verifier, hashing path, or manifest
  parser to the strict profile;
- issue manifest version 4, or change `ManifestVersionMax` (still `3` in this
  revision);
- change the hash formula, genesis hash, or any existing golden vector or
  fixture;
- relax the existing legacy behaviour in any language SDK.

The authored fixtures under [`test/jcs/`](../../test/jcs/README.md) are part of
this specification. They are normative inputs to a future conformance runner;
they are not executable strict-parser tests yet.

---

## 2. Profile identifiers

| Profile id             | Meaning                                                       | Current status |
|------------------------|---------------------------------------------------------------|----------------|
| `atb-jcs-legacy-1`     | Byte-frozen behaviour of the serialisers shipped at `a7837e2`. | In production  |
| `atb-jcs-rfc8785-1`    | Strict RFC 8785 profile with the ATB restrictions in §5.      | Specification  |

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

---

## 5. Conservative safe-integer policy

The strict profile imposes an **ATB interoperability restriction** on
integer-valued numeric literals:

- Reject any integer-valued numeric literal with an absolute value greater
  than `2^53 - 1` (9,007,199,254,740,991) with `number_out_of_policy`.
- Reject non-finite literals (`NaN`, `Infinity`, `-Infinity`) with
  `non_finite_number`.
- Reject finite-looking literals that cannot be represented as binary64 with
  `number_unrepresentable`.

This is a conservative policy for **new strict-profile input**. It is not a
claim that RFC 8785 universally rejects larger numbers: RFC 8785 permits finite
binary64 values generally, while RFC 7493 recommends keeping integers within
the `2^53 - 1` interoperability range. ATB chooses the narrower range so strict
bundles cannot silently lose integer precision across languages. Bundles under
the legacy profile are unaffected.

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
| `number_out_of_policy`  | Integer-valued literal outside the safe-integer policy in §5.             |
| `number_unrepresentable`| Numeric literal that cannot be represented as a finite binary64 value.    |
| `invalid_json`          | The input is not syntactically valid JSON for another reason.             |
| `missing_profile`       | Manifest v4 does not declare `canonical_profile`.                         |
| `unsupported_profile`   | Manifest v4 declares an unknown or unsupported profile id.                |

`missing_profile` and `unsupported_profile` are manifest-selection errors; the
remaining classes are payload validation errors. Both sets are fail-closed.

Negative fixtures for the payload classes are authored in
[`test/jcs/strict_negative_vectors.json`](../../test/jcs/strict_negative_vectors.json).
They are not executed against a strict parser in this revision because none
exists.

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

## 8. Migration acceptance prerequisites

Before any production switching, all of the following must land as explicit,
separately reviewed changes:

1. A strict canonicaliser implementing §4–§6, with a conformance runner that
   executes the positive and negative fixtures in `test/jcs/`.
2. A manifest-v4 writer path that is opt-in and emits `canonical_profile`.
3. Reader/verifier dispatch that selects the profile by declaration only and
   fails closed for missing/unknown profiles.
4. A deliberate `ManifestVersionMax` bump to `4`, with the forward-compatibility
   guard updated in the same change.
5. Cross-language strict-profile parity for Go, Python, and TypeScript, with new
   strict vectors authored independently of any ATB canonicaliser.
6. A `CHANGELOG.md` entry and the version-bump treatment required by
   `VERSIONING.md`.
7. Confirmation that no existing golden vector, fixture, legacy hash, or
   signature pre-image changes.

---

## 9. Status and next steps

In this revision:

- The legacy profile is the only implemented and dispatched profile.
- The strict profile exists **only as this specification and its authored
  vectors**.
- `ManifestVersionMax` is `3`; manifest v4 is not issued or accepted.
- No production canonicaliser, hasher, manifest writer/parser, or verifier
  dispatch has changed.

The next step is to implement the strict canonicaliser and a conformance runner
that executes [`test/jcs/`](../../test/jcs/README.md) without changing legacy
behaviour.
