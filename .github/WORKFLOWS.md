# GitHub workflows

| Workflow | Purpose |
| --- | --- |
| `ci.yml` | Cross-platform build, tests, SDK parity, version/support checks, docs, and smoke gates |
| `security.yml` | History, source, dependency, filesystem, and image security scans |
| `codeql.yml` | GitHub CodeQL analysis |
| `release.yml` | Validate, build, verify, gate, and publish non-Docker release artefacts |
| `docker-publish.yml` | Build and publish signed-by-digest multi-architecture images |
| `ops.yml` | Weekly registry health and lightweight repository triage |

The repository-local composite action under `.github/actions/go-module`
exposes the checkout as a local Go module to downstream integration jobs.

Actions are pinned to full commit SHAs. Update a pin by reviewing the upstream
release and changing the `uses:` line in a focused pull request.
