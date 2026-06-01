# Changelog

All notable changes to ATB will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Composite demo bundle `examples/bundles/demo-workflow/` (~20-event support escalation narrative; passes `policy_decision` and `human_override`).
- Profile workflow demo scripts under `examples/demo/` (Python and TypeScript) with CLI verify.
- LangGraph reference integration (`examples/python/langgraph_demo.py`, `docs/integrations/langgraph.md`, optional `langgraph` Python extra).
- `pkg/corroborate`: shared corroboration package defining `Actor`, `Event`, `CorrobResult` (including `QueryURL`), `Corroborator`, and `ErrEventTypeNotSupported`; both adapters import this shared contract.
- `pkg/corroborate/langchain`: `LangChainCorroborator` constructing LangSmith run query URLs for `atb.tool.call`, `atb.data.export`, and `atb.retrieval.*` events using `url.Values`. Live HTTP lookup deferred.
- `pkg/corroborate/github`: `GitHubCorroborator` constructing GitHub Audit Log API query URLs for `atb.tool.call`, `atb.data.export`, `atb.policy.decision`, and `atb.human.override` events. Prefers actor email over display name. Live HTTP lookup deferred.
- `pkg/otel`: OTel inbound transport scaffold with stub `Translate(span OTelSpan) (*Event, error)` interface.
- `internal/sessionindex`: `BuildIndex` and `GroupByActor`, with canonical profile inference and anomaly detection rules per `spec-dashboard.md`.
- View server: `GET /api/v1/sessions` and `GET /api/v1/sessions/by-actor` endpoints returning `SessionEntry` arrays and actor-grouped maps.
- `atb view --sessions` flag: accepts a directory or glob of `.atb` bundles to populate the session index at startup.
- `custos/internal/receipt`: `FileSystemWORMStore` with atomic POSIX write protocol (`tmp` -> `fsync` -> `rename`), idempotent by SHA-256 content hash, and pre-write head-hash integrity check.
- `custos/internal/receipt`: `FileSystemReceiptStore` with JSON save/get/list, typed `ErrReceiptNotFound` sentinel, and ascending `SubmittedAt` sort on list.
- `custosd`: `--worm-dir` and `--receipt-dir` flags with tilde expansion and in-memory store fallback when both flags are explicitly empty.
- `custosd` HTTP API: `POST /ingest`, `GET /receipts/:id`, and `GET /receipts/:id/verify` endpoints.
- `custosd`: `--max-ingest-bytes` flag (default 32 MiB) bounding the `/ingest` request body via `http.MaxBytesReader`; oversize uploads return HTTP 413 before the body is buffered into memory or verified. `custos/cmd/custosd/main_test.go` covers the 413/422/400 ingest paths and the bind-config guard; `worm_fs_test.go` adds a path-traversal regression for `Retrieve`.
- `internal/proxy`: `CustosPusher` with `PushBytes(ctx, []byte)`, one retry on network fault, and locked bundle byte snapshot taken on `atb.session.close` before the recorder mutex is released.
- `atb intercept --custos` flag to configure the Custos ingest endpoint; prints configured endpoint or disabled status on startup.
- Web viewer: `SessionList`, `ActorSessions`, and `AnomalyBadge` components with Vitest test coverage.
- Web viewer: `/sessions` page with `Navbar` navigation entry.
- `internal/event`: `TypeToolCall`, `TypeDataExport`, `TypeHumanOverride`, `TypeHumanApproval` constants and Registry entries for the four canonical oversight event types.
- `internal/emit`: new `Emitter` package; `ToolCall`, `DataExport`, `HumanOverride`, `HumanApproval` emitters with required-field validation and British English error messages; seven unit tests (stub appendFn, no file I/O).
- `sdk/python/atb/oversight.py`: `ToolCallEmitter`, `DataExportEmitter`, `HumanOverrideEmitter`, `HumanApprovalEmitter`; duck-typed event sink (emit/append dispatch); seven pytest tests.
- `sdk/typescript/src/oversight.ts`: matching four TypeScript emitter classes with options interfaces, session_id propagation, and JSDoc; seven Vitest tests.
- `internal/proxy/export_test.go`: test seams (`NewProxyForTest`, `CaptureRequestForTest`, `CaptureResponseForTest`) for black-box proxy tests.
- `Makefile`: `check-generated` target regenerates the schema-driven bindings (`internal/event/types_generated.go`, `sdk/python/atb/event_types_generated.py`, `sdk/typescript/src/eventTypes_generated.ts`) and fails if any drift from the committed output. Wired into `hygiene-quick`, so the gold-release gate now rejects a schema change that was not regenerated, or a hand-edit to a generated file.
- `internal/proxy`: `CountToolCallsFromResponse` helper counts tool/function invocations in a provider response body (Anthropic `tool_use`, OpenAI `tool_calls`/`function_call`); used to populate `tool_calls_count` on `atb.exchange.complete`.
- `test/schema/emitter_contract_test.go`: executable contract test that drives the Go oversight emitters (`internal/emit`) and the proxy lifecycle records (`SessionCloseRecord`, `ExchangeCompleteRecord`) and asserts every emitted field is declared in `schemas/event.v1.json` `documented_event_types.properties` and every required field is present — turns required/optional field drift into a build failure rather than a silent divergence.
- `internal/event`: `TestRegistryMatchesGenerated` asserts the legacy hand-maintained `Registry` and the schema-generated `RegistryGenerated` describe exactly the same event-type set.
- `docs/custos-handoff.md`: production hardening checklist making the `custosd` threat model explicit — the bearer token is a transport guard, not an identity system, so TLS termination, token rotation, per-tenant/per-user identity, rate limiting, and the hash-chain-only scope of `GET /receipts/{id}/verify` are called out as operator responsibilities before any non-local exposure.
- View server: `GET /api/v1/schema/status` endpoint returning ecosystem-level contract health for the loaded bundle — declared-vs-observed event types, per-type required-field completeness, and any undeclared (unknown) types. Gated by the same session-token auth and verification check as the other read endpoints; covered by `pkg/api/v1/schema_status_test.go` (logic, auth gating, and method rejection).
- Web viewer: `SchemaStatus` component and a "Contract status" panel on the `/sessions` page surfacing the new endpoint — summary cards plus a per-type table that flags `undeclared`, `N incomplete` (with the missing field names), `complete`, and `not observed` states so contract drift is visible at a glance rather than buried in per-session detail. Vitest coverage in `web/components/SchemaStatus.test.tsx`.
- `ai.action.error` forensic event type (schema + Go/Python/TypeScript bindings) recording a privileged action that was attempted but did not succeed (`error_class`: `failed|blocked|timeout|exception|denied_at_sink`), distinct from the success-shaped `ai.action.executed`.
- `atb.llm.request` / `atb.llm.response` registered as capture event types (previously emitted by `atb intercept` but absent from the schema registry).
- `atb intercept` now derives accountability events from captured traffic: `atb.tool.call` per tool the model requests (Anthropic `tool_use`, OpenAI Chat `tool_calls`, OpenAI Responses `function_call`) and `ai.action.error` per failed Anthropic `tool_result` (`is_error: true`). Arguments and error detail are digested, never stored raw.
- `atb incident list` and `atb incident report` commands (`internal/incident`): discover the sessions captured in a bundle, and produce a session-scoped forensic report — integrity, bundle signature provenance, anomaly flags (e.g. `tool_without_approval`), and an ordered event list with each record's hash (markdown or JSON).
- `examples/bundles/incident-capture/`: deterministic agent-incident demo bundle (privileged tool call with no approval + a failed action), wired into `make goldens`.
- `docs/guides/agent-incident-forensics.md`: capture → discover → review walkthrough.
- `internal/emit.ActionError` and Python `atb.oversight.ActionErrorEmitter`: standalone `ai.action.error` writers for direct SDK instrumentation.
- Web viewer: shared `web/lib/event-family.ts` (`eventFamily` / `eventFamilyClass` / `eventSummary`); the timeline colour-codes capture and action events (new `ev-action` family) and the event inspector shows a one-line summary for forensic events.
- Web viewer: `SessionAnomalies` banner on the `/view` dashboard surfacing the loaded bundle's session anomaly flags (e.g. `tool_without_approval`), via an authenticated `getSessions` / `useSessionsQuery` and `web/lib/schemas/session.ts`.
- `data.export.error` event for failed data exports (schema + Go/Python/TS bindings); the TypeScript and Python `DataExportGate` emit it on failure instead of `data.export.executed` with `execution_outcome="error"`.
- Acting-principal attribution: an optional `principal` (`type` human/agent/tool, `id_hash`, `on_behalf_of`) on `ai.action.precommit`, recorded by the `ActionGate` and `HumanOverrideGate` (TypeScript + Python; exported as `ActionPrincipal`).
- `effective_scope` (optional) on `ai.action.executed` — the permission scope/role/grant the action ran under — recorded by the `ActionGate`. Completes the accountability-core schema (principal + execution scope + error event).
- `ai.action.error` writers for direct instrumentation: Go `emit.ActionError` and Python `oversight.ActionErrorEmitter`; the TypeScript and Python `ActionGate` and `HumanOverrideGate` emit `ai.action.error` on failure. The forensic summaries (incident report + viewer) render `ai.action.precommit`/`executed` with principal and scope, and the data-export and action-error events.
- `atb.capture.scope` capture-coverage attestation written by `atb intercept` at startup (captured targets, capture mode, redacted headers, and a plain out-of-scope statement of what the proxy cannot see), surfaced in the incident report header.
- `atb incident export --bundle <path> --session <id> --out <pack.zip>` — a self-contained, independently verifiable incident evidence package (bundle, incident report, verifier report, and a `MANIFEST.json` chain-of-custody record digesting every file; the packed bundle re-verifies on its own).
- Custos signs custody receipts: each receipt carries an Ed25519 `attestation` (`custos/internal/receipt.Signer`) proving Custos received bundle `<hash>` at `<submitted_at>`, verifiable with `receipt.VerifyAttestation` against the embedded public key; `custosd` generates and persists a stable receipt-signing key on first run.

