## Current state

ATB v1.13.0 ships a verified core bundle engine, six obligation profiles with CAS scoring, Go/Python/TypeScript SDKs, the EU AI Act retention guard, the `atb` CLI, MCP transport, and the `verify.report.v1` custody contract. The shipped runtime covers local capture, hash-chained bundle integrity, signing, encryption, TSA anchoring, WORM export, queue push, and corroboration event recording.

## Completed — Phase 9 (Q3 2026)

- ✅ OTel translator scaffold (`pkg/otel`) — span structs mapped to canonical AI trace events (completed 28 May 2026)
- ✅ GitHub audit log corroboration (`pkg/corroborate/github`) (completed 28 May 2026)
- ✅ LangChain/LangGraph corroboration (`pkg/corroborate/langchain`) (completed 28 May 2026)
- ✅ Session index and actor grouping (added 28 May 2026 — completed out of roadmap order)
- ✅ UI session list and anomaly badge views (added 28 May 2026 — completed out of roadmap order)

## Completed — Agent incident forensics (May–June 2026)

The agent-incident-forensics wedge: capture an agent session, then reconstruct
and review it from a tamper-evident bundle.

- ✅ Accountability core (per the research memo): `ai.action.error` for failed/denied actions, `data.export.error` for failed exports, optional `principal` (human/agent/tool + on_behalf_of) on `ai.action.precommit`, and `effective_scope` on `ai.action.executed` — schema + Go/Python/TS bindings, recorded by the SDK gates and surfaced in the forensic report and viewer
- ✅ `atb intercept` records captured exchanges privacy-safely — bodies digested by default (`--capture-bodies` to retain raw), credential/session-secret headers stripped
- ✅ `/view` dashboard surfaces session anomaly flags (e.g. `tool_without_approval`)
- ✅ Capture-coverage attestation (`atb.capture.scope`) — the recorder states what it can and cannot see
- ✅ `atb incident export` — self-contained, independently verifiable incident evidence package (bundle + reports + chain-of-custody manifest)
- ✅ Custos signs custody receipts (Ed25519 attestation of receipt, verifiable against the embedded key)
- ✅ Detection: `policy_denied_executed` and `action_failed` anomalies over the `ai.*` gate/proxy events
- ✅ Explained findings: the incident report turns each raised anomaly flag into a located finding (severity, plain-English meaning, triggering event sequence numbers) in markdown and JSON, with per-event `triggered_flags` in the NDJSON for SIEM record-level alerting
- ✅ Streamed (SSE) tool-call extraction for OpenAI and Anthropic
- ✅ NDJSON incident report format for SIEM ingestion
- ✅ Proxy emits accountability events: `atb.tool.call` per requested tool, `ai.action.error` per failed Anthropic `tool_result`
- ✅ Registered `atb.llm.request` / `atb.llm.response` capture event types
- ✅ `atb incident list` (discover sessions) and `atb incident report` (session-scoped report: integrity, signature provenance, anomalies, hash-addressed events)
- ✅ Viewer renders capture/action events by family with one-line summaries
- ✅ SDK action gates emit `ai.action.error` on failure (TS + Python ActionGate, TS + Python HumanOverrideGate); Go `emit.ActionError` + Python `oversight.ActionErrorEmitter`

## Near term — Q3 2026

- ✅ Phase 10: Transparent Proxy Capture (`atb intercept`) — local HTTPS MITM/reverse proxy that records AI API traffic, tool calls, and failures into a live bundle (shipped May–June 2026)
- ✅ Proxy auto-push to Custos on session close (added 28 May 2026 — completed out of roadmap order)
- ✅ Formalise obligation-profile DSL v1 (completed 29 May 2026)
- ✅ Produce verifier report v1 structured output (completed 29 May 2026)
- Wire automatic capture to Claude and OpenAI SDK callbacks
- Enforce EU AI Act Article 12 logging in automatic capture path

## Medium term — Q4 2026 to Q1 2027

- OTLP decode and GenAI semconv mapping (pkg/otel full implementation)
- DB reconciliation assurance packs
- Reviewer identity anchoring (EU AI Act Article 14 gap closure)
- Retention enforcement access logging
- Automated compliance evidence pack export (Articles 17–20 gap)
- CAS v1 formalisation with provability ladder output

## Out of scope (explicit)

- Managed storage, SSO, or RBAC
- Hosted tracing or telemetry collection
- Real-time prevention or blocking of AI actions
- Universal completeness guarantees — `atb intercept` captures provider API traffic, but an agent that bypasses the proxy (or a direct in-process SDK call) is not seen; completeness is bounded by what flows through the recorder
- Training data governance (Articles 10–11)

## Custos

Custos is the in-repo reference receipt, custody, and attestation layer for ATB bundles. It lives under `custos/` as a separate Go module and is being scaffolded incrementally: the ingestion boundary, receipt store, per-org signing policy, and auth packages have unit tests today, while discovery, registry, onboarding, oversight, and insights are early scaffolds. Custos demonstrates how recorded bundles are ingested, signed, and held under custody — it is reference infrastructure, not a finished product.

Hosted, multi-tenant concerns — central auditor portal hosting, billing, SSO/RBAC, legal hold, and custodian-of-record operations — remain outside the ATB runtime and outside this repository, per `AGENTS.md`. The roadmap below tracks the in-repo reference layer only.

## Custos Enterprise Layer (in-repo reference)

- ✅ Phase 10: Ingestion engine scaffold (custos/ package tree) — Q3 2026 (completed 28 May 2026)
- ✅ Custody log made auditable: content-addressed ingest fixed (filesystem stores now accept real bundles), `GET /receipts` enumerates the log, `GET /receipts/:id/attestation` verifies the Ed25519 custody attestation server-side, and `GET /custody/key` publishes the signing key for independent (operator-token-free) attestation verification and rotation detection (June 2026)
- ✅ Phase 10: Automatic bundle signing policy per org — `custos/signing` now
  persists per-org policy (key source/reference, RFC 3161 TSA toggle, cron
  rotation schedule) via `InMemoryPolicyStore` and an owner-only
  `FileSystemPolicyStore`, with `SigningPolicy.Validate()` guarding org ID, key
  reference, key source, and the cron schedule. Custos records the policy; ATB
  core performs the signing (completed June 2026)
- Phase 10: AI tool discovery + registry — Q3 2026
- Phase 10: Onboarding flow + API key provisioning — Q3 2026
- Phase 11: Human-in-the-loop review queue + oversight — Q4 2026
- Phase 11: Auditable work tree UI + handoff lineage — Q4 2026
- Phase 11: Insight extraction + pitfall detection — Q4 2026
- Phase 12: Org/team management + per-team allow-lists — Q1 2027
- Phase 12: EU AI Act Article 12 retention enforcement per org — Q1 2027
