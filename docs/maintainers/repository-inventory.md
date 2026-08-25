# Repository inventory

This inventory records the 2026-08-25 convergence pass against baseline commit
`192870aea2286717470c6703e398ec835d04553c`. The authoritative machine-readable
decisions are in [repository-inventory.json](./repository-inventory.json).

## Retained product surfaces

| Surface | Decision | Result |
| --- | --- | --- |
| CLI and Go implementation (`cmd/`, `internal/`, `pkg/`) | KEEP | One supported binary, private implementation packages, and an intentional public Go API. |
| Event and custody schemas (`schemas/`) | KEEP | Frozen contracts and reviewed digests remain the compatibility boundary. |
| Python and TypeScript SDKs (`sdk/`) | MERGE | Package-specific publishing/change notes collapsed into canonical READMEs and root release history; scoped legal files added. |
| Investigator viewer (`web/`) | KEEP | Local embedded review surface remains part of the product rather than a separate web product. |
| Test suites (`test/` and package tests) | KEEP | Golden, integration, schema, performance, security, and release contracts remain executable. |
| Automation (`.github/`, `Makefile`, `scripts/`) | MERGE | Six workflows and eleven scripts each have a distinct owner and purpose. |

## Canonical documentation tree

| Audience or task | Canonical location |
| --- | --- |
| First use and configuration | `docs/getting-started/` |
| Architecture, evidence, and trust concepts | `docs/concepts/` |
| Capture | `docs/capture/` |
| Incident investigation, verification, and tampering | `docs/investigate/` |
| Profiles, CAS, and keys | `docs/evidence/` |
| Normative formats and APIs | `docs/specification/` and `docs/api/` |
| Integrations and compliance | `docs/integrations/` and `docs/compliance/` |
| Release and repository maintenance | `docs/maintainers/` |
| Historical convergence evidence | `docs/releases/` |

## Applied reductions

Measured against the baseline tree, the effective source set fell from 761 to
711 files. Root files fell from 29 to 21, current Markdown documents from 55 to
39, scripts from 16 to 11, and example files from 28 to 10. The working diff
removes 11,904 lines while adding 4,212 lines of consolidated documentation,
tests, limits, and release controls. Test files increased from 230 to 256 as
adversarial and consumer-install coverage replaced demo and planning surface.

- Root launch checklists, readiness reports, release reviews, and UI audit notes
  were deleted; they described a moment in project development, not the product.
- Overlapping security, trust, RBAC, compliance, Mortise, release, and SDK
  publishing documents were merged into the canonical tree above.
- Four single-purpose workflows were deleted or merged so CI owns normal gates,
  release owns publication gates, and operations owns registry/triage automation.
- Five obsolete helper scripts and the old post-launch verifier name were removed;
  the remaining scripts are invoked by a documented Make or workflow contract.
- Three competing demo families, stale screenshots, and the unreferenced binary
  project-bootstrap fixtures were deleted. `examples/incident-demo/` is the sole
  end-to-end incident narrative; the quickstart and SDK snippets remain focused
  integration examples.
- Ignored private test keys, scratch bundles, duplicate generated bundles, local
  scanner reports, and Finder metadata were removed from the working tree.

Every non-KEEP decision above has been applied. Empty legacy directories are
not repository objects and disappear on checkout.

## Go package review

Every repository-owned package was reviewed for callers, API exposure,
duplication, product relevance, and test protection. Third-party Go sources
inside ignored `node_modules` trees are not repository packages.