### Fixed
- Viewer session token is re-read from the URL hash on each API request so navigating to a new `atb view` session works without a hard refresh.
- Removed cross-module internal import of Custos receipt types from `internal/proxy` (Items 1-2, Findings 1-2).
- Removed unused imports in `internal/sessionindex` and `custos/internal/receipt` (Findings 3, 6).
- Corrected Custos module import paths in `custosd` and ingest handler from `github.com/pcguest/atb/custos/...` to `github.com/pcguest/custos/...` (Findings 4-5).
- Replaced forbidden `atb/internal/verify` usage in Custos integration tests with public `pkg/custody` types (Finding 7).
- Updated stale `NewBundleRecorder` call sites to pass nil `CustosPusher` (Finding 8).
- Added nil-store guard in Custos ingest handler to prevent panic on valid bundle with uninitialised stores (Finding 10).
- Async Custos push now uses an immutable locked byte snapshot, preventing a file-level race between session close and concurrent bundle appends (Finding 11).
- Session close is now idempotent under lock; idle timer stop with channel drain prevents duplicate `atb.session.close` events and duplicate Custos pushes (Finding 12).
- `internal/sessionindex`: profile inference now matches the canonical event types `ai.policy.decision` and `ai.retrieval.*` (previously the non-existent `atb.policy.decision` / `atb.retrieval.*`), so `policy_decision` and `rag_answer` profiles are correctly inferred.
- `internal/sessionindex`: the `tool_without_approval` anomaly now respects record order — an `atb.human.approval` only closes the flag when it precedes the `atb.tool.call` in the same session; a later approval no longer retroactively clears it. Flags remain scoped per `session_id`.
- `custosd`: refuses to start when bound to a non-loopback `--host` while `CUSTOS_AUTH_TOKEN` is empty (previously only warned). A loopback bind without a token remains allowed for local development.
- Web viewer tests: `SessionList.test.tsx` and `ActorSessions.test.tsx` now import `beforeEach` from `vitest` (the suites previously threw `ReferenceError: beforeEach is not defined` because vitest runs without `globals`).
- Web viewer test harness: `vitest.setup.ts` now registers a global `afterEach(cleanup)` so rendered output no longer accumulates across `it` blocks; without it, `getByText` queries collided with lingering DOM from earlier tests ("found multiple elements").
- View server: `actorsForRequest` and `sessionsForRequest` now return a cloned session index on the cached-error path as well as the success path, so an `sessionIndexErr` response can no longer alias the server's shared session slice/actor map.
- Web viewer: `ActorSessions` actor headers and session rows are keyboard-accessible (`role="button"`, `tabIndex={0}`, Enter/Space activation, and `aria-expanded` on the expandable header) rather than mouse-only `onClick` targets.
- `internal/sessionindex`: bundle-level records (manifest, signature, anchor, push marker, snapshot) no longer seed a path-derived pseudo-session — signing a captured bundle no longer adds a spurious second session to `atb incident list` / `/api/v1/sessions`.
- `internal/proxy`: the `ai.action.error` emitted for a failed tool result now carries `session_id`, so it groups with its capture session instead of a separate path-derived one.
- `examples/bundles/demo-workflow`: verifies PASS without an explicit `--profile` by declaring its workflow class (`policy_decision`) via the `ai.request.received` `purpose_tag`, instead of falling through to the `privileged_tool_action` heuristic.

