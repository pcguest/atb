# ATB repository guidance

ATB is the public, open-source foundation for portable AI-agent evidence. ATB must not depend on private Mortise or Tenon implementation; those products may depend on ATB.

## Core contracts

- Bundles are append-only NDJSON. Record hashes use `SHA-256(UTF-8(hex(prev_hash)) || RFC8785(event))`; the genesis hash is 64 zero hex characters.
- ATB proves integrity of recorded data, not capture completeness, truth, identity, or correct behaviour.
- Preserve public wire identifiers and compatibility aliases unless a versioned migration authorises a change. Schema changes need a CHANGELOG entry; canonicalisation changes need a migration.
- Keep optional integrations optional. Verification must not depend on hosted or private services. Never commit credentials or private URLs.

## Development

- Run `make hygiene-quick`, `make test-golden`, and `go test ./...` for every change. Run `make build` first when checks require generated viewer output.
- See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, workflow, and documentation rules; see [docs/README.md](docs/README.md) for the reading route.
- Personal agent configuration, model choices, experiments, and operating records belong outside the public tree.

## Repository map

- `cmd/atb/`: CLI; `internal/`: implementation; `pkg/`: public Go packages (`pkg/api/v1` is the local viewer HTTP API, not a product SDK).
- `sdk/`: Python and TypeScript SDKs; `web/`: embedded viewer; `docs/`: canonical documentation; `test/`: tests and golden vectors.
- Generated content includes `docs_embed.go`, `trust_embed.go`, profile fixtures, and `web/out/`; see the relevant build scripts before editing generated files.
