# Contributing

ATB is an open project. Contributions, bug reports, and feedback are
welcome.

## Local setup

```bash
git clone https://github.com/pcguest/atb.git
cd atb
go build ./...
go test ./...
```

For `atb view` or the embedded dashboard, build the web layer first:

```bash
cd web && npm ci && npm run build && cd ..
go build -o atb ./cmd/atb
```

## Development workflow

- Work lands on `main`.
- Run `make hygiene-quick` before every push. It runs `go fmt`,
  `go vet`, short Go tests, and the web lint and typecheck steps.
- Install the pre-commit hook with `make install-hooks`.
- Do not add new Go module dependencies without prior discussion.

## Python SDK

```bash
cd sdk/python
python3 -m venv venv && source venv/bin/activate
pip install -e .[dev]
pytest -v
```

## TypeScript SDK

```bash
cd sdk/typescript
npm ci && npm run typecheck && npm run build
```

## Commit style

Use conventional commits:

```text
feat(scope): description
fix(scope): description
docs(scope): description
chore(scope): description
test(scope): description
```

## Pull requests

Open a short-lived branch or fork and submit against `main`. Include:

- a concise summary of the change
- `go test ./...` and `make hygiene-quick` output
- any follow-up work intentionally left out

## Schema changes

Changes to `schemas/event.v1.json` that alter the canonical hash input
require a CHANGELOG entry noting that bundles written before the change
will not re-verify against the new implementation. Additive optional
fields do not require this notice.

## Release process

1. Run `./scripts/release-check.sh`
2. Confirm version parity across `cmd/atb/main.go`,
   `sdk/python/pyproject.toml`, `sdk/python/atb/__init__.py`,
   `sdk/typescript/package.json`, `sdk/typescript/package-lock.json`,
   `web/package.json`, `web/package-lock.json`, `README.md`, and any
   release metadata that carries the current version string
3. Tag and push with
   `git tag -a vX.Y.Z -m "Release vX.Y.Z" && git push origin vX.Y.Z`
4. Monitor the `Release` workflow in GitHub Actions

## Security

See [SECURITY.md](SECURITY.md) for the vulnerability disclosure
policy.
