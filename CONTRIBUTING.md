# Contributing

Thanks for your interest in improving ATB.

Detailed contributor guidance is maintained in [docs/contributing.md](docs/contributing.md).

## Local Setup

```bash
git clone https://github.com/pcguest/atb.git
cd atb
```

## Development Workflow

This project uses a **main-only** workflow.

- **All work happens on `main`.** No long-lived feature branches.
- **Run tests before pushing:** `go test ./...` and `make hygiene-quick`.
- **Use feature flags** (`--ui-experimental`) to hide incomplete features.
- **Every commit must pass** `make hygiene-quick`.
- **No new module dependencies** are accepted without prior discussion.
- **Tag releases:** use `vX.Y.Z-rcN` for release candidates and `vX.Y.Z` for gold releases.
- **Security fixes** must pass security review before merge.

### Quick Start

```bash
git pull origin main
make hygiene-quick
# If pass, commit and push
```

## Pre-Commit Hooks

Install pre-commit hooks to catch issues before committing:

```bash
make install-hooks
```

The pre-commit hook will:
- Check for Go/NPM dependency vulnerabilities (warnings only)
- Run `make hygiene-quick` (blocks on failure)
- Ensure tests pass

To skip hooks (not recommended): `git commit --no-verify`

## Release Process

1. **Tag rc:** `git tag -a v1.1.0-rc1 -m "Release Candidate 1"`
2. **Push tag:** `git push origin --tags`
3. **Create GitHub Release:** Use tag, paste release notes from `docs/releases/`
4. **E2E testing:** 48-hour window for internal testers
5. **Address feedback:** Fix critical issues, tag rc2 if needed
6. **Gold release:** `git tag -a v1.1.0 -m "Gold Release"`
7. **Publish:** Update the GitHub Release notes and notify relevant stakeholders

### Go CLI

```bash
go test ./...
go build -o atb ./cmd/atb
```

### Python SDK

```bash
cd sdk/python
python3 -m venv venv
source venv/bin/activate
pip install -e .[dev]
pytest -v
```

### TypeScript SDK

```bash
cd sdk/typescript
npm ci
npm run typecheck
npm run build
```

## Security and Trivy Scans

Security scanning runs weekly via GitHub Actions. For local vulnerability checks, see [SECURITY.md](SECURITY.md).

## Commit Style

Use conventional commits:

- `feat(scope): description`
- `fix(scope): description`
- `docs(scope): description`
- `chore(scope): description`
- `test(scope): description`

The expected shape is `<type>(<scope>): <subject>`.

## Pull Requests

If you are contributing from outside the main checkout, open a short-lived
branch or fork and submit a pull request against `main`.

Include:

- a concise summary of the change
- test output (`go test ./...` and `make hygiene-quick`)
- any follow-up work that is intentionally left out

## Schema changes

Changes to `schemas/event.v1.json` that alter the canonical hash input require a CHANGELOG entry
noting that bundles written before the change will not re-verify against the new implementation.
Additive optional fields do not require this notice.

## Release Publishing

The unified release workflow is [`release.yml`](.github/workflows/release.yml). It validates version parity across the repo, builds CLI and SDK artefacts, smoke-tests them, then publishes GitHub Releases, PyPI, npm, and Docker images.

Before tagging a release:

1. Run `./scripts/release-check.sh`
2. Confirm `sdk/python/pyproject.toml` and `sdk/typescript/package.json` match the intended release version
3. Create and push an annotated tag that matches that version, for example:

```bash
git tag -a v1.1.0 -m "Release v1.1.0"
git push origin v1.1.0
```

4. Monitor the `Release` workflow in GitHub Actions and verify the published artefacts

## Link Verification

To verify external links:

```bash
rg --no-filename -o 'https?://[^\s)"`]+' README.md docs/ | grep -v "github.com/pcguest/atb" | sort -u | while read -r url; do curl -sI "$url" | head -1; done
```

Note: `www.npmjs.com` may return `403` to automated `HEAD` requests; confirm package availability with:

```bash
curl -sI https://registry.npmjs.org/@pcguest/atb-sdk | head -1
```
