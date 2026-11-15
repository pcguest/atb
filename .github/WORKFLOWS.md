# GitHub workflows

| Workflow | Purpose |
| --- | --- |
| `ci.yml` | Cross-platform build, tests, SDK parity, version/support checks, docs, and smoke gates |
| `security.yml` | History, source, dependency, filesystem, and image security scans |
| `codeql.yml` | GitHub CodeQL analysis |
| `release.yml` | Validate, build, verify, gate, and publish non-Docker release artefacts |
| `docker-publish.yml` | Build multi-architecture images without credentials; publish only in the protected `docker-release` environment |
| `ops.yml` | Weekly registry health and lightweight repository triage |

The repository-local composite action under `.github/actions/go-module`
exposes the checkout as a local Go module to downstream integration jobs.

A tag push triggers `release.yml` only. `docker-publish.yml` has no tag trigger;
it publishes only when dispatched manually with `confirm=PUBLISH`, a stable
SemVer tag, and that tag's commit SHA.

Environment protection is a deliberate-action control, not an authorization
boundary. Requiring reviewers on the `release` and `docker-release`
environments makes an approval a deliberate act; it does not stop a repository
writer from changing these workflows or the environment settings. Recommended
hardening, not yet verified: move `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`,
`NPM_TOKEN`, and `ATB_SIGNING_KEY_PEM` to environment-scoped secrets on those
environments, and prevent self-review of environment approvals.

Actions are pinned to full commit SHAs. Update a pin by reviewing the upstream
release and changing the `uses:` line in a focused pull request.
