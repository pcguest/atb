# Maintainer notes

Operational references for people shipping ATB releases. Contributor workflow
and invariants live in [CONTRIBUTING.md](../../CONTRIBUTING.md).

| Document | Purpose |
| --- | --- |
| [disaster-recovery.md](./disaster-recovery.md) | Quarterly source, secret, and release-pipeline recovery |
| [automatic-audit-capture-definition.md](./automatic-audit-capture-definition.md) | Capture-completeness terminology |
| [manual-test-playbook.md](./manual-test-playbook.md) | Local operator commands for ATB, Custos, and UI smoke testing |
| [public-mit-extraction-checklist.md](./public-mit-extraction-checklist.md) | Manual allowlist, exclusion, and validation steps for public ATB extraction |

Release gates and versioning: [release.md](../release.md), [VERSIONING.md](../../VERSIONING.md).

## Latest release proof

`v1.14.5` (2026-06-13) passed Version Gate, tag Gold, Docker Publish, and the
Release workflow. GitHub Release, PyPI `atb-sdk`, npm `@pcguest/atb-sdk`, and
Docker `pcguest/atb` all advanced to `1.14.5`.

Independent verification found that the retained Actions bundle's
pre-publication signature was invalidated when the later npm capture rewrote
the NDJSON serialization. The immutable workflow artifact remains available as
the historical record. The GitHub release carries
`atb-release-evidence-v1.14.5.atb`, rebuilt from the unsigned build checkpoint,
with a publication-verification snapshot and a final signature from the same
release key. The corrected asset verifies successfully with ATB `1.14.5`.

The workflow now preflights the signing key on a temporary copy before
publication and signs the retained evidence only after all capture steps.
