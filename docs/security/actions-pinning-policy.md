# GitHub Actions Pinning Policy (v1.1.0+)

## Policy
All GitHub Actions `uses:` directives MUST be pinned to full commit SHA, not version tags.

## Rationale
- Version tags such as `@v4` can move or be force-updated.
- Full commit SHAs are immutable and auditable.
- This aligns with supply-chain hardening and change-management controls.

## Process
1. When adding a new action, identify the upstream tag or branch being adopted.
2. Resolve that ref to a full commit SHA.
3. Pin the workflow entry as `uses: owner/repo@SHA # original-ref`.
4. Add the mapping to [pin-actions.sh](/Users/paddyguest/atb/scripts/pin-actions.sh) so future bulk refreshes stay consistent.

## Updating Pins
Quarterly, or when intentionally upgrading an action:
1. Resolve the new upstream ref to a full SHA.
2. Update [pin-actions.sh](/Users/paddyguest/atb/scripts/pin-actions.sh).
3. Re-run the script across `.github/workflows`.
4. Validate that no `@v*` refs remain and all workflow actions are SHA-pinned.

## Exceptions
None. All workflow actions must be SHA-pinned.
