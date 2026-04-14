# ATB — agent orientation

I built ATB to solve one specific problem: most AI systems produce logs, but logs are not proof. ATB records workflow events into locally stored, SHA-256 hash-chained bundles. If anything is modified, reordered, or removed after capture, `atb verify` detects it. The file is portable — you can verify it on any machine, without a server, without network access.

This is the orientation file for tooling and agents working in this repo.

## What ATB is — and what it is not

ATB is a local-first evidence bundler. It is not an observability tool, a hosted tracing backend, or a governance platform. It does not stream events to a dashboard. Events are written to a portable `.atb` bundle file on disk, under the operator's control.

The comparison that matters: observability tools (Langfuse, LangSmith, OpenTelemetry) give you dashboards for debugging running systems. ATB gives you a portable, cryptographically verifiable artefact for incident review, audit, and customer handoff. These are complementary, not competing.

Distinguishing properties:
- No reverse proxy, no hosted service, no third-party routing of LLM traffic.
- SHA-256 hash chains over RFC 8785 canonical JSON. Conservative and auditable.
- Six schema-locked obligation profiles aligned to concrete workflows.
- Optional RFC 3161 TSA anchoring for third-party time commitment.
- WORM export (v1.6, planned) complements local guarantees; it does not replace them.

## Hard invariants — do not break these

- `go test ./...` passes before every commit.
- British English, no em dashes, in any committed file.
- One logical change per commit, 72-character subject, imperative mood.
- Never edit a test to make it pass — fix the implementation.
- Bundle format is frozen at v1.0 (`docs/spec-v1.0.md`). Do not modify.
- Profile IDs and their required/warning event sets are canonical. Do not change without a major version bump.
- Local-first invariant: no implicit network calls in the core record or verify paths.

## Do not implement without explicit instruction

- `atb push` / WORM S3 export (planned v1.6; spec in `docs/spec/bundle-push.md`)
- ACP gating middleware
- Article 17 GDPR deletion-marker flow
- Web dashboard changes (`atb view --ui-experimental`)

## Key packages

- `cmd/atb/` — CLI entry point and MCP server; source of truth for command names, flags, and version
- `internal/bundle/` — bundle read/write
- `internal/verify/` — chain verification and CAS scoring
- `internal/profiles/` — YAML profile templates; source of truth for profile behaviour
- `internal/anchor/` — RFC 3161 TSA anchoring
- `internal/encrypt/` — AES-256-GCM versioned KDF
- `pkg/api/v1/` — local viewer API (127.0.0.1 only)
- `sdk/python/` — Python SDK
- `sdk/typescript/` — TypeScript SDK

## How to treat WORM/S3 and remote exports

WORM export is an opt-in complement to local bundle integrity. The WORM policy lives in the storage bucket (S3 Object Lock COMPLIANCE mode), not in ATB. ATB requests the lock when `--lock-until` is supplied; the bucket enforces it. Do not describe ATB as "enforcing WORM" or "providing WORM storage" — it uploads a content-addressed bundle and requests the lock header.

Remote export is always explicit and opt-in. It never replaces the local hash chain as the primary integrity primitive.

## How to treat compliance language

ATB maps to control families (EU AI Act Article 12, NIST AI RMF, SOC 2, ISO 42001). It does not certify compliance with any of them. Use "supports", "maps to", or "can be used as evidence for" — not "compliant with" or "satisfies".

Always name the limits explicitly:
- ATB does not prove recording completeness.
- ATB does not prove model correctness or risk controls.
- ATB proves integrity of what was recorded.

## Validation expectations

- Run `go test ./...` after Go changes.
- Run Python and TypeScript SDK tests when SDK-facing behaviour or docs change.
- Keep version strings aligned: CLI constant, `sdk/python/pyproject.toml`, `sdk/python/atb/__init__.py`, `sdk/typescript/package.json`, `web/package.json`. `scripts/check-versions.sh` validates this and gates releases.
- Do not add undocumented CLI behaviour or leave stale workflow examples in docs.

## Release tag messages

Format: `vX.Y.Z — <brief description>`

- British English throughout.
- Describe what the release contains, not how it was produced.
- No references to tooling, AI assistants, or session names.
- Maximum 80 characters including the version prefix.
- En dash (—) as the separator.

Example: `v1.5.1 — repository hygiene, docs consistency, version-parity gate`
