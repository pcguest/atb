## Current state

ATB v1.12.0 ships a verified core bundle engine, six obligation profiles with CAS scoring, Go/Python/TypeScript SDKs, the EU AI Act retention guard, the `atb` CLI, MCP transport, and the `verify.report.v1` custody contract. The shipped runtime covers local capture, hash-chained bundle integrity, signing, encryption, TSA anchoring, WORM export, queue push, and corroboration event recording.

## Near term — Q3 2026

- Implement Phase 9 corroboration adapter wiring (OTel → bundle events)
- Implement GitHub audit log corroboration (pkg/corroborate/github)
- Implement LangChain/LangGraph corroboration (pkg/corroborate/langchain)
- Formalise obligation-profile DSL v1
- Produce verifier report v1 structured output
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
- Universal completeness guarantees (direct API bypass is a known gap)
- Training data governance (Articles 10–11)

## Custos

Custos is the planned commercial layer for central ATB bundle ingest, retention enforcement, auditor portal access, and policy gate dashboards. It is not yet implemented and is out of scope for this repository.

## Custos Enterprise Layer

- Phase 10: Ingestion engine scaffold (custos/ package tree) — Q3 2026
- Phase 10: AI tool discovery + registry — Q3 2026
- Phase 10: Onboarding flow + API key provisioning — Q3 2026
- Phase 10: Automatic bundle signing policy per org (key reference,
  TSA toggle, rotation schedule) — Q3 2026
- Phase 11: Human-in-the-loop review queue + oversight — Q4 2026
- Phase 11: Auditable work tree UI + handoff lineage — Q4 2026
- Phase 11: Insight extraction + pitfall detection — Q4 2026
- Phase 12: Org/team management + per-team allow-lists — Q1 2027
- Phase 12: EU AI Act Article 12 retention enforcement per org — Q1 2027
