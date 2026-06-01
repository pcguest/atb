# Public surface and private core

This repository is the **private source-of-truth** implementation for ATB. A
separate public demo or product repository may be generated from an allowlisted
subset of this tree. The export pipeline lives under `scripts/export-public-demo.sh`.

## Private core (do not export)

These areas contain the full obligation evaluator, CAS scoring, capture mapping
logic, KMS integrations, and operational harness material:

- `internal/` — bundle engine, verify, profiles, trust, capture, corroboration, push, sign
- `cmd/atb/` — full CLI implementation
- `AGENTS.md` — maintainer and agent harness
- `.github/workflows/release.yml`, `scripts/quality-evidence.sh`, and other release internals
- Adversarial and fuzz corpora under `internal/bundle/`

## Safe to expose publicly

- `schemas/event.v1.json` and trimmed specification docs
- `pkg/otel/` and `pkg/corroborate/` (`github`, `langchain`) as opt-in integration adapters
- `sdk/python/` and `sdk/typescript/` (integration surface)
- `examples/` and generated sample bundles
- `test/golden/` cross-language parity fixtures
- `web/` viewer and landing (with copy aligned to public positioning)
- Conceptual docs: `docs/why-atb.md`, `docs/security.md`, `docs/provability-ladder.md`, quickstart guides

## atb view server

### Session index endpoints

`GET /api/v1/sessions`

Returns a JSON array of `SessionEntry` objects for all bundles in the session
index. Each entry includes: `session_id`, `actor.display_name`, `actor.email`,
`started_at`, `closed_at`, `exchange_count`, `inferred_profile`, `cas_grade`,
`anomaly_flags`, and `bundle_path`. Requires session token auth when
configured.

`GET /api/v1/sessions/by-actor`

Returns a JSON object mapping actor display name to an array of `SessionEntry`
objects. Same auth requirements as above.

### Contract status endpoint

`GET /api/v1/schema/status`

Returns ecosystem-level contract health for the loaded bundle: `schema_source`,
`declared_types`, `observed_types`, `total_events`, `incomplete_events`,
`undeclared_types`, and a per-type `types` array (`type`, `criticality`,
`declared`, `observed`, `required_fields`, `incomplete`, `missing_fields`). It
scores every event type against the schema-generated contract so operators can
see declared-vs-observed coverage, required-field completeness, and any
undeclared (unknown) types at a glance. Same session-token auth and
verification gate as the session index endpoints.

ATB surfaces aggregated session metadata for audit and oversight purposes; it
records and verifies evidence but does not certify legal or regulatory
compliance.

## atb intercept

### Body capture (privacy default)

By default the capture proxy records only a SHA-256 digest (`body_sha256`) and
byte length (`body_bytes`) of each request and response body — never the raw
prompt, completion, or any PII it contains. Credential and session-secret
headers (`Authorization`, `X-Api-Key`, `Proxy-Authorization`, `Cookie`) are
always stripped before recording. Pass `--capture-bodies` to retain raw bodies
in the bundle where that tradeoff is acceptable.

### --custos flag

`--custos <url>`

When set, automatically POSTs the closed bundle to the configured Custos ingest
endpoint on each `atb.session.close` event, using an immutable byte snapshot
taken under the recorder lock. The endpoint URL is printed on startup.
Auto-push is disabled when the flag is not set.

## atb incident

`atb incident list --bundle <path> [--format markdown|json]`

Lists the agent sessions captured in a bundle — session id, actor, exchange
count, inferred profile, CAS grade, and anomaly flags — so a reviewer can
discover which session to report on.

`atb incident export --bundle <path> --session <id> --out <pack.zip>`

Writes a self-contained, independently verifiable incident evidence package: the
signed bundle, the incident report (markdown + JSON), the full verifier report,
and a `MANIFEST.json` chain-of-custody record (bundle hash, signature
provenance, capture scope, and a SHA-256 of every file). The packed bundle
re-verifies on its own with `atb verify bundle.atb` — no trust in the package
required.

`atb incident report --bundle <path> --session <id> [--format markdown|json|ndjson]`

