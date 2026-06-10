# ATB GitHub Workflows

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| ci.yml | push/PR to main | Test matrix + golden parity + docs smoke |
| security.yml | push/PR to main, schedule, manual | gosec, bandit, npm audit, and scheduled Trivy security gates |
| release.yml | tag push | Build CLI + publish to PyPI/npm + GitHub Release |
| docker-publish.yml | tag push, manual | Build and publish Docker image |
| gold-release.yml | tag push | Gold release validation gate |
| release-gates.yml | push/PR to main, manual | Exact cross-language golden-vector and full Go release gates |
| ops.yml | schedule/push/PR/issues | Docs deploy, registry health, feedback digest, labeling (conditional) |
| version-gate.yml | push/PR on version files | Cross-file version parity check via `check-versions.sh` |

The private composite action at `.github/actions/go-module` exposes this checkout
as a local Go module to other private repositories owned by `pcguest`. It exists
so downstream CI can test the pinned ATB contract without storing a
cross-repository access token.

## Adding a New Job

1. Pick the workflow that matches the job's purpose.
2. Add conditional `if:` to avoid unnecessary runs.
3. Test locally with `act` or trigger via `gh workflow run`.
4. Update this file.