| Package | Purpose and disposition | Test protection |
| --- | --- | --- |
| root module | Viewer/trust/document embeds; KEEP. | Embed and installed-binary tests. |
| `cmd/atb` | Sole CLI composition root; KEEP. | Command contracts, workflows, smoke, and golden exports. |
| `internal/agent` | Loopback capture-session service; KEEP. In-memory manager moved to `_test.go`. | Unit, file lifecycle, API, and concurrency tests. |
| `internal/anchor` | RFC 3161 request and token evidence; KEEP. | Token and bounded-response tests. |
| `internal/archive` | Archive operation and tamper-evident ledger; KEEP. | Unit and coverage tests. |
| `internal/bundle` | Authoritative bounded bundle reader/writer; KEEP. | Malformed, hostile, signature, locking, and external tests. |
| `internal/canonicalize` | RFC 8785 implementation; KEEP. | Golden, fuzz, and coverage tests. |
| `internal/capture` | Bounded chatlog import; KEEP. | Unit and external contract tests. |
| `internal/compliancepack` | Deterministic evidence mapping pack; KEEP, explicitly non-certifying. | Pack and external contract tests. |
| `internal/corroboration` | Bounded generic external-evidence HTTP adapter; KEEP. | URL, redirect, response-limit, and contract tests. |
| `internal/emit` | Canonical Go oversight event emitters; KEEP. | Unit and external schema tests. |
| `internal/encrypt` | Authenticated bundle encryption; KEEP. | Format and failure tests. |
| `internal/event` | Canonical event model and generated registry; KEEP/GENERATED. | External registry and generation parity tests. |
| `internal/evidence` | Structured single-bundle evidence summary; KEEP. | Unit and CLI tests. |
| `internal/evidencepack` | Verified multi-bundle evidence summary; KEEP. | Unit and CLI tests. |
| `internal/export` | Compliance/SOC2/GDPR archive primitives used by CLI; KEEP. | Covered through `cmd/atb` export tests. |
| `internal/hash` | Authoritative chain algorithm; KEEP. | Cross-language golden corpus and vectors. |
| `internal/identity` | Local API-key-to-actor resolver; KEEP. | External package tests. |
| `internal/identityevidence` | Explicitly unverified caller identity extraction; KEEP. | Incident, trust, and verifier consumer tests. |
| `internal/incident` | Deterministic findings and traceable reports; KEEP. | Unit, external, CLI, and flagship demo tests. |
| `internal/mcp` | Small stdio evidence tool server; KEEP. | Protocol and coverage tests. |
| `internal/mortise` | Optional bounded custody client; KEEP. | Response, redirect, URL, and receipt tests. |
| `internal/profiles` | Obligation profile DSL and built-ins; KEEP. | Loader, evaluator, fixture, and external tests. |
| `internal/proxy` | Privacy-minimising HTTPS capture boundary; KEEP. `StubHandler` renamed to `LoggingHandler` with compatibility alias. | Forwarding, auth, privacy, body-limit, and external tests. |
| `internal/push` | Explicit S3/Object Lock export boundary; KEEP. | Config and storage-contract tests. |
| `internal/retentionaudit` | Separate retention operations evidence ledger; KEEP. | Unit and coverage tests. |
| `internal/sessionindex` | Local multi-bundle investigator index; KEEP. | Unit and viewer API tests. |
| `internal/sign` | Local bundle signatures; KEEP. | Signature and coverage tests. |
| `internal/signer` | Local/remote/KMS signing backends; KEEP. | Backend, fallback, timeout, and provider tests. |
| `internal/trust` | Trust/CAS presentation models; KEEP. | Report and external tests. |
| `internal/verify` | Integrity, signatures, profiles, CAS, anchors, and reports; KEEP. | Broad unit, malformed, snapshot, and external tests. |
| `pkg/api/v1` | Supported authenticated local viewer API; KEEP. | Handler, auth, schema, sessions, and method tests. |
| `pkg/auth` | Session and OIDC/JWKS authentication; KEEP. | Token, redirect/outbound, and middleware tests. |
| `pkg/corroborate` | Published query-construction compatibility contract; KEEP for v1 compatibility, not a live verifier. | Adapter and integration tests. |
| `pkg/corroborate/github` | GitHub Audit Log query constructor; KEEP for v1 compatibility. | External adapter and integration tests. |
| `pkg/corroborate/langchain` | LangSmith query constructor; KEEP for v1 compatibility. | External adapter tests. |
| `pkg/custody` | Stable local bundle/receipt handoff contract; KEEP. | External package, Mortise conformance, and consumer tests. |
| `pkg/jcs` | Public RFC 8785 wrapper over golden-tested canonicalisation; KEEP. | Public wrapper tests plus internal goldens. |
| `pkg/otel` | Bounded OTLP/JSON semantic bridge; KEEP. Public test stub removed and unknown attributes preserved digest-first. | Decoder, translator, receiver, CLI, and integration tests. |
| `pkg/proxy` | Published proxy API aliases; KEEP. Logging handler has a compatibility alias. | Internal proxy suite and external command tests. |
| `scripts` Go package | Deterministic profile-fixture generator command; GENERATED support. | Exercised by `profile-fixtures` and Go gates. |
| `test/golden` | Cross-surface evidence invariants; KEEP. | Is the golden test package. |
| `test/integration` | Cross-component and cross-SDK contracts; KEEP. | Integration-tag gate. |
| `test/performance` | Bundle/load performance guard; KEEP. | Benchmark gate. |
| `test/release` | Release/version/workflow/legal invariants; KEEP. | Normal Go test gate. |
| `test/schema` | Schema/emitter compatibility; KEEP. | Normal Go test gate. |
