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

## Python SDK Release to PyPI

The `Publish Python SDK` workflow runs on git tags matching `v*`.

1. Configure one publishing path:

- Token-based: set repository secret `PYPI_API_TOKEN`.
- Trusted publishing: create a matching publisher in PyPI for
  `pcguest/atb/.github/workflows/pypi.yml` with tag refs.
2. Validate package locally:

```bash
cd sdk/python
python -m pip install --upgrade pip
python -m pip install build twine
python -m build
twine check dist/*
```

3. Create and push an annotated tag (match package version):

```bash
git tag -a v0.1.1 -m "Release v0.1.1"
git push origin v0.1.1
```

4. Confirm publish status in GitHub Actions and verify:

```bash
go install github.com/pcguest/atb/cmd/atb@latest
```

## TypeScript SDK Release to npm

The `Publish TypeScript SDK to npm` workflow also runs on git tags matching `v*`.

1. Set repository secret `NPM_TOKEN` (publish token).
2. Validate package locally:

```bash
cd sdk/typescript
npm ci
npm run typecheck
npm run build
npm pack
```

3. Push a tag that matches `sdk/typescript/package.json` version:

```bash
git tag -a v0.1.1 -m "Release v0.1.1"
git push origin v0.1.1
```

4. Confirm publish status and verify:

```bash
npm install @pcguest/atb-sdk
```

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
