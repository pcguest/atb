# JCS canonicalisation-profile fixtures

**Status: specification-only.** The fixtures in this directory describe two
canonicalisation profiles but do not switch any production behaviour. The
normative specification is
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
  produced by any ATB canonicaliser. They are the independent oracle for a
  future strict implementation.
- **Negative cases are not executable against a strict parser in this
  increment** because no strict parser exists. Tests validate only their
  schema, category coverage, and declared error class.
- **Legacy expectations are frozen** from the implementations at `a7837e2`.
  They must not be regenerated. A change to any legacy expectation is a
  breaking canonicalisation change and requires the manifest-version and
  migration process in `VERSIONING.md`.
- The strict and legacy fixtures must keep their existing contents. Tests in
  Go, Python, and TypeScript load and validate them read-only.

## Consumers

- `test/jcs/vectors_test.go` validates all four files and pins the Go legacy
  behaviour.
- `internal/bundle/canonical_profile_test.go` pins the current reader's
  fail-closed handling of v3/v4 profile-selection cases.
- `sdk/python/tests/test_jcs_spec_fixtures.py` validates the fixtures and pins
  Python legacy behaviour.
- `sdk/typescript/src/jcs-spec-fixtures.test.ts` validates the fixtures and pins
  TypeScript legacy behaviour.

## Future work

A later change will add a strict canonicaliser implementation and a conformance
runner that executes the positive and negative vectors against it, together
with the manifest-v4 writer and declaration-only dispatch described in the
specification. That change is explicitly out of scope here.