### Changed
- `internal/proxy/session.go`: `actor_id` now unconditionally emitted on `atb.session.close`; was previously omitted when identity resolution returned an empty string.
- `internal/proxy/recorder.go`: `AppendEventHash` returns the appended record hash to enable provability linkage from `atb.exchange.complete` → hashed request record.
- `internal/proxy/forward.go`: `atb.exchange.complete` now emits a superset payload — `actor_id`, `model`, `input_tokens`, `output_tokens`, `tool_calls_count`, `latency_ms`, `completed_at`, and `request_event_id` sourced from the real append hash — in addition to the original required fields.
- `internal/proxy/session.go`: `tool_calls_count` on `atb.exchange.complete` now reflects a real count parsed from the response body (`CountToolCallsFromResponse`: Anthropic `tool_use` blocks, OpenAI Chat Completions `tool_calls`, OpenAI Responses `function_call`/`tool_call` items) instead of a hardcoded `0`. Best-effort: an unrecognised or unparseable body yields `0` and never blocks recording.
- `docs/spec-ai-traces.md`: the `atb.exchange.complete` subsection now distinguishes required, always-emitted (`actor_id`, `completed_at`, `tool_calls_count`), and optional fields, matching the emitted payload exactly.
- `schemas/event.v1.json`: added the four canonical oversight event types (`atb.tool.call`, `atb.data.export`, `atb.human.override`, `atb.human.approval`) and the two proxy-internal session types (`atb.session.close`, `atb.exchange.complete`) to the schema source of truth, with `required_fields` and matching `documented_event_types` entries. Regenerated the Go/Python/TypeScript bindings; the generated `internal/event/types_generated.go` now owns the canonical type constants (previously hand-declared in `types.go`), removing the duplicate-declaration drift between the generator and the committed output.
- `internal/emit`, `sdk/python/atb/oversight.py`, `sdk/typescript/src/oversight.ts`: renamed the optional tool-call digest fields from `input_hash`/`output_hash` to `tool_input_digest`/`tool_output_digest` to follow the ATB `*_digest` naming convention used elsewhere in the schema. The Go emitter validation error prefix is now `atb:` (was `emit:`), matching the Python and TypeScript surfaces.
- `docs/spec-ai-traces.md`: reconciled the `atb.tool.call`, `atb.data.export`, `atb.human.override`, and `atb.human.approval` field tables with the shipped SDK contract (required fields are the essential identifier plus `session_id`; `actor_id`, digests, `record_count`, and `classification` are optional) and added `atb.bundle.pushed` to the complete event type registry table.
- `tools/eventgen`: the generator now maps the generic `atb.<namespace>.<name>` form to the canonical hand-named constants (e.g. `atb.tool.call` to `TypeToolCall`) so schema-driven generation matches the existing Go and SDK constant names.
- `schemas/event.v1.json`: `documented_event_types` for `atb.session.close` and `atb.exchange.complete` now declare their always-emitted and optional fields as `properties` (`model`, `exchange_count`, `total_tokens`, `closed_at` for session close; `actor_id`, `completed_at`, `tool_calls_count`, `model`, `input_tokens`, `output_tokens`, `latency_ms` for exchange complete), matching the proxy payload exactly. The `required` arrays are unchanged, so the schema-consistency test and generated bindings are unaffected.
- `docs/spec-ai-traces.md`: the `atb.session.close` subsection now documents its always-emitted fields (`model`, `exchange_count`, `total_tokens`, `closed_at`), matching the schema and emitted payload.
- `internal/event/types.go`: the legacy deprecated `Registry` now includes `atb.session.close` and `atb.exchange.complete`, ending its silent divergence from the schema-generated `RegistryGenerated` (guarded by the new parity test).
- `internal/proxy/session.go`: simplified `NoteExchange` token accumulation to `TotalTokens += promptTokens + outputTokens`, removing a tautological no-op branch; behaviour is unchanged.
- `atb intercept` records request/response bodies as a SHA-256 digest (`body_sha256`) and byte length (`body_bytes`) by default, not raw content; pass `--capture-bodies` to retain raw bodies. `ScanHeaders` now also strips `Proxy-Authorization`, `Cookie`, and `Set-Cookie` (in addition to `Authorization` / `X-Api-Key`), so no credential or session secret is persisted.
- SDK action gates emit `ai.action.error` when a gated action fails, instead of a success-shaped `ai.action.executed` with `execution_outcome="error"` — TypeScript and Python `ActionGate` (sync + async) and `HumanOverrideGate`. Success still emits `ai.action.executed`.
- `docs/roadmap.md` and `README.md`: record the shipped agent-incident-forensics capability (capture, `ai.action.error`, `atb incident list`/`report`, viewer rendering) and mark Phase 10 proxy capture complete.

