## Current state

The current source baseline (`v1.15.2` plus unreleased convergence work) ships
the agent-incident-forensics evidence path:
local proxy/SDK capture, session anomalies and incident reports, six obligation
profiles with CAS, `verify.report.v1`, optional reviewer identity evidence,
retention operations logging, and deterministic compliance evidence packs.
The v1.0 bundle format and canonical hash semantics remain unchanged.

## Completed — Phase 9 (Q3 2026)

- ✅ OTel OTLP/JSON input and translator (`pkg/otel`) — supported span data is
  mapped to canonical AI trace events; binary protobuf/gRPC collector transport
  and broader GenAI semantic-convention mapping remain backlog items
  (completed 28 May 2026)
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
- ✅ Mortise signs custody receipts (Ed25519 attestation of receipt, verifiable against the embedded key)
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
- ✅ Proxy auto-push to Mortise on session close (added 28 May 2026 — completed out of roadmap order)
- ✅ Formalise obligation-profile DSL v1 (completed 29 May 2026)
- ✅ Produce verifier report v1 structured output (completed 29 May 2026)
- ✅ Wire automatic capture to Claude and OpenAI SDK clients — opt-in
  `wrapOpenAI`/`wrapAnthropic` (TS) and `wrap_openai`/`wrap_anthropic` (Python)
  thin adapters that record the chat/messages `create` call (request, response,
  token usage, tool calls, errors) through the existing recorders; streaming and
  non-chat endpoints are documented blind spots (shipped June 2026)
- ✅ Enforce EU AI Act Article 12 logging in automatic capture path — every
  automatic capture surface now records the `atb.capture.scope` boundary
  attestation: the intercept proxy at startup (existing) and the SDK wrappers
  (`wrap_openai`/`wrap_anthropic`, `wrapOpenAI`/`wrapAnthropic`) at wrap time,
  with `capture_mode` derived from the privacy mode (shipped June 2026)
- ✅ Reviewer identity anchoring context — optional digest-only IdP assertion
  evidence on oversight/action events, surfaced as caller-provided evidence in
  verify, trust, incident, and SDK APIs (shipped June 2026)
- ✅ Retention enforcement access logging — policy set/change, local archive,
  and accepted S3 Object Lock request events in `.atb/operations.atb`
  (shipped June 2026)
- ✅ Automated compliance evidence pack export — deterministic, offline,
  profile-aware `atb compliance pack` with CAS, obligations, incident reports,
  mappings, checksums, and relevant retention evidence (shipped June 2026)

## Medium term — Q4 2026 to Q1 2027

- OTLP decode and GenAI semconv mapping (`pkg/otel`) — ◐ partial: GenAI semantic-convention
  mapping (`gen_ai.*` → canonical AI trace events), **OTLP/JSON** decode
  (`DecodeTraceJSON`, dependency-free, hex ids / unix-nano / AnyValue union /
  enum-as-int-or-string), and the OTLP/JSON **ingest path** (`Receiver.ReceiveJSON`
  + `atb import otel`, appending translated spans to a bundle with trace linkage
  preserved) shipped June 2026; OTLP/protobuf (gRPC) transport remains deferred as
  it would require an OpenTelemetry proto dependency
- DB reconciliation assurance packs
- Further CAS/provability-ladder formalisation

## Out of scope (explicit)

- Organisational SSO/RBAC and multi-tenant viewer operation (the local viewer
  has session-token authentication and optional OIDC role mapping)
- Hosted tracing or telemetry collection
- Real-time prevention or blocking of AI actions
- Universal completeness guarantees — `atb intercept` captures provider API traffic, but an agent that bypasses the proxy (or a direct in-process SDK call) is not seen; completeness is bounded by what flows through the recorder
- Training data governance (Articles 10–11)

## Mortise coordination

Mortise is the optional commercial custody and organisational layer. ATB's roadmap covers
only the frozen public contracts, optional client flows, and cross-repository
conformance evidence needed to keep that integration stable. Mortise product,
hosting, storage, IAM, witness, and auditor roadmaps remain outside the ATB
repository.

## Design notes (forward-looking — not commitments)

Trust boundaries and never-claims are in [public-surface.md](./public-surface.md).
Mortise implementation status is established in its own release material. The
items below are historical direction notes, not ATB release commitments:

- **Transparency-log compatibility** — keep ATB receipt and inclusion-proof
  contracts suitable for independently operated witness infrastructure
- **EU AI Act Article 12 mapping** — [compliance/eu-ai-act.md](./compliance/eu-ai-act.md)
- **Capture and custody scope** — each integration sees only routed traffic;
  see [public-surface.md](./public-surface.md)
