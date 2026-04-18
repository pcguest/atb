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

Two scripts are involved:

- `scripts/release-check.sh` — full preflight suite: Go tests,
  TypeScript build and tests, Python tests and package build, web
  dashboard build, installed binary smoke gate, and Docker smoke build.
  Run this before tagging.
- `scripts/check-versions.sh` — targeted version-string agreement check
  across the seven canonical locations (see below). Run with
  `ATB_SKIP_TAG_CHECK=1` on a feature branch before the tag exists; run
  without the flag after tagging to confirm the tag matches. This check
  also runs in CI via the `version-gate.yml` workflow on every push and
  pull request.

Steps:

1. Bump the version constant in `cmd/atb/main.go` (source of truth).
2. Update the remaining six locations to match:
   - `sdk/python/pyproject.toml`
   - `sdk/python/atb/__init__.py`
   - `sdk/typescript/package.json`
   - `sdk/typescript/package-lock.json` (both the root `version` field
     and the `packages[""].version` field)
   - `web/package.json`
3. Run `ATB_SKIP_TAG_CHECK=1 bash scripts/check-versions.sh` — must
   print `ok: all version strings agree`.
4. Run `./scripts/release-check.sh` — all seven steps must pass.
5. Tag and push:
   `git tag -a vX.Y.Z -m "Release vX.Y.Z" && git push origin vX.Y.Z`
6. Monitor the `Release` workflow in GitHub Actions.

## Commit history

The repository commit history includes entries from an AI-assisted
development period. Some commit messages from before v1.5.0 may contain
co-authorship attributions, session labels, or phrasing that does not
follow the current style guide. New contributions must follow the commit
style rules above. Historical commits are not being rewritten as they are
already on the remote.

## Security

See [SECURITY.md](SECURITY.md) for the vulnerability disclosure
policy.