## [v1.12.0] - 2026-05-25

### Added
- [Custos development handoff](docs/custos-handoff.md): layer model, stable contracts, Custos build phases, and demo script for custodian-of-record development.
- Custos ingest conformance test (`test/custos/conformance_test.go`) locking `verify.report.v1` on pass fixtures and checking emitted top-level report fields against the frozen custody schema.
- Profile workflow SDK helpers (TypeScript and Python): `DataExportGate`, `PolicyDecisionRecorder`, `HumanOverrideGate`, and `BackgroundJobTracker` for emitting canonical events aligned with built-in obligation profiles.
- `examples/bundles/generate-profile-fixtures.sh` to regenerate passing and failing `.atb` fixtures for all six built-in profiles.
- `go run ./scripts/generate_profile_fixtures.go` as the canonical fixture generator (also available via the shell wrapper).
- Viewer API endpoint `GET /api/v1/bundle/verify/report` returning the stable `verify.report.v1` JSON contract for auditors.
- Optional `corroboration_bonus` and `effective_score` fields in `atb verify --format json` output when corroboration policy applies.
- `atb evidence pack` to verify multiple local bundles and emit a combined JSON or Markdown evidence summary.
- `atb agent run` CLI entrypoint for the optional local Agent service, including loopback health/info, capture session open/append/close, and read-only workspace bundle listing APIs.
- TypeScript and Python `AutomationSession` helpers for multi-hop workflow capture, with optional routing through the local ATB Agent when `ATB_AGENT_URL` or `ATB_AGENT_AUTO` is set.
- Internal TypeScript and Python Agent HTTP clients used by `AutomationSession` for local session open/append/close flows.

### Changed
- Expanded `docs/cas-guide.md` with canonical sub-score table, grade bands, and per-profile interpretation examples.
- Updated `docs/api/verify-schema.md` and `docs/profiles.md` (blind spots, `required_when` semantics) to match verifier behaviour.
- `verify.report.v1` now includes `profile_version` and propagates `residual_risk.drivers` / `recommended_next_evidence` from the internal verifier report.
- Human-readable `atb verify` text output opens with a concise Summary block (integrity, profile, CAS, top issues, exclusion count).
- Fixed stale integration tests that used `ai.action.*` events for the `atb.profile.data_export` profile.
- ATB Agent and automation docs now describe the implemented local capture/workspace APIs rather than the earlier health/info-only placeholder.

### Fixed
- `atb.profile.policy_decision` test fixtures now include required `ai.action.precommit` (triggered by `required_when` when `ai.request.received` is present).
- Pinned the `verify.report.v1.schema.1` SHA-256 in tests so custody schema changes are reviewed deliberately.

## [v1.11.0] - 2026-05-23

### Docs
- Added `AGENTS.md` as the canonical maintainer and coding-agent harness and aligned the core Markdown estate around it.
- Tightened README scope and quickstart flow so current capability, non-goals, and planned work are more clearly separated.
- Aligned contributing, release, roadmap, security, and versioning docs with the current local-first viewer and release model.

### Changed
- Main branch CI and toolchain hygiene now align with the agent-safe subset from
  `audit/complete-atb`: Go 1.26.3, support-matrix drift checks, quality evidence,
  and refreshed npm lockfiles for the TypeScript SDK and web viewer.

### Added
- `atb capture run` — wrap any child command with ATB capture environment
  variables; stamps the resulting bundle with a capture run ID for provenance.
- `atb import chatlog` — import saved AI chat logs (Claude, OpenAI, generic
  JSONL) into a local ATB bundle, mapping turns onto the canonical event taxonomy.
- Capture v1 event mapping: user turns become `ai.request.received`, assistant
  turns become `ai.model.invoked` + `ai.model.output` + `ai.response.sent`,
  tool turns become `ai.tool.exec`, and system context turns are recorded as
  prompt-window context.
- `internal/evidence` package and `atb evidence --bundle <path> [--format text|json]`
  for structured local bundle evidence summaries, including manifest, snapshot, and
  per-signature provenance.
- `exitLockContention = 9` for advisory bundle lock contention so automation can
  distinguish retryable contention from general system failures.
