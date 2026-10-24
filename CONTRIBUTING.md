# Contributing

ATB is an open project. Contributions, bug reports, and focused documentation improvements are welcome.

This file covers contributor workflow. Release preparation and versioning details live in
[docs/maintainers/release.md](docs/maintainers/release.md) and
[VERSIONING.md](VERSIONING.md).

For architecture and trust boundaries, see the [architecture](docs/concepts/architecture.md)
and [trust model](docs/concepts/trust-model.md).

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
- Run `make hygiene-quick` before every push. It checks `gofmt`,
`go vet`, short Go tests, and the web lint and typecheck steps.
- Install the pre-commit hook with `make install-hooks`.
- Do not add new Go module dependencies without prior discussion.

## Documentation expectations

- Canonical locations: architecture and product boundaries in [architecture](docs/concepts/architecture.md); user workflow in the quickstart and task guides; wire and API contracts in specifications; contributor and release procedures here and in the maintainer guides.
- Update the relevant canonical documentation with behaviour, schema, API, or release-facing changes; remove superseded explanations and repair their links.
- Keep plans, agent transcripts, and progress reports local. Add public Markdown only when its audience and purpose are not served by an existing document. Clearly label planned capabilities; do not describe them as implemented.
- Tracked follow-on work belongs in [docs/roadmap.md](docs/roadmap.md).

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

If the change touches viewer routes or DTOs, update
`docs/specification/viewer.md` and `docs/api/openapi.yaml` as part of the same
patch.

### Local release gates

If GitHub Actions is unavailable (for example, a billing or spending-limit
block), run the release gates locally before merging to `main`:

```sh
GOCACHE=$(pwd)/.gocache/release GOTOOLCHAIN=go1.26.10 make test-golden
GOCACHE=$(pwd)/.gocache/release GOTOOLCHAIN=go1.26.10 go test ./... -count=1
bash scripts/check-versions.sh
```

These mirror the release-gate checks: golden-vector parity (ci.yml), Go tests, and version agreement (release.yml); all three must pass.

## Schema changes

Changes to `schemas/event.v1.json` that alter the canonical hash input
require a CHANGELOG entry noting that bundles written before the change
will not re-verify against the new implementation. Additive optional
fields do not require this notice.

## Release process

The maintainer release sequence is documented in
[docs/maintainers/release.md](docs/maintainers/release.md). Two scripts are
central:

- `scripts/release-check.sh` — full preflight suite: Go tests,
TypeScript build and tests, Python tests and package build, web
dashboard build, installed binary smoke gate, and Docker smoke build.
Run this before tagging.
- `scripts/check-versions.sh` — targeted version-string agreement check
against the release tag across the checked version locations. Run with
`ATB_SKIP_TAG_CHECK=1` on a feature branch before the tag exists; run
without the flag after tagging to confirm the tag matches. This check
also runs in the Linux CI job on every push and pull request.

The release tag is the release source of truth. The checked-in version strings must match it.

## Security

See [SECURITY.md](SECURITY.md) for the vulnerability disclosure
policy.

## Maintainer rules

These invariants apply to changes that touch the bundle format, verifier, or public CLI contract. Full versioning policy is in [VERSIONING.md](VERSIONING.md).

### Product boundary

See the [canonical architecture](docs/concepts/architecture.md) for ATB’s
purpose, limits, and optional integration boundaries. Keep verification usable
without external or hosted services.

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
GOCACHE=$(pwd)/.gocache/release GOTOOLCHAIN=go1.26.10 make test-golden
GOCACHE=$(pwd)/.gocache/release GOTOOLCHAIN=go1.26.10 go test ./... -count=1
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
- `schemas/event.v1.json`, `docs/specification/bundle-v1.md`,
  `docs/specification/events.md`
- `internal/verify/report.go`, `pkg/custody/schema/`
- `cmd/atb/exit_codes.go`, public CLI help, documented JSON output
- `sdk/python/`, `sdk/typescript/`
- `LICENSE`, `SECURITY.md`, `VERSIONING.md`

Schema or canonicalisation changes require docs, tests, and CHANGELOG entries in
the same release-preparation pass. Do not describe ATB as certifying compliance
or replacing a custodian.

### Compatibility-sensitive public contracts

External users and optional custody integrations depend on these surfaces:

- the bundle specification, event schema, and cross-language hash vectors;
- `pkg/custody`, `pkg/jcs`, SDK APIs, and `verify.report.v1`;
- documented CLI help, exit codes, JSON output, profile IDs, and CAS semantics;
- signatures, anchors, encryption metadata, and custody export envelopes.

Canonicalisation or on-disk changes require the versioning review and
`make test-golden`. Public API removal requires a deprecation or explicit major
version decision; do not silently move a contract into an internal package.
