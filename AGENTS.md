# ATB Development Instructions

## Product Identity

ATB is the public/open-source technical foundation for portable AI agent evidence.
Mortise and Tenon are private products built on ATB.

## Public/Private Boundary

- **ATB**: PUBLIC / OPEN SOURCE
- **Mortise**: PRIVATE / PROPRIETARY
- **Tenon**: PRIVATE / PROPRIETARY

ATB must not depend on private Mortise or Tenon implementation.
Private products may depend on ATB.

## Architectural Invariants

- Bundles are append-only NDJSON
- Record hash: `SHA-256(UTF-8(hex(prev_hash)) || RFC8785(event))`
- Genesis sentinel: 64 zero hex characters
- Local-first by default
- ATB proves integrity, not completeness or truth

## Canonical Validation Commands

```bash
make hygiene-quick
make test-go
make test-golden
go test ./...
```

## Security Boundaries

- Never commit secrets or credentials
- Never expose private URLs in public documentation
- Never make ATB verification dependent on private services

## Repository Structure

- `cmd/atb/` - CLI entry point
- `internal/` - Core implementation
- `pkg/` - Public APIs (`pkg/api/v1` is local viewer API, not product SDK)
- `sdk/python/` - Python SDK
- `sdk/typescript/` - TypeScript SDK
- `web/` - Embedded viewer UI
- `docs/` - Canonical documentation
- `test/` - Tests and golden vectors

## Allowed Change Discipline

- Schema changes require CHANGELOG entry
- Canonicalisation changes require version migration
- Breaking changes require major version bump
- All changes must pass `hygiene-quick` and `test-golden`

## Generated File Policy

- Embedded docs: `docs_embed.go`, `trust_embed.go`
- Generated fixtures: `examples/bundles/profiles/`
- Web build: `web/out/` (not committed)
