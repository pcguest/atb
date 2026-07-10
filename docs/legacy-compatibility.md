# Legacy bundle compatibility

ATB preserves current `.atb` bundle evidence by verifying historical formats
without silently rewriting them. When a bundle cannot be verified by the current
reader, ATB must fail loudly and leave the source bytes untouched.

## Current compatibility contract

- Manifest v1 remains the default writer format.
- Manifest v2 is readable and opt-in for writers.
- Bundles declaring a manifest version newer than this build supports are
  refused with an error that includes `unsupported manifest version`.
- Legacy pre-v1.0 bundles without a manifest record may still be parsed by low
  level readers, but `LoadVerified` and `atb verify` require a manifest record
  for a current evidence claim.
- The historical v1.1.2 float canonicalisation break is documented in
  [VERSIONING.md](../VERSIONING.md). Affected bundles should be verified with
  the original release and re-exported; the current release must not silently
  reinterpret their hash chain.

## Refusal policy

Unsupported, malformed, truncated, or non-bundle inputs are not migrated in
place. Current readers should return a typed error class where possible:

| Case | Expected class |
| --- | --- |
| Empty bundle file | `ErrNoManifest` |
| NDJSON that is not an ATB bundle | `ErrNotABundle` |
| Broken hash chain | `ErrTamper` |
| Unsupported manifest version or malformed JSON | `ErrMalformed` |

This refusal mode is intentional. Rewriting source evidence would create a new
hash chain and could make later incident review ambiguous about which bytes were
originally collected.

## Migration guidance

There is no `atb migrate` command today. Until one exists:

1. Preserve the original `.atb` file exactly as collected.
2. Verify it with the ATB release that originally wrote it, if available.
3. Save that verifier output as migration evidence.
4. Export events from the original release and import them with the current
   release to create a new bundle.
5. Retain both the original bundle and the new bundle. The new bundle does not
   retroactively certify the original hash chain.

A future migration command should produce a separate migration evidence record
that names the source bundle hash, source head hash, source ATB version, new
bundle hash, new head hash, and the operator who performed the migration. It
must not overwrite the original bundle.

## Test evidence

The current refusal and no-rewrite behavior is covered by
`internal/bundle.TestLegacyCompatibilityRefusesWithoutRewriting`. Scheduled
parser and canonicalizer fuzzing runs in `.github/workflows/nightly-fuzz.yml`;
pull requests keep only bounded unit and integration checks on the fast path.