- Python SDK local signing provenance fields (`backend`, `key_id`, `signed_at`) and
  a `verify()` signatures array matching the Go JSON shape.
- `atb.corroboration.external` event type in the new `atb.corroboration.*` namespace.
  Required fields: `source`, `reference_id`, `digest`, `retrieved_at`. Optional fields:
  `adapter`, `raw_evidence` (base64, capped at 4 KB), `truncated`. Schema locked at v1.
- HTTP gateway receipt adapter in `internal/corroboration/`. Fetches a JSON receipt from a
  configured URL, computes the SHA-256 digest of the response body, and returns a record
  ready to append as an `atb.corroboration.external` event. Raw evidence is stored up to
  4 KB; payloads larger than 4 KB set `Truncated=true` and omit the raw body.
- `atb corroborate` subcommand: fetches a receipt from an external adapter and appends an
  `atb.corroboration.external` event to the active bundle. Flags: `--source` (required),
  `--url` (required for `http-gateway`), `--ref` (required), `--bundle` (optional),
  `--dry-run`, `--format text|json`.
- Verifier awards XC sub-score credit for well-formed `atb.corroboration.external` events.
  One valid corroboration event earns XC=1.0. Bundles without corroboration events return
  their anchor-based XC score unchanged — identical to v1.9.0 behaviour.
- `internal/push/`: `Push` interface, `S3Pusher`, and `QueuePusher` for signed queue
  gateway envelopes.
- `atb push`: `--queue <endpoint-url>` and `--hmac-key <hex-key>` flags. Queue pushes
  POST a signed JSON envelope after any S3 upload completes.
- `atb push --dry-run`: previews the queue endpoint and envelope JSON as well as the
  existing S3 target resolution path.
- S3 push coverage now checks that Object Lock PUT requests carry
  `x-amz-object-lock-mode: COMPLIANCE` and
  `x-amz-object-lock-retain-until-date`.

### Fixed
- Bundle signature append now uses `writeAtomic` (temp file, fsync, rename) instead
  of truncate-in-place writes, preserving the original bundle if the final write fails.
- Stale documentation and maintenance references to the removed legacy viewer flag.

### Docs
- `docs/spec-v1.0.md` and `schemas/event.v1.json`: signature provenance fields
  documented as current optional `atb.bundle.signature` payload fields.
- `docs/spec-ai-traces.md`: `atb.corroboration.*` namespace and required field schema
  documented, corroboration event added to the complete event type registry table.
- `docs/architecture.md`: corroboration model section added, covering the problem addressed,
  what the event records, XC scoring, the trust limitation, and adapter extension points.
- All six built-in profile templates: blind-spot text updated to note XC credit conditions.
- `docs/integrations/push-transports.md`: S3 WORM headers, queue gateway envelope and
  HMAC signing, and the transport security boundary.

## [v1.10.0] - 2026-04-21

### Fixed
- Viewer events endpoint returning incorrect data
- Version string consistency across all packages (cmd/atb/main.go, SDKs, web)
- Integration test golden value updated to v1.10.0
- check-versions.sh now derives expected version from latest git tag

## [v1.9.0] - 2026-04-20

### Added
- **CAS v1 corroboration bonus**: `CorroborationPolicy` struct (`AnchorBonus` 0.05,
  `SignatureBonus` 0.03, `SnapshotBonus` 0.02, `MaxBonus` 0.10) with `Validate()` and
  `DefaultCorroborationPolicy()`. `EvaluateBundle` accepts a new `WithCorroborationPolicy`
  option; when set, `CASResult` gains `corroboration_bonus` and `effective_score` fields
  (grade derives from `effective_score`; nil policy produces output identical to v1.8.0).
  `atb verify` automatically applies the default policy when `--with-anchor` is present.
- **`--corroboration-policy <path>`** flag on `atb verify`: accepts a JSON file matching
  `CorroborationPolicy` to override the default bonus values.
- **Typed error sentinels** in `internal/verify/evaluate.go`: `ErrBundleNotFound`,
  `ErrChainInvalid` (returned when `RequireValidChain` is set and the chain fails), and
  `ErrProfileUnknown` (all supplied profiles were nil). Callers can use `errors.Is`.
- **`--policy-doc <path>`** flag on `atb append` (`ai.policy.decision` events only):
  reads the file, computes `SHA-256(contents)` hex, and embeds it as `policy_doc_hash`.
  When `--sign-policy` is also set, stores a compound Ed25519 `policy_doc_signature`
  over `SHA-256(canonical payload) || SHA-256(doc bytes)`.
- **`VerifyPolicyDocSignature`** in `internal/sign`: verifies the compound policy-doc
  signature. `policy_doc_signature_valid` boolean surfaced in `TrustReport` (nil when
  no `policy_doc_hash` present, true/false otherwise).
- **`atb version --json`**: outputs `{"version":"1.9.0","algorithm":"SHA-256+RFC8785","anchor":"RFC3161-optional"}` and exits 0.
- **TypeScript SDK `version()` and `SDK_VERSION`**: `version()` returns
  `{ version: SDK_VERSION, algorithm: "SHA-256+RFC8785" }`; `SDK_VERSION` exported as
  a named constant.

### Fixed / CI
- CodeQL workflow upgraded from floating `@v3` to SHA-pinned `@v4` refs; UI embed seed
  step added before Autobuild to resolve the `uiembed.go` pattern error.
- `version-gate.yml` floating `@v4` tag replaced with pinned SHA (`actions/checkout`).

### Tests
- `atb profiles validate --format json` snapshot test: asserts exit 0, all built-in
  profiles present, every entry reports `valid: true` with no errors.
