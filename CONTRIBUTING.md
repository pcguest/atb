# Contributing

Thanks for your interest in improving ATB.

Detailed contributor guidance is maintained in [docs/contributing.md](docs/contributing.md).

## Local Setup

```bash
git clone https://github.com/pcguest/atb.git
cd atb
```

## Development Workflow (v1.1.0+)

This project is operating in a **main-only** workflow for the v1.1.0 cycle.

- **All work happens on `main`.** No long-lived feature branches.
- **Use feature flags** (`--ui-experimental`) to hide incomplete features.
- **Every commit must pass** `make hygiene-quick`.
- **Tag releases:** `v1.1.0-rc1` -> testing -> `v1.1.0`.
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
7. **Publish:** Update README, announce on Discord/Twitter

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

Follow [SECURITY.md](SECURITY.md) for vulnerability handling and disclosure process.

Run local security checks before opening high-impact PRs:

```bash
# Filesystem scan (matches security workflow severity gates)
trivy fs --scanners vuln --severity HIGH,CRITICAL .

# Optional image scan if Docker changes are included
docker build -t atb:security-scan .
trivy image --scanners vuln --severity HIGH,CRITICAL atb:security-scan
```

## Commit Style

Use conventional commits:

- `feat(scope): description`
- `fix(scope): description`
- `docs(scope): description`
- `chore(scope): description`
- `test(scope): description`

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
grep -roE 'https?://[^\s\)"`]+' README.md docs/ | grep -v "github.com/pcguest/atb" | sort -u | xargs -n1 curl -sI | grep -E "HTTP|404|500"
```

Portable fallback:

```bash
rg --no-filename -o 'https?://[^\s)"`]+' README.md docs/ | grep -v "github.com/pcguest/atb" | sort -u | while read -r url; do curl -sI "$url" | head -1; done
```

Note: `www.npmjs.com` may return `403` to automated `HEAD` requests; confirm package availability with:

```bash
curl -sI https://registry.npmjs.org/@pcguest/atb-sdk | head -1
```
