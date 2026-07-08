# Public MIT extraction checklist

This checklist is for creating a polished public ATB repository from the private
trunk while keeping proprietary Mortise product material private.

## Include in public ATB

- Core CLI and runtime: `cmd/atb/`, `internal/`, `pkg/`, `schemas/`, `tools/`, `scripts/`.
- SDKs and compatibility tests: `sdk/python/`, `sdk/typescript/`, `test/`, `examples/`.
- Local viewer/site: `web/` and `docs/` that describe ATB local-first evidence workflows.
- Release and maintenance material that applies to ATB itself: `Makefile`, `go.mod`, `go.sum`, `VERSIONING.md`, `SECURITY.md`, `CONTRIBUTING.md`, `LICENSE`, `.github/workflows/ci.yml`, `.github/workflows/security.yml`, release gates, and public docs.

## Exclude from public ATB

- Proprietary Mortise product implementation, roadmap, customer/evaluator material, hosted custody operations, billing, SSO, legal-hold, managed witness, and private scoring/evaluation logic.
- Any Mortise runtime or product implementation. ATB keeps only its internal
  HTTP client, public custody contracts, conformance tests, and boundary docs.
- Local/generated artefacts: `.atb/`, `run.atb/`, `*.atb`, `*.atb.enc`, `coverage.out`, `trivy-report.json`, `web/.next/`, `web/out/`, `node_modules/`, SDK build outputs, caches, and local keys.
- Any real secrets or environment files: `.env*`, `*.pem`, `*.key`, `credentials.json`, cloud credential files, signing keys, and operator tokens.

## Ordered extraction steps

1. Start from a clean private worktree and run the hardening gates: `go test ./...`, `make check-generated`, `make test-golden`, SDK tests, web tests/build, `make test-embed`, and `make test-integration`.
2. Create a throwaway extraction directory outside the private repo, then copy only the public allowlist with `rsync -a --delete` rather than editing the private tree in place.
3. Remove private or generated paths from the extraction directory using the exclude list above; inspect with `find` and `rg` before initializing the public repository.
4. Re-run secret checks in the extraction directory:
   `rg -n "sk-|api[_-]?key|SECRET|TOKEN|PASSWORD|PRIVATE_KEY|AKIA|github_pat_|ghp_" .`.
5. Verify public messaging: README and docs must describe ATB as the MIT local-first evidence core; Mortise must be described only as a separate companion product, not as required infrastructure.
6. Confirm `LICENSE` is MIT, update `NOTICE` or third-party notices if added, and keep package manifests scoped to ATB only.
7. Run the public-tree validation commands: `go test ./...`, `make check-generated`, `make test-golden`, Python SDK tests, TypeScript SDK tests/typecheck, web tests/typecheck/build, and `npm audit --audit-level=high` for web and TypeScript packages.
8. Only after validation, initialize or update the public repository and review `git status --short` for accidental private files before any commit or publication.