- Corroboration bonus: nil-policy, signature-only, signature+snapshot, all-three-capped
  test cases in `internal/verify/evaluate_test.go`.
- Policy-doc signature: round-trip, absent-signature, tampered-doc, tampered-event
  cases in `internal/sign/policy_test.go` and CLI integration tests in
  `cmd/atb/main_test.go`.
- `TrustReport.PolicyDocSignatureValid`: nil-when-absent, true-when-verified,
  false-when-absent-signature test cases in `internal/verify/trust_report_test.go`.

### Docs
- `docs/roadmap.md` updated to reflect CAS v1 corroboration bonus and policy-doc compound
  signature shipped in v1.9.0; long-term objective section added.

## [v1.8.0] - 2026-04-19

### Added
- Added a verifier evaluation shim in `internal/verify/evaluate.go`: `EvaluateBundle` and `EvaluateConfig` centralise bundle loading, hash-chain integrity, RFC 3161 anchor verification, CAS normalisation, profile stamping, residual risk, and post-profile transformations in one place. The CLI, viewer, and API surfaces now derive reports from this function.
- Added `atb profiles validate`: validates all built-in profiles and any additional profiles supplied via `--file` or `--dir`; checks required fields, duplicate IDs, and CAS weight-vector sums; exits 0 or 1 and supports text or stable JSON output via `--format json`.
- Added `docs/roadmap.md`: in-repo roadmap covering short-term hardening for the profile DSL and verifier report path, medium-term CAS v1 and source signatures, and longer-term corroboration adapters, queue and storage gateways, reconciliation, and assurance-pack exports. Linked from `README.md`.

### Fixed
- TypeScript SDK parity suite now passes after completing the TypeScript 6.0.3 migration; `sdk/typescript/tsconfig.json` now carries the TS 6 declaration-build compatibility setting required by the updated toolchain.

### Tests
- Added `internal/verify/evaluate_test.go`: shim tests covering healthy bundles, broken hash-chains, and missing-event obligation failures.
- Added `cmd/atb/profiles_test.go`: coverage for built-in profile validation, malformed DSL files with bad weight sums and missing required fields, duplicate profile IDs, and JSON output format.

### Docs
- Added `docs/roadmap.md` and linked it from `README.md` under Planned work.

## [v1.7.3] - 2026-04-18

### Added
- Added a GitHub CodeQL static analysis workflow for Go on pushes and pull requests to `main`, plus a weekly scheduled scan.

### Changed
- Bumped `golang.org/x/crypto` to `v0.50.0`.
- Bumped `actions/setup-python` from `v5.6.0` to `v6.2.0`.
- Updated the `/web` dependency group: `next` to `16.2.3`, `axios` to `1.15.0`, `basic-ftp` to `5.3.0`, `follow-redirects` to `1.16.0`, `lodash` to `4.18.1`, and `vite` to `7.3.2`.
- Updated `llama-index` in `sdk/python` to `>=0.14.16`.
- Added `.atb-agent/` to `.gitignore` to prevent local agent tool binaries from being committed accidentally.

### Fixed
- Static Cypress runner now passes `--env MOCK_API=true` so the dashboard reaches loaded state before assertions run; fixes the gold release gate E2E failure introduced when `waitForDashboard` began checking that `trust-score-value` leaves the loading (`TEST-MODE`) state.

### Fixed
- Trust Score dashboard card: background colour corrected to comma-separated
  `hsl(H, S%, L%)` syntax on the `motion.div` wrapper and inner `Card` (loaded
  state) and on the loading-state `Card`; resolves axe `color-contrast`
  violations caused by space-separated HSL being rejected by some browsers.
- Accessibility test (`cypress/support/e2e.ts`): `waitForDashboard` now waits
  for the trust-score value element to leave the loading state before running
  axe, ensuring the audit targets rendered content rather than skeletons;
  violation details are now written to stderr via `console.error` so they
  surface in headless CI logs.

## [v1.7.2] - 2026-04-16

### Added
- Profile DSL v1: user-defined profile format defined in YAML. Custom profiles are
  evaluated identically to built-in profiles; `atb verify --profile <path>` and
  `atb trust-report --profile <path>` both accept a YAML file.
- CAS support for custom profiles via `genericSchemaSubScores`; any DSL profile with
  `supports_cas: true` now produces a full CAS object including sub-scores.
- `atb push` supports S3-compatible storage endpoints via `--endpoint-url`; push
  defaults (`target`, `endpoint-url`, `region`, `lock-mode`, `lock-until`,
  `credentials-source`) can be stored in `.atb/config.json` under a `push` key.
- `atb view --profile <id-or-path>`: evaluates the bundle against a named built-in
  profile or a DSL YAML file at startup and serves the result at
  `GET /api/v1/bundle/profile` (204 when no report; 200 + `ProfileReportSummary` when
  computed). `POST /api/v1/bundle/verify` recomputes and caches a fresh report.
- Legacy viewer redirect guidance added during the transition to the current single-view dashboard.

### Fixed
- Bundle save is now atomic: written to a temp file then renamed, preventing partial
  writes on crash or disk-full.
- `loadSchemaIfAvailable` replaced panic-recover with a proper error return.
- `classifyAnchorRoots` global eliminated; root certificates are now passed through
  the call chain, removing a global mutable state hazard.
- Misaligned `else` brace and struct field alignment in `verify.go` corrected.
- Read-only directory test skipped on Windows where the permission model differs.
- Go toolchain pinned to 1.24.2 across `go.mod`, Makefile, CI, and Dockerfile;
  all affected stdlib CVEs cleared.

