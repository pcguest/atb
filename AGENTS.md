# ATB agent notes

This repository is local-first. The Go CLI under `cmd/atb` is the
authoritative implementation for bundle creation, verification, export,
and the local viewer. The Python and TypeScript packages are SDKs that
write the same bundle format.

## Working assumptions

- Treat `cmd/atb/main.go` as the CLI source of truth for command names, flags, and version output.
- Treat `internal/verify/` and the YAML templates under `internal/profiles/` as the source of truth for profile behaviour and CAS.
- Treat `docs/security.md`, `docs/key-management.md`, and `docs/compliance/` as normative project guidance for security and audit posture.
- Build the web assets before validating `atb view` or the embedded dashboard path.

## Validation expectations

- Run `go test ./...` after Go changes.
- Run the Python and TypeScript SDK tests when SDK-facing behaviour or documentation changes.
- Keep version strings aligned across the CLI, SDK manifests, web package, and README release markers.
- Do not add undocumented CLI behaviour or leave stale workflow examples in the docs.
