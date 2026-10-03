---
name: canonical-parity
description: Use when changing canonicalisation, hashing, event schema, or generated event types across Go/Python/TypeScript. Trigger on: RFC8785, canonical JSON, record hash, prev_hash, genesis sentinel, schemas/event.v1.json, *_generated.*, cross-language vectors, breaking schema change, version migration.
---

# Canonical Parity

## Purpose

Guarantee that the same event hashes identically across Go, Python, and
TypeScript, and that schema/generated artefacts stay consistent.

## Invariants (do not weaken)

- Record hash = SHA-256(UTF-8(hex(prev_hash)) || RFC8785(event))
- Genesis sentinel = 64 zero hex characters
- Bundles are append-only NDJSON
- Schema changes require a CHANGELOG entry
- Canonicalisation changes require version migration
- Breaking changes require a major version bump

## When to invoke

- Editing `schemas/event.v1.json` or `schemas/event.v1.sha256`
- Editing `internal/event/types*.go`, `internal/canonicalize/*`, `internal/hash/*`
- Regenerating `sdk/python/atb/event_types_generated.py` or
  `sdk/typescript/src/eventTypes_generated.ts`
- Adding/removing an event type or required field

## Bounded procedure

1. Change the source of truth, then regenerate — never hand-edit generated files.
2. Keep Go/Python/TS event type tables in lockstep.
3. Recompute `schemas/event.v1.sha256`.
4. Run golden vectors for all three languages.
5. Add/adjust a golden vector if semantics changed; never loosen an existing one.

## Stop conditions

- Stop if a generated file was hand-edited.
- Stop if any language disagrees on a vector.
- Stop if a canonical change is made without a migration/version note.

## Canonical references

- docs/specification/events.md
- docs/specification/bundle-v1.md
- VERSIONING.md
- Makefile targets: `test-golden`, `check-generated`, `check-versions`

## Tests / checks

- `make test-golden`
- `make hygiene-quick` (check-generated, check-versions, check-notices)
- `go test ./...`

## Prohibited interpretations

- A passing golden gate proves representation parity, not semantic correctness.
- Do not "fix" a vector to match broken behaviour.
