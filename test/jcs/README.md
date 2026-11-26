# JCS canonicalisation-profile fixtures

**Status: specification plus non-production strict implementations.** The
fixtures in this directory describe three canonicalisation profiles. The frozen
`atb-jcs-rfc8785-1` strict implementation and the additive
`atb-jcs-rfc8785-2` canonical-output-closed implementation have executable
conformance runners, but no production behaviour has switched: hashing,
manifest-v4 issuance/acceptance, and verifier dispatch still use the legacy
profile only. The normative specification is
[`docs/specification/canonicalisation-profiles.md`](../../docs/specification/canonicalisation-profiles.md).

## Profiles

- **`atb-jcs-legacy-1`** — the byte-frozen behaviour of the serialisers that
  ship at the current ATB revision. Existing bundles depend on it.
- **`atb-jcs-rfc8785-1`** — the frozen strict profile: RFC 8785 plus the ATB
  safe-integer input restriction. Its bytes and error classes must not change.
- **`atb-jcs-rfc8785-2`** — the additive canonical-output-closed strict
  profile: retains the v1 plain-integer input cap and rejects fixed-notation
  canonical outputs above `2^53 - 1`. It is non-production and not dispatched.

Manifest version **4** with the field **`canonical_profile`** is the planned
wire declaration for strict-profile bundles. In this revision
`ManifestVersionMax` is still `3`, so no v4 bundle is issued or accepted.

## Files

| File | Purpose |
|------|---------|
| `strict_positive_vectors.json` | Frozen v1 strict input/expected-canonical byte pairs, grouped by category. |
| `strict_negative_vectors.json` | Frozen v1 strict rejection cases and their error class. |
| `strict_v2_positive_vectors.json` | Executable v2 positive byte pairs. It mirrors the v1 set except for the authored `pos-number-thresholds-v2` case. |
| `strict_v2_negative_vectors.json` | Executable v2 canonical-output-policy rejections, all `number_out_of_policy`. |
| `numeric_parity_cases.json` | Frozen legacy-vs-strict F6 parity case for `1e-6`, including the Python legacy divergence. |
| `legacy_freeze_vectors.json` | Frozen raw and value expectations captured from the implementations at `a7837e2`. |
| `profile_selection_cases.json` | Manifest-version/profile declarations and how a current reader and a future strict reader must treat them. |

## Provenance and do-not-regenerate rules

- **Strict expected bytes were authored** against RFC 8785 and ECMA-262
  `Number::toString` with UTF-16 code-unit key ordering. They were **not**
  produced by any ATB canonicaliser. They are the independent oracle for the
  strict implementation.
- **Negative cases are now executable** against the new, non-production strict
  canonicaliser via `strict_conformance_test.go`; the declared error class must
  match the implementation's reported class.
- **Legacy expectations are frozen** from the implementations at `a7837e2`.
  They must not be regenerated. A change to any legacy expectation is a
  breaking canonicalisation change and requires the manifest-version and
  migration process in `VERSIONING.md`.
- **v1 strict fixtures are frozen; v2 strict fixtures are executable and may be
  independently extended.** New v2 cases may be appended without changing
  existing case ids or expected bytes, provided their expected results are
  authored independently of the implementation. Legacy fixtures remain frozen
  and must not be regenerated. Tests in Go, Python, and TypeScript load and
  validate all fixtures read-only. The v1 and v2 files intentionally reuse most
  case ids because v2 is a delta of the v1 corpus; id uniqueness is enforced
  within each file, and the numeric-parity id is distinct.

## Consumers

- `test/jcs/vectors_test.go` validates the v1 and legacy fixture metadata and
  pins the Go legacy behaviour.
- `test/jcs/strict_conformance_test.go` executes the v1 positive and negative
  strict vectors against the non-production v1 entry points.
- `test/jcs/strict_v2_conformance_test.go` executes the v2 vectors, runs the v1
  negatives through v2, checks raw round-trip closure, and pins the
  numeric-parity case.
- `internal/bundle/canonical_profile_test.go` pins the current reader's
  fail-closed handling of v3/v4 profile-selection cases.
- `sdk/python/tests/test_jcs_spec_fixtures.py` validates fixture metadata and
  pins Python legacy and numeric-parity behaviour; the strict v1 and v2
  conformance modules execute the byte cases.
- `sdk/typescript/src/jcs-spec-fixtures.test.ts` validates fixture metadata and
  pins TypeScript legacy and numeric-parity behaviour; the strict v1 and v2
  conformance modules execute the byte cases.

## Future work

A later change will add the manifest-v4 writer and declaration-only reader
dispatch described in the specification, and must explicitly choose a strict
profile. `atb-jcs-rfc8785-2` is defined but not dispatched. Production hashing
and verification stay on the legacy profile until then.