### Changed
- Anchor verification consolidated: `internal/anchorverify` package removed; logic
  merged into `internal/trust`. No change to the public `atb verify` behaviour.

### CI
- Security scans now run on every push and PR, not only on schedule.
- `-race` flag added to the Go test step; TypeScript SDK tests added to CI.
- `check-versions.sh` runs in CI via a dedicated version-gate workflow.

### Tests
- Added RAG GC fixed-score, empty-bundle verify, and MCP RAG tool round-trip tests.
- Integration CAS assertions updated to expect non-nil for all named profiles.

### Docs
- `docs/integrations/worm-s3.md`: S3-compatible endpoint support documented; push
  event language removed to match implementation (local bundle is not modified on export).
- `docs/spec-dashboard.md`: `--profile` CLI interface, new API routes, and
  Profile/CAS summary panel spec added.
- `docs/compliance/eu-ai-act.md`: identity attribution boundary section clarifies
  ATB proves non-alteration but not truthfulness of claimed actor identities.
- `docs/security.md`: identity attribution caveat strengthened; recommended controls
  updated to require an independent identity layer or signing scheme.
- Quickstart Python example corrected to use canonical `rag_answer` event types.
- PageIndex event type mismatch with `rag_answer` profile and fixed GC sub-score
  documented.

## [v1.6.0] - 2026-04-15

### Added
- `atb trust-report --format text` adds a human-readable trust report with ANSI
  status colour (PASS/FAIL/WARN) and a conditional CAS block showing profile,
  grade, anchor quality label, and all eight sub-scores.
- `atb snapshot <name>` appends an `atb.snapshot` record containing `name`,
  `bundle_hash` (SHA-256 hex of serialised bundle), `record_count`, and
  `snapshot_at` (RFC 3339 UTC). Accepts `--bundle` and `--quiet`.
- `internal/event` adds `TypeSnapshot = "atb.snapshot"`.
- `internal/verify` adds an offline RFC 3161 fixture
  (`testdata/anchor_token_verified.tsr`) and generator, and unskips
  `TestClassifyAnchor_Verified`.
- SDK version parity updates `sdk/python` and `sdk/typescript` to `1.6.0`.

### Changed
- `cmd/atb/main.go` replaces the snapshot stub with the real command in
  `snapshot.go`.
- `internal/verify/anchor_classify.go` adds a narrow root-pool hook for test
  override while leaving the production path unchanged.

### Pre-launch surface polish

- `atb view` now accepts `--profile <id-or-path>`: evaluates the bundle against the
  named built-in profile or a DSL YAML file at startup and exposes the result via
  `GET /api/v1/bundle/profile`. Without `--profile`, a "Run verify" button triggers
  `POST /api/v1/bundle/verify` to compute and cache a fresh `ProfileReportSummary`.
- Added `GET /api/v1/bundle/profile` (204 No Content when no report; 200 +
  `ProfileReportSummary` when computed) and `POST /api/v1/bundle/verify` to the
  dashboard API server.
- Rewrote README above-the-fold: explicit Why ATB bullets, What ATB does not do
  sub-section, `atb push` WORM export added to multi-surface list, demo asset hooks.
- Updated `docs/spec-dashboard.md` with new API routes, `--profile` CLI interface,
  and Profile/CAS summary panel spec; updated `docs/quickstart.md` section 4 to cover
  `--profile` usage and the verify report screenshot reference.
- Added `docs/launch/assets/` canonical path table to `docs/launch/README.md` with
  regeneration reminders for demo GIF and verify-report screenshot.

## [v1.5.1] - 2026-04-13

### Changed
- Align version constants to v1.5.1 across CLI, Python SDK, TypeScript SDK, and web package.

## [v1.5.0] - 2026-04-13

### Added
- MCP in-process integration test (`TestServeMCPVerifyIntegration`) exercises `atb_init` and `verify` via the MCP server without shelling out: initialises a bundle, appends an event via the bundle package, calls `verify` through the MCP tool handler, and asserts `exit_code=0` with no critical failures (implies `chain_valid=true`). Closes the test gap noted by the TODO in `internal/mcp/server_test.go`.
- `docs/integrations/mcp.md`: Claude Code CLI configuration example (`claude mcp add` and `.mcp.json`); MCP `initialize` handshake response shape; full input schemas and required-field tables for `verify`, `rag_index_record`, and `rag_retrieval_record`; tool list order now matches the implementation.

## [v1.4.0] - 2026-04-12

### Added
- `atb verify --with-snapshot-check` validates each `atb.snapshot` `bundle_hash` against the serialised prefix; mismatches report `snapshot_hash_mismatch at seq N`.
- `cas` object on `atb trust-report --json` output (alongside existing `cas_score` / `cas_grade`), aligned with verify report CAS.
- EU AI Act Article 12 mapping (`docs/compliance/eu-ai-act.md`) with profile table and limitations.
- Text verify output note when obligations fail: CAS is diagnostic only and does not overturn FAIL.
- NIST AI RMF practitioner mapping (`docs/compliance/nist-ai-rmf.md`) for CAS sub-scores and built-in profiles.
- End-to-end quickstart example under `examples/quickstart/`, including a runnable script and captured terminal output for a verifiable `privileged_tool_action` bundle with a CAS grade line.
- OpenTelemetry comparison guide (`docs/comparisons/opentelemetry.md`).

