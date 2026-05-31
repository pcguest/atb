## Current state

ATB v1.12.0 ships a verified core bundle engine, six obligation profiles with CAS scoring, Go/Python/TypeScript SDKs, the EU AI Act retention guard, the `atb` CLI, MCP transport, and the `verify.report.v1` custody contract. The shipped runtime covers local capture, hash-chained bundle integrity, signing, encryption, TSA anchoring, WORM export, queue push, and corroboration event recording.

## Completed — Phase 9 (Q3 2026)

- ✅ OTel translator scaffold (`pkg/otel`) — span structs mapped to canonical AI trace events (completed 28 May 2026)
- ✅ GitHub audit log corroboration (`pkg/corroborate/github`) (completed 28 May 2026)
- ✅ LangChain/LangGraph corroboration (`pkg/corroborate/langchain`) (completed 28 May 2026)
- ✅ Session index and actor grouping (added 28 May 2026 — completed out of roadmap order)
- ✅ UI session list and anomaly badge views (added 28 May 2026 — completed out of roadmap order)

## Near term — Q3 2026

- Phase 10: Transparent Proxy Capture (`atb intercept`) — local HTTPS MITM proxy that records AI API traffic into a live bundle
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
- Universal completeness guarantees (direct API bypass is a known gap until proxy capture ships)
- Training data governance (Articles 10–11)

## Custos

Custos is the in-repo reference receipt, custody, and attestation layer for ATB bundles. It lives under `custos/` as a separate Go module and is being scaffolded incrementally: the ingestion boundary, receipt store, per-org signing policy, and auth packages have unit tests today, while discovery, registry, onboarding, oversight, and insights are early scaffolds. Custos demonstrates how recorded bundles are ingested, signed, and held under custody — it is reference infrastructure, not a finished product.

Hosted, multi-tenant concerns — central auditor portal hosting, billing, SSO/RBAC, legal hold, and custodian-of-record operations — remain outside the ATB runtime and outside this repository, per `AGENTS.md`. The roadmap below tracks the in-repo reference layer only.

## Custos Enterprise Layer (in-repo reference)

- ✅ Phase 10: Ingestion engine scaffold (custos/ package tree) — Q3 2026 (completed 28 May 2026)
- Phase 10: AI tool discovery + registry — Q3 2026
- Phase 10: Onboarding flow + API key provisioning — Q3 2026
- Phase 10: Automatic bundle signing policy per org (key reference,
  TSA toggle, rotation schedule) — Q3 2026
- Phase 11: Human-in-the-loop review queue + oversight — Q4 2026
- Phase 11: Auditable work tree UI + handoff lineage — Q4 2026
- Phase 11: Insight extraction + pitfall detection — Q4 2026
- Phase 12: Org/team management + per-team allow-lists — Q1 2027
- Phase 12: EU AI Act Article 12 retention enforcement per org — Q1 2027
