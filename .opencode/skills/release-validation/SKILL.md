---
name: release-validation
description: "Use before declaring a slice complete or before any remote step. Covers the canonical deterministic validation gates for ATB, web, SDKs, and acquisition. Trigger on: \"is it done\", final validation, pre-commit, pre-PR, hygiene, golden, demo-incident, web build, SDK tests."
---

# Release Validation

## Purpose

Run the deterministic gates that are authoritative for ATB. Model opinions never
override these results.

## When to invoke

- Before committing a completed slice locally
- Before any remote step (which remains blocked unless explicitly instructed)
- After a material change that touches Go, web, SDKs, or acquisition

## Bounded procedure (run in order; record raw results)

1. `git diff --check`
2. `make hygiene-quick` (already ends with `make test-go`)
3. `make test-golden`
4. `make demo-incident`
5. `make build` (the harness needs the `./atb` binary), then end-user harness: `harnesses/end-user/run.sh --out .local/dev/atb-enduser.report.json` (or pass a built binary via `--atb`/`ATB_BIN`)
6. Python SDK: `.venv/bin/python -m pytest sdk/python/tests`
7. TypeScript SDK: `npm --prefix sdk/typescript run typecheck && npm --prefix sdk/typescript test && npm --prefix sdk/typescript run build`
8. Web: `npm --prefix web run lint && npm --prefix web run typecheck && npm --prefix web test && npm --prefix web run build`
9. Acquisition: `go test ./internal/acquisition/...`
10. Viewer: `make test-e2e` and `npm --prefix web run test:a11y`

## Stop conditions

- Stop if any gate fails: report the exact failing gate, do not paper over it.
- Stop before push/PR/merge/tag/release/publish (remote boundary).

## Canonical references

- Makefile
- docs/maintainers/local-acceptance.md
- docs/maintainers/release.md
- CONTRIBUTING.md

## Tests / checks

Run the exact invocations listed in the steps above; each numbered item is a
literal command. Loopback binding may require elevated local-listener
permission; a sandbox denial is not a product failure, but must be reported as
such rather than as a pass.

## Prohibited interpretations

- A skipped/unavailable gate is NOT a pass.
- "All green" does not prove correctness or security.