### Changed
- CAS weight vectors for `background_automation`, `policy_decision`, and `human_override` YAML templates to the documented profile-specific values (`data_export` unchanged).
- README: Install moved to follow Trust Model (verification narrative before installation).
- `docs/security.md` Limitations expanded: intra-bundle integrity vs capture completeness, local-first filesystem trust boundary.
- `docs/spec-v1.0.md`: snapshot `bundle_hash` definition, `--with-snapshot-check` behaviour, and that the field is not verified without the flag.
- `docs/compliance/eu-ai-act.md` rewritten to match the Article 12 mapping structure (overview, profile table, out-of-scope paragraph).
- `atb view` keeps a loopback default host and accepts an explicit `--host` override.
- `atb verify` and `atb trust-report` now report RFC 3161 anchor state explicitly as verified, partial, or failed.
- `docs/security.md` now states the local viewer exposure boundary and the exact RFC 3161 checks performed during `--with-anchor`.
- `docs/key-management.md` now states the versioned PBKDF2-SHA256 parameters for new and legacy encrypted bundles.
- CI now checks internal Markdown links under `docs/` and `README.md`.
- `sdk/python/README.md` now reflects the actual exported Python SDK import paths, append flow, and event type constants from `sdk/python/atb/`.
- `sdk/typescript/README.md` now reflects the actual exported TypeScript SDK package imports, append flow, and event type constants from `sdk/typescript/src/`.
- Docs and examples navigation updated to add the quickstart and comparison entries after auditing the edited index links.

### Fixed
- `--sign-policy` exits non-zero when no signing key is present (`no signing key found; run 'atb keygen' before using --sign-policy`).
- `atb verify` no longer reports a successfully validated TSA anchor as unverified in text or JSON output.

### Security
- `atb encrypt` writes ATBE wire version `0x02` with PBKDF2-SHA256 at `600000` iterations; `atb decrypt` accepts `0x01` (`100000`) and `0x02`.
- `atb verify --with-anchor` now requires TSA certificate-chain verification against the system roots in production before AC receives anchor credit.

### Tests
- CLI: `--sign-policy` without keypair; `--with-snapshot-check` tamper path and verify without the flag unchanged.
- `internal/verify`: CAS scenarios for extended profiles; trust-report JSON asserts `cas` when present.
- Viewer listener test now asserts that the default bind address is `127.0.0.1`.
- Anchor verification tests now cover verified, partial, and failed reporting states.

## [v1.0.0.1] - 2026-04-09

### Changed
- Version metadata aligned to `v1.0.0.1` release tag
- Docker publish hang fixed; login logout disabled, timeout added
- Smoke check drift resolved in release guidance and checklist

## [v1.0.0] - 2026-04-09

### Changed
- Version bumped to `1.0.0` across CLI, Python SDK, TypeScript SDK, and web package
- GitHub Actions updated to Node 24-compatible versions
- Docker publish workflow reviewed against `DOCKERHUB_USERNAME` secret presence; no gate added because the repository secrets are configured

### Notes
- `v1.0.0-rc` tag remains at `a65a70b`; `v1.0.0` supersedes it
- `v1.0.0` tag from 2026-03-10 (`bb2cccb`) refers to an earlier development iteration and is distinct from this release

## [v1.0.0-rc] - 2026-04-09

### Added
- Ed25519 bundle signing integration test
- v1.0.0-rc release-readiness checklist
- Key rotation procedure in `docs/key-management.md`

### Changed
- Version bumped to `1.0.0-rc` across the CLI, Python SDK, TypeScript SDK, and web package
- Roadmap updated to reflect completed `v0.9.x` items and `v1.0.0-rc` scope

## [v0.9.2-beta] - 2026-04-08

### Changed
- Version bump to 0.9.2-beta; closes the April 2026 release window
- Python SDK version aligned to 0.9.2b1

## [v0.9.1-beta] - 2026-04-08

### Added
- `atb trust-report --format json` TrustReport output with profile-specific evidence sections for all six built-in profiles
- `atb export --format compliance --json` ComplianceManifest output
- MCP integration guide (`docs/integrations/mcp.md`)
- Event type constants for Python and TypeScript SDKs
- Contributor orientation and working standards documentation

### Fixed
- `background_automation` profile migrated to `ai.job.*` taxonomy; template, verifier, and auto-detection all consistent
- ResidualRisk now set to Critical when chain integrity fails
- TSA and CAS capability claims corrected in README and security.md

### Tests
- Snapshot tests for `atb verify --format json` across all six profiles
- Snapshot tests for `atb trust-report --format json` across all six profiles, including negative and edge cases
- Golden fixture for compliance manifest JSON output

## [v0.9.1-beta] - 2026-04-07

### Changed
- Align release metadata across CLI version output, README badges/status, and SECURITY supported-version table.

## [v0.9.0-beta] - 2026-04-XX

### Changed
- Versioning reset to v0.9.0-beta to accurately reflect pre-production status
- TSA verification: certificate chain validation is implemented and used for CAS scoring
- Bundle-level Ed25519 signing: fully implemented via `atb sign` and `atb verify`

## [v1.5.0] - 2026-03-31

### Added
- Add `atb bundle new` as an alias for `atb init`.

### Fixed
- Guard against a `nil` `SubScores` map in verify output and add SC fallback
  handling for unmatched profiles.
- Copy all embedded source files into the Go builder Docker stage.

### Changed
- Build web assets before `go vet` in the Gold Release Gate workflow.
- Install `sdk/typescript` dependencies before Gold Release Gate tests.
- Bump SDK versions to `1.4.0` for release tag parity.
- Pin GitHub Actions workflow steps to full commit SHAs.
- Update `vitest` from `4.0.18` to `4.1.2` to address the `flatted` CVE.
- Update `docker/setup-buildx-action` from `3.12.0` to `4.0.0` for Node 24.
- Update `actions/upload-artifact` from `4.6.2` to `7.0.0` for Node 24.
