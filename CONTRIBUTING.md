# Contributing

ATB is an open project. Contributions, bug reports, and focused documentation improvements are welcome.

This file covers contributor workflow. Release preparation and versioning details live in
`[docs/release.md](docs/release.md)` and `[VERSIONING.md](VERSIONING.md)`.

## Local setup

```bash
git clone https://github.com/pcguest/atb.git
cd atb
make hygiene-quick
```

For `atb view` or the embedded review UI, build the web layer first:

```bash
cd web
npm ci
npm run build
cd ..
go build -o atb ./cmd/atb
```

## Development workflow

- Use a short-lived branch or fork and submit against `main`.
- Do not push directly to `main` routinely.
- Run `make hygiene-quick` before every push. It runs `go fmt`,
`go vet`, short Go tests, and the web lint and typecheck steps.
- Install the pre-commit hook with `make install-hooks`.
- Do not add new Go module dependencies without prior discussion.

## Documentation expectations

- When behaviour, schema, API, or release-facing text changes, update the relevant docs in the same change.
- Do not leave stale references to removed flags, UI modes, or deleted files.
- Tracked follow-on work belongs in `[docs/roadmap.md](docs/roadmap.md)`; do not describe unshipped capability as current in user-facing docs.

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

If the change touches viewer routes or DTOs, update `docs/spec-dashboard.md` and `docs/api/openapi.yaml` as part of the same patch.

### Local release gates

If GitHub Actions is unavailable (for example, a billing or spending-limit
block), run the release gates locally before merging to `main`:

```sh
GOCACHE=$(pwd)/.gocache/release GOTOOLCHAIN=go1.26.4 make test-golden
GOCACHE=$(pwd)/.gocache/release GOTOOLCHAIN=go1.26.4 go test ./... -count=1
bash scripts/check-versions.sh
```

These mirror the `release-gate.yml` workflow; all three must pass.

## Schema changes

Changes to `schemas/event.v1.json` that alter the canonical hash input
require a CHANGELOG entry noting that bundles written before the change
will not re-verify against the new implementation. Additive optional
fields do not require this notice.

## Release process

The maintainer release sequence is documented in [docs/release.md](docs/release.md). Two scripts are central:

- `scripts/release-check.sh` — full preflight suite: Go tests,
TypeScript build and tests, Python tests and package build, web
dashboard build, installed binary smoke gate, and Docker smoke build.
Run this before tagging.
- `scripts/check-versions.sh` — targeted version-string agreement check
against the release tag across the checked version locations. Run with
`ATB_SKIP_TAG_CHECK=1` on a feature branch before the tag exists; run
without the flag after tagging to confirm the tag matches. This check
also runs in CI via the `version-gate.yml` workflow on every push and
pull request.

The release tag is the release source of truth. The checked-in version strings must match it.

## Security

See [SECURITY.md](SECURITY.md) for the vulnerability disclosure
policy.

## Maintainer rules

These invariants apply to every change that touches the bundle format, verifier,
or public CLI contract. Full versioning policy is in [VERSIONING.md](VERSIONING.md).

### Product boundary

ATB is a local-first tamper-evident audit trail. It proves integrity of what was
recorded; it does not prove capture completeness, model correctness, actor
identity, or regulatory compliance by itself. Hosted custody, auditor access,
retention policy, and custodian-of-record work belong in
[Mortise](https://github.com/pcguest/mortise) or another external product.

### Core invariants

1. Bundles are append-only NDJSON; existing records are never edited.
2. Record hash: `SHA-256(UTF-8(hex(prev_hash)) || RFC8785(event))`.
3. Genesis sentinel: 64 zero hex characters.
4. Canonical hash input is frozen unless a deliberate manifest-version migration is performed.
5. Default writer manifest version is `1` unless VERSIONING.md says otherwise.
6. Local-first by default; network activity ties to explicit commands or configured backends.
7. `LoadVerified` is required for integrity-sensitive reads.
8. ATB Agent is optional and loopback-only.

### Release gates

Before a public release:

```bash
GOCACHE=$(pwd)/.gocache/release GOTOOLCHAIN=go1.26.4 make test-golden
GOCACHE=$(pwd)/.gocache/release GOTOOLCHAIN=go1.26.4 go test ./... -count=1
/bin/bash scripts/check-versions.sh
```

Broader preflight when the environment can build web artefacts:

```bash
/bin/bash scripts/release-check.sh
```

Go is canonical. Python and TypeScript must match Go byte-for-byte via
`make test-golden` before release.

### Files requiring extra review

- `internal/bundle/`, `internal/hash/`, `internal/canonicalize/`
- `schemas/event.v1.json`, `docs/spec-v1.0.md`, `docs/spec-ai-traces.md`
- `internal/verify/report.go`, `pkg/custody/schema/`
- `cmd/atb/exit_codes.go`, public CLI help, documented JSON output
- `sdk/python/`, `sdk/typescript/`
- `LICENSE`, `SECURITY.md`, `VERSIONING.md`

Schema or canonicalisation changes require docs, tests, and CHANGELOG entries in
the same release-preparation pass. Do not describe ATB as certifying compliance
or replacing a custodian.