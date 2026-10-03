---
name: release-validation
description: Use before declaring a slice complete or before any remote step. Covers the canonical deterministic validation gates for ATB, web, SDKs, and acquisition. Trigger on: "is it done", final validation, pre-commit, pre-PR, hygiene, golden, demo-incident, web build, SDK tests.
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
5. Python SDK: `pytest sdk/python/tests`
6. TypeScript SDK: typecheck + tests + build + consumer typecheck
7. Web: lint, typecheck, unit tests, production build
8. Acquisition: three-pass fixture, dedupe, changed source, finding idempotence,
   checkpoint restart/mismatch, legacy bundle, mixed retrospective/live, tamper
9. Viewer: Firefox Cypress, strict axe, keyboard, desktop/narrow/small,
   long digest, legacy bundle, mixed acquisition, SOURCE_RECORD_CHANGED

## Stop conditions

- Stop if any gate fails: report the exact failing gate, do not paper over it.
- Stop before push/PR/merge/tag/release/publish (remote boundary).

## Canonical references

- Makefile
- docs/maintainers/local-acceptance.md
- docs/maintainers/release.md
- CONTRIBUTING.md

## Tests / checks

Exactly the commands above. Loopback binding may require elevated local-listener
permission; a sandbox denial is not a product failure, but must be reported as
such rather than as a pass.

## Prohibited interpretations

- A skipped/unavailable gate is NOT a pass.
- "All green" does not prove correctness or security.
