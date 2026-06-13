# Maintainer notes

Operational references for people shipping ATB releases. Contributor workflow
and invariants live in [CONTRIBUTING.md](../../CONTRIBUTING.md).

| Document | Purpose |
| --- | --- |
| [disaster-recovery.md](./disaster-recovery.md) | Quarterly source, secret, and release-pipeline recovery |
| [automatic-audit-capture-definition.md](./automatic-audit-capture-definition.md) | Capture-completeness terminology |

Release gates and versioning: [release.md](../release.md), [VERSIONING.md](../../VERSIONING.md).

## Latest release proof

`v1.14.4` (2026-06-13) passed Version Gate, tag Gold, and Docker Publish.
GitHub Release, PyPI `atb-sdk`, npm `@pcguest/atb-sdk`, and Docker
`pcguest/atb` all advanced to `1.14.4`. The Release workflow published the
registries but failed its post-publication evidence-signing step because
`ATB_SIGNING_KEY_PEM` was absent; the secret is now configured and the workflow
orders signing before publication with retry-safe npm handling.