`ndjson` emits one JSON object per session event (denormalised with the
session's integrity status and anomaly flags) for direct SIEM ingestion.

Builds a session-scoped forensic report over a captured bundle: integrity
status, **bundle signature provenance** (who signed it, when, and whether the
signature is valid — an unsigned bundle is stated plainly), the session's actor
and anomaly flags (e.g. `tool_without_approval`), and an ordered list of the
session's events with a one-line summary and the record hash of each.

A bundle's integrity is verified across the whole hash chain, so a single
session cannot be carved into an independently verifiable sub-bundle. The full
signed bundle therefore remains the authoritative evidence; the report scopes
one session for review, and every event row's record hash is checkable against
that bundle.

## Public demo repository notice

When publishing a public tree, include this notice in the public README:

> This repository is the public product and demo surface for ATB. It includes
> specifications, SDKs, sample bundles, and a local viewer sufficient to evaluate
> tamper-evident audit trails for AI workflows. It is not the complete private
> implementation repository. Advanced obligation evaluation, enterprise
> integrations, and operational tooling may differ from what is published here.

## Before each export

1. Run `scripts/export-public-demo.sh --dry-run`
2. Review `EXPORT_REVIEW.md` for unexpected paths
3. Confirm secret scan passed (no PEM blocks, API keys, or private bundles)
4. Set `APPROVED_BY=<reviewer>` before any push to the public remote

## Public demo repository shape

The generated public tree (for example `atb-demo`) is intentionally narrow:

| Included | Purpose |
|----------|---------|
| `schemas/` | Event schema authority |
| Trimmed `docs/` | Specification, provability ladder, quickstart (no maintainer harness) |
| `sdk/python/`, `sdk/typescript/` | Integration surface with golden parity CI |
| `examples/` + sample bundles | Runnable demos including valid/tampered pairs |
| `test/golden/` | Cross-language hash regression contract |
| `web/` | Local viewer (mock API default when verifier binary absent) |
| Public README | Repository notice (see above) |

| Excluded | Reason |
|----------|--------|
| `internal/`, `cmd/atb/` | Full obligation evaluator and CLI source |
| Release/adversarial CI | Operational internals |
| Private keys and root `*.atb` | Secret and fixture hygiene |

The verifier ships as a **release binary artefact** in the public repo; obligation
evaluation logic is not re-exported as Go source.

## custosd HTTP API

`custosd` is the ATB custody daemon. It receives, verifies, stores, and
receipts ATB bundles for long-term custody and independent re-verification. It
records custody evidence; it does not certify regulatory compliance.

`POST /ingest`

Accepts a raw ATB bundle (`.atb` file) as the request body. Verifies integrity
via the public custody verifier. On success, stores the bundle via `WORMStore`
(atomic, idempotent by SHA-256 content hash) and issues a `Receipt`. Returns
the `Receipt` as JSON. Returns 422 for empty or invalid bundles. Returns 500
if required stores are not initialised.

`GET /receipts/:id`

Returns the stored `Receipt` JSON for the given receipt ID. Returns 404 with
`ErrReceiptNotFound` if the ID is unknown.

`GET /receipts/:id/verify`

Re-runs verification on the stored bundle bytes and returns a fresh
`verify.report.v1` JSON object. Use this endpoint to confirm a previously
ingested bundle still passes integrity checks at any point after ingest.

## Corroboration

Corroboration adapters in `pkg/corroborate` construct query URLs for
independent verification of ATB events against external audit sources. All
adapters share the `CorrobResult` type defined in
`pkg/corroborate/corroborate.go`.

`CorrobResult.QueryURL`

Present on all `CorrobResult` values. Contains the fully-constructed URL for
the external audit source query. ATB constructs this URL but does not make the
external HTTP request and does not certify the result. Intended for manual
review or integration with automated verification pipelines.

`LangChainCorroborator` (`pkg/corroborate/langchain`)

Constructs LangSmith run query URLs for `atb.tool.call`, `atb.data.export`,
and `atb.retrieval.*` events. Uses `url.Values.Encode()` for safe parameter
encoding. Live HTTP lookup is deferred; `QueryURL` is returned for external
use.

`GitHubCorroborator` (`pkg/corroborate/github`)

Constructs GitHub Audit Log API query URLs for `atb.tool.call`,
`atb.data.export`, `atb.policy.decision`, and `atb.human.override` events.
Prefers actor email over display name as the phrase filter. Live HTTP lookup
is deferred; `QueryURL` is returned for external use.
