# JCS canonicalisation-profile fixtures

**Status: specification plus non-production strict implementation.** The
fixtures in this directory describe two canonicalisation profiles. A strict
RFC 8785 implementation and an executable conformance runner now exist, but no
production behaviour has switched: hashing, manifest-v4 issuance/acceptance,
and verifier dispatch still use the legacy profile only. The normative
specification is
[`docs/specification/canonicalisation-profiles.md`](../../docs/specification/canonicalisation-profiles.md).

## Profiles

- **`atb-jcs-legacy-1`** — the byte-frozen behaviour of the serialisers that
  ship at the current ATB revision. Existing bundles depend on it.
- **`atb-jcs-rfc8785-1`** — the target strict profile: RFC 8785 plus the ATB
  safe-integer restriction.

Manifest version **4** with the field **`canonical_profile`** is the planned
wire declaration for strict-profile bundles. In this revision
`ManifestVersionMax` is still `3`, so no v4 bundle is issued or accepted.

## Files

| File | Purpose |
|------|---------|
| `strict_positive_vectors.json` | Strict-profile input/expected-canonical byte pairs, grouped by category. |
| `strict_negative_vectors.json` | Strict-profile rejection cases and their error class. |
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
- **Strict fixtures are executable and may be independently extended.** New
  strict cases may be appended without changing existing case ids or expected
  bytes, provided their expected results are authored independently of the
  implementation. Legacy fixtures remain frozen and must not be regenerated.
  Tests in Go, Python, and TypeScript load and validate all fixtures read-only.

## Consumers

- `test/jcs/vectors_test.go` validates all four files and pins the Go legacy
  behaviour.
- `test/jcs/strict_conformance_test.go` executes the positive and negative
  strict vectors against the non-production strict entry points.
- `internal/bundle/canonical_profile_test.go` pins the current reader's
  fail-closed handling of v3/v4 profile-selection cases.
- `sdk/python/tests/test_jcs_spec_fixtures.py` validates the fixtures and pins
  Python legacy behaviour.
- `sdk/typescript/src/jcs-spec-fixtures.test.ts` validates the fixtures and pins
  TypeScript legacy behaviour.

## Future work

A later change will add the manifest-v4 writer and declaration-only reader
dispatch described in the specification. Production hashing and verification
stay on the legacy profile until then.
