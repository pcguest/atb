# Contributing to ATB

Thanks for helping improve ATB.

## Development Setup

```bash
git clone https://github.com/pcguest/atb.git
cd atb
```

For a quick CLI smoke-check that matches README install guidance:

```bash
pip install atb
atb init
atb view --no-open
```

## Local Validation

Run all core checks before opening a PR:

```bash
go test ./...

cd sdk/python
python3 -m venv venv
source venv/bin/activate
pip install -e .[dev]
pytest -v

cd ../typescript
npm ci
npm run typecheck
npm run test
npm run build
```

## Dashboard Build

```bash
cd web
npm ci
npm run build
```

## Commit and PR Standards

- Use conventional commit messages (`feat:`, `fix:`, `docs:`, `chore:`, `test:`)
- Keep changes scoped and reviewable
- Include test evidence in PR description
- Update docs for behavior or contract changes

## Release Preflight

Before creating a release tag, run:

```bash
./scripts/release-check.sh
```

This validates lockfiles, tests, package builds, and release prerequisites.

## Security

- Never commit secrets
- Report vulnerabilities through the process in [SECURITY.md](../SECURITY.md)

Run Trivy checks for security-sensitive changes:

```bash
# Filesystem scan (HIGH/CRITICAL)
trivy fs --scanners vuln --severity HIGH,CRITICAL .

# Image scan when Docker/runtime layers change
docker build -t atb:security-scan .
trivy image --scanners vuln --severity HIGH,CRITICAL atb:security-scan
```
