# Tenon / ATB / Mortise convergence master report

**Status:** synthesis of Lead decisions. Not an implementation. Not a freeze.
**Date:** 2026-09-01
**ATB HEAD:** `53ad440` — merge of [PR #11](https://github.com/pcguest/atb/pull/11), tag `v1.15.4`
**Working tree:** `v1.15.4-dirty` on `main` (tracks `origin/main`)
**Official freeze verdict:** **FAIL** (Trust J + Freeze L). v1.15.4 Docker recovery is complete and is a separate line.

Architecture is substantially complete. This report does not invent another one. The remaining work is cleanup, honesty, and productisation.

---

## 1. Executive verdict

ATB **v1.15.4 Docker recovery is COMPLETE**. PR #11 published Hub images for `linux/amd64` and `linux/arm64`. That line is frozen. It must not receive semantic or UI work.

ATB **v1.16 is NOT a freeze candidate**. Dirty 1.16 work sits unbranched on `main` (`git describe` = `v1.15.4-dirty`). Trust copy, the CAS contract, MCP advertising, and viewer claims still overclaim. Shipping this tree unchanged is an official freeze **FAIL**.

Architecture does not need another pass. Hashed event identity, dual wire families, profile evaluation, hash-chain verify, optional Mortise custody, and the dirty investigation IA already exist. Do not reopen them.

Mortise exists as private `pcguest/mortise` (tag `v0.5.0` plus later `main`). It pins ATB **1.14.3**. Local `/Users/paddyguest/custos` is not Mortise.

**F-MARKET-01 is CLOSED** (research, 2026-09-01). See `docs/research/`. Market validation: **PASS WITH NON-BLOCKING REFINEMENTS**. The locked IA survives. No architecture change. Do not blur ATB into Langfuse, LangSmith, or OpenTelemetry.

**First action after this file:** GROUP 0 — move every dirty 1.16 file off `main` onto `feat/v1.16.0-semantics` before any other commit.

---

## 2. Current repository state

| Fact | Evidence |
| --- | --- |
| Tagged source | `v1.15.4` in `cmd/atb/main.go` (`version = "1.15.4"`), `web/package.json`, `sdk/python/pyproject.toml`, `sdk/typescript/package.json`, `SECURITY.md` |
| HEAD | `53ad440` Merge pull request #11 from `pcguest/release/v1.15.4` |
| Describe | `v1.15.4-dirty` |
| Branch | `main` tracking `origin/main`. No `feat/v1.16.0-semantics` exists |
| Changelog | `CHANGELOG.md` `[Unreleased]` is empty (`<!-- No unreleased changes. -->`) while the working tree is not |
| Custody schema | `pkg/custody/schema.go` `VerifyReportSchemaVersion = "verify.report.v1.schema.2"` |
| Dirty contract | Working tree adds `integrity_valid`, `coverage_score`, `assurance_valid` on `VerifierReport` (`internal/verify/report.go`) without schema.3 |
| Docker | v1.15.4 Hub multi-arch proven. Out of scope for 1.16 |

Untracked 1.16 packages (KEEP, not debris): `internal/capabilities/`, `internal/contextlineage/`, `internal/mcp/evidence.go`, `pkg/api/v1/investigation.go` (+ types/tests), `web/app/view/components/CommandPalette.tsx`, `web/lib/schemas/investigation.ts`.

Modified 1.16 surfaces include event schema and generated bindings (`schemas/event.v1.json`, `internal/event/types_generated.go`, Python/TS generated files), verify/CAS, MCP server, OTel translator, investigation handlers, and the viewer (`web/app/view/page.tsx`).

v1.15.4 CI on `main` is green for the tagged line. Green tests on the dirty tree do not clear freeze: several encode the rejected `coverage_score`-on-invalid-chain behaviour.

---

## 3. Market findings

**F-MARKET-01 — CLOSED.** Bounded research 2026-09-01. Artefacts: `docs/research/MARKET_UI_PATTERN_MATRIX.md`, `PROFESSIONAL_UI_GRAMMAR.md`, `ATB_MARKET_UI_APPLICATION.md`. Official docs only (LangSmith, Langfuse, Phoenix, Braintrust, Weave, Galileo, Scale, Sentry, Splunk ES, Elastic Security, Datadog, Grafana, GitHub Security, Vanta; Sigstore/Rekor for custody). **MARKET VALIDATION: PASS WITH NON-BLOCKING REFINEMENTS.**

Workflow split in the market:

| Neighbourhood | User path | Primary object | ATB relation |
| --- | --- | --- | --- |
| Agent debug / eval (LangSmith, Langfuse, Weave, Galileo, Braintrust, Scale) | RUN → TRACE → EVALUATE → IMPROVE | Trace / experiment | **Adjacent. Do not copy the home screen.** |
| Incident / security (Sentry, Splunk ES, Elastic, GitHub Security) | ALERT/ISSUE → INVESTIGATION → TIMELINE → EVENT | Finding / issue | **Same job as ATB View.** |
| Assurance (Vanta) | CONTROL → TEST → EVIDENCE → GAP STATE | Control (not auditor opinion) | **Trust honesty, not a GRC app.** |

Consensus to **adopt** (professional convention): finding-first triage; dense tables; split list+inspector; human label then IDs; JSON on demand; named gaps (not a 0–100); ⌘K; persistent header; timeline/waterfall **before** topology graphs; colour never sole status.

Consensus to **reject**: trace-first home; graph-first sessions; flame/service maps; eval scores and playgrounds; NL investigation agents; SIEM query languages; message-trajectory dedup; urgency dashboards; “certified” slogans.

Timeline > graph is independently how Datadog (waterfall default vs map), Weave (tree default vs graph alternative), and Elastic (Timeline as the workspace) already work. Finding-first is how Sentry and GitHub Security work. Three-question Trust matches Vanta’s “control status is not the auditor’s assessment” plus named “Needs evidence” vs fail.

No Lead decision changed. No new P0 beyond the existing freeze-FAIL UI/trust items (Context honesty, kill Health/graph-as-home, coverage not shown on invalid chain, findings→evidence, palette a11y). P1: URL `surface`+`seq`, provenance badges, type filters.

Why ATB if you already have Langfuse/LangSmith: those products’ first object is a **trace you debug and score**. ATB’s first object is an **incident over a hash-chained bundle**. Why ATB if you have OTel: OTel is plumbing; ATB is the verifiable artefact. Why Mortise if you retain logs: logs are not verify-on-ingest + `custos.receipt.v1` + tlog. Why Tenon: GitHub/Grafana/Datadog-style family grammar (shared chrome, different primary object) — not a third dashboard.

---

## 4. Product positioning

Answerable in one minute:

```text
TENON — AI evidence assurance

        ATB                         MORTISE
  What happened?              What happened to the evidence?
  (evidence core)             (custody)

        TENON
  What does the evidence allow the organisation to establish?
```

| Product | Question | Job |
| --- | --- | --- |
| **ATB** | What happened? | Independently verifiable evidence of what the agent did, for later investigation. Local-first. MIT. No Mortise or Tenon cloud dependency. |
| **Mortise** | What happened to the evidence? | Independently verifiable organisational custody outside the originating system: verify-on-ingest, signed receipt, transparency log. Optional. |
| **Tenon** | What does the evidence allow the organisation to establish? | Family name and operational assurance narrative over evidence + custody. Not a third runtime. |

Why ATB if you already have Langfuse, LangSmith, or OTel: those help debug and improve; ATB is the artefact you can prove.

Why Mortise if you already retain logs: logs are not verify-on-ingest + signed receipt + tlog.

Why Tenon: family identity and honest limits, not another binary.

`README.md` already leads correctly: “ATB proves the integrity of what was recorded. It does not prove that every relevant event was captured.” Keep that sentence. Do not replace it with coverage percentages.

---

## 5. ATB freeze inventory

Summary of Lead KEEP / REFINE / CONSOLIDATE / MOVE / DEPRECATE. No wire rename.

| Surface | Current | Decision | Release |
| --- | --- | --- | --- |
| Hashed event identity / RFC 8785 chain | Shipped | KEEP | frozen 1.15.4 |
| Dual wire families (`ai.*` vs `atb.*` vs `data.export.*`) | Distinct types, disjoint evaluators | KEEP. Overlay, never aliases | 1.16 display only |
| `retrieval.performed` | `internal/capabilities` (untracked) | KEEP as derived overlay | 1.16 |
| `custos.receipt.v1` | `internal/mortise/client.go` `receiptVersion` | KEEP wire ID | frozen |
| `--custos` / `ATB_CUSTOS_TOKEN` aliases | CLI flags | KEEP until major | DEPRECATE, not delete |
| `cas_score = 0` on integrity fail | `internal/verify` | KEEP for old consumers | 1.16 contract |
| Completeness Assurance Score | `docs/evidence/cas.md`, `ComputeCAS` | KEEP name. Never “Capture” | frozen name |
| RECORD / EXPECT / PROVE | Implicit in append / profile / verify | Overlay on docs/UI, not new event types | 1.16 docs |
| PageIndex | `sdk/python/atb/pageindex.py` extra | KEEP as adapter | 1.16 honesty |
| Dirty IA | Incident → Findings → Timeline → Context → Relationships → Evidence → Trust | KEEP shell | 1.16 UI |
| Docker 1.15.4 | PR #11 Hub multi-arch | KEEP frozen. No semantic mix-in | 1.15.4 |
| `data_export` docs vs YAML | `bundle-v1.md` claims `ai.action.*`; YAML requires `data.export.*` | REFINE docs to YAML | GROUP 1 |
| MCP docs | `docs/integrations/mcp.md` still `2024-11-05` | REFINE dual-era. Do not claim 2026-07-28 complete | GROUP 1 + NEXT |
| Trust UI | Health score / Dashboard leftovers | REFINE to three questions | 1.16 must-fix |
| Context heading | “Context supplied to model” | REFINE heading; bind required | **TB-UI-01** |
| `coverage_score` on invalid chain | Computed at 0.9 in tests | REFINE: omit or `untrusted` when `integrity_valid=false` | **F-CAS-01** |
| OTel `atb.event_type` | Taxonomy authority | REFINE allowlist | **F-OTEL-01** |
| MCP RAG query | Plaintext required | REFINE: digest default, plaintext opt-in | **F-MCP-02** |
| Display event-family maps | `web/lib/event-family.ts` vs `investigation.go` | CONSOLIDATE display only | GROUP 5 |
| Mortise clients | Go `internal/mortise`; Py/TS thinner | CONSOLIDATE clients, not products | NEXT |
| Evidence-pack names | “compliance pack” vs custody pack | CONSOLIDATE naming | docs |
| Tenon tokens | `verified` / `warning` / `danger` / `unknown` in `web/tailwind.config.ts` | CONSOLIDATE from existing ATB colours | 1.16 UI |
| Dirty 1.16 files on `main` | Unbranched | MOVE to `feat/v1.16.0-semantics` | **GROUP 0 / F-PROC-01** |
| `--custos`, `sdk/python/atb/cli.py`, langchain shim | Live shims | DEPRECATE until major | GROUP 7 |
| `atb_cli_stub*` glob | `sdk/python/pyproject.toml` line 43, matches zero files | DELETE glob only | GROUP 2A |
| Wire names | Dual families, receipt ID, CAS keys | RENAME: none. Additive aliases only | — |

---

## 6. Deletion candidates

Nothing enters DELETE unless ownership, references, compatibility, tests, docs, and migration all pass.

**DELETE now (six-point PASS):** the unused setuptools include `atb_cli_stub*` in `sdk/python/pyproject.toml`. Change `include = ["atb*", "atb_cli_stub*"]` to `include = ["atb*"]`. Do not delete `sdk/python/atb/cli.py`.

**Never delete in v1.16:**

- `custos.receipt.v1`
- Dual event families and generated bindings (`internal/event/types_generated.go`, Python/TS generated files)
- `--custos*` aliases and `ATB_CUSTOS_TOKEN`
- `internal/mcp/evidence.go` — unused is not dead; KEEP and wire
- Viewer Health leftovers (`web/lib/trust-score.ts`, `web/app/view/loading.tsx`, `StatsOverview`, Cypress Trust Dashboard specs) until the UI replace ships
- `sdk/typescript/src/eventTypes_generated.ts` (generate-gate artefact)
- `scripts/smoke-view.sh`, `scripts/verify-published-release.sh`, `scripts/verify-container-architecture.sh`
- `pkg/corroborate`, `StubHandler`, `web/out/placeholder.txt`, `docs/releases/v1.15.3-convergence.md`
- Local `/Users/paddyguest/custos` (not in this repo; not Mortise)

**Document, do not delete:** `atb serve` (stdio MCP alias), `atb identity`, `python -m atb.cli`, langchain shim.

---

## 7. Evidence semantic convergence

Source of truth: `schemas/event.v1.json` → generated bindings.

**KEEP dual families.** These are not aliases:

| Concept | Family A | Family B |
| --- | --- | --- |
| Approval | `ai.human.approval` | `atb.human.approval` |
| Retrieval | `ai.retrieval.executed` | `atb.event.rag_retrieval` |
| Export | `data.export.precommit` / `data.export.executed` | `atb.data.export` |
| Tool / action | `ai.action.*` | `atb.tool.call` |

Disjoint evaluators consume them (profiles vs session-index findings vs PageIndex/MCP). Overlay with derived capabilities. Never rewrite history. Never retarget MCP `rag_retrieval_record` onto `ai.retrieval.executed`.

`retrieval.performed` already exists in `internal/capabilities` (`MappingVersion = "atb.capability.v1"`). It maps both retrieval families. It is an investigation aid, not a substitute record.

RECORD / EXPECT / PROVE is the epistemic overlay, not a type system:

- **RECORD** — hashed events in the bundle
- **EXPECT** — obligation profiles (`internal/profiles/templates/*.yaml`)
- **PROVE** — verify + CAS + incident findings over recorded evidence

**F-SEM-01:** `atb.profile.data_export` YAML requires `data.export.precommit` / `data.export.executed` (`internal/profiles/templates/data_export.yaml`). `docs/specification/bundle-v1.md` and `docs/specification/events.md` still describe `ai.action.*` as that profile’s critical path. Session inference keys on `atb.data.export`. YAML wins. Docs-only in GROUP 1; one vocabulary for inference is NEXT, not a wire merge.

New dirty types (`ai.context.unit`, `ai.context.operation`, `atb.mcp.operation`) are in `event.v1.json` and absent from `docs/specification/events.md`. Catalogue them after GROUP 0. `privacy.reveal` remains sidecar-documented, not a bundle-required type.

Hashed identity stays. `identity_evidence` remains `caller_provided_unverified` (`internal/identityevidence`). `clientInfo` / `serverInfo` on MCP initialize stay non-identity.

---

## 8. Profile / CAS convergence

**Name:** Completeness Assurance Score. Frozen. Not Capture (collides with the non-claim that ATB does not prove complete capture). Not content-addressed storage.

Integrity: **VALID | INVALID**. On integrity fail, `cas_score = 0` stays for old consumers.

Coverage is additive `verify.report.v1.schema.3`. `pkg/custody/schema.go` is still schema.2 with `additionalProperties: false`. Emitting `integrity_valid` / `coverage_score` on `VerifierReport` today is **F-CONTRACT-01**. Cut schema.3 (or strip the fields) before those fields ship.

**F-CAS-01 (must-fix):** `coverage_score` is still computed when integrity is false (tests lock `CoverageScore = 0.9`). Lead lock: omit the field or mark `untrusted`. Do not show a percentage when `integrity_valid=false`. Do not retarget `cas_score` to coverage.

ReviewReady ≈ `chain_valid ∧ profile_pass`. Do not invent a third score for freeze.

Do **not** implement DSL v2.

Do **not** flip `required_when` / approval-after-execute in v1.16. Privileged profile YAML uses `at_or_after: true` on execute. Document the shipped rule as **`attestation_after_execute`**. Incident preceding-approval (`atb.human.approval` before `atb.tool.call`) remains a different story; do not silently unify them.

RAG `cas_score` still includes GC = 0.3 (`internal/verify/profiles.go` `ragAnswerSubScores`). That stays for historical Overall compatibility. The coverage path must not treat GC as gating proof.

**F-CAS-02** (missing timestamps → TC = 1.0): required for true PASS; may remain dated debt under “PASS WITH DOCUMENTED DEBT” only if explicitly owned. Preferred 1.16 refine: TC unassessable when timestamps are absent. Do not call TC “causal.”

Custom DSL SC assessable-at-0 is verifier REFINE, not a delete.

---

## 9. MCP convergence

stdio **KEEP**. `atb mcp serve` / `atb serve` stay.

`internal/mcp/server.go`:

```go
ProtocolVersion = "2026-07-28"
LegacyProtocolVersion = "2024-11-05"
```

Discover currently emits `protocolVersions`, ignores `_meta`, has no `supportedVersions`, no response `_meta`, no `resultType`. `BuildOperationEvent` in `internal/mcp/evidence.go` is tested and never called from `server.go`.

**F-MCP-01:** stop advertising 2026-07-28 as complete until `_meta`, `supportedVersions`, and `resultType` exist, and `BuildOperationEvent` is wired. GROUP 1 docs: dual-era handshake only. Wiring is NEXT on the 1.16 branch. Do not delete `evidence.go`.

**F-MCP-02:** `rag_retrieval_record` requires and stores plaintext `query`. Default to `query_digest`; plaintext opt-in. Drop “reasoning-based” and unchecked “foreign key” `index_id` claims.

MCP is a semantic capture boundary, not ATB’s architecture and not a generic MCP observability layer.

---

## 10. PageIndex / RAG convergence

PageIndex is an **adapter**. Canonical retrieval evidence is ATB events. The Python extra (`sdk/python/atb/pageindex.py`, optional extra in `pyproject.toml`) appends `atb.event.rag_retrieval` via CLI. That is allowed. It does not make PageIndex the evidence model.

Do not retarget PageIndex/MCP onto `ai.retrieval.executed`.

Profile `rag_answer` SC/TC that look only at `ai.retrieval.executed` must either credit both families **or** docs must say PageIndex does not satisfy that obligation. Overlay `retrieval.performed` is the investigation join, not a new required event.

MCP tool text must not say “reasoning-based retrieval.” ATB has no chain-of-thought type (`internal/contextlineage` package comment). Producer payloads (`node_summary`, model output) may still contain reasoning text; copy must say so.

---

## 11. Context Lineage

`internal/contextlineage` is a projection over `ai.context.unit` / `ai.context.operation`. It does not execute retrieval, cache, compaction, or assembly. It has no CoT representation. Provenance enum: `asserted` / `derived` / `attested`.

**F-LINEAGE-01:** `contextlineage.Build` ignores RAG / PageIndex retrieval events. Production emitters (`atb.event.rag_retrieval`, `ai.retrieval.executed`, OTel retrieval, MCP RAG) never enter lineage. Operations copy `invocation_id` if present and never bind it to `ai.model.invoked.prompt_digest`.

Lead rules for freeze:

- No heading “supplied to model” without `invocation_id` ↔ `prompt_digest` bind (**TB-UI-01**)
- Empty lineage ≠ no retrieval (**TB-UI-02**). Render capabilities or “no `ai.context.*` records”; never “no context was captured”
- Lineage consumes retrieval **or** the Context UI cannot imply model-supply (true PASS). For PASS WITH DOCUMENTED DEBT, honest heading + capabilities display is the bar

---

## 12. Metadata model (ASSERTED / DERIVED / ATTESTED)

Already on context units (`internal/contextlineage.MetadataProvenance`). Retrieval and MCP paths do not feed it.

| Class | Meaning | Examples |
| --- | --- | --- |
| **ASSERTED** | Producer said so; hashed, not independently verified | actor fields, `identity_evidence`, `atb.capture.scope`, OTel `privacy.mode`, MCP `clientInfo` |
| **DERIVED** | ATB computed from recorded events | `retrieval.performed`, incident findings, CAS scores, investigation `family` |
| **ATTESTED** | Independent check against recorded material | hash-chain VALID, configured signature match, RFC 3161 imprint check, Mortise receipt verify-on-ingest |

UI and reports must label derived overlays as derived. Capabilities must not look like extra events. Profile pass is EXPECT evaluation, not ATTTESTED capture completeness.

`atb.capture.scope` “absence of an event is evidence of scope” is an asserted completeness claim. Document it as recorder assertion, not proof.

---

## 13. Incident / findings model

Findings are deterministic observations about **recorded** evidence (`internal/incident/findings.go`). Each finding already carries `what_atb_can_conclude` / `what_atb_cannot_conclude` / `boundedness`. Keep that shape.

Session index remains the flag authority; a Finding is the located explanation.

Empty-state honesty: Findings copy already says absence does not prove complete capture. Incident empty-state (“No investigation findings were identified in the captured evidence”) does not. Use the Findings caveat on Incident (**TB-UI-03**). Default surface is Incident; that sentence will be screenshotted.

Do not treat “no matching approval in the recorded evidence” as “no approval existed.”

Approval-after-execute vs preceding-approval vs human-override HITL are three causal stories. Document; do not flip in v1.16.

---

## 14. ATB UI convergence

**KEEP** the dirty 1.16 investigation IA:

**Incident → Findings → Timeline → Context → Relationships → Evidence → Trust**

That shell is the freeze candidate. The committed v1.15.4 three-rail “Trust Dashboard” (timeline / graph / inspector + Viewer Health 0–100) is rejected as the product grammar. Do not revert.

**Kill:**

- Viewer Health score (`web/lib/trust-score.ts`, `TrustScoreCard`, `loading.tsx` “Loading Trust Dashboard”)
- Trust Dashboard chrome (`web/cypress/e2e/live-dashboard.cy.ts`, `trust-score.cy.ts`, `dashboard.cy.ts`)
- Graph-first Relationships (dirty tree still leads with a ~420px graph)

Health leftovers stay in tree until the replacement Trust surface ships. Then delete.

**Trust as three questions**, green/red named checks only — not a 92/100:

1. Is the recorded chain intact? (`integrity_valid` / hash chain VALID\|INVALID)
2. Does the selected profile pass over recorded evidence? (`profile_pass`)
3. Is organisational custody recorded? (local vs Mortise receipt recorded — never “immutable”)

No “supplied to model” without bind. No empty lineage as universal non-retrieval. Label coverage “profile-scoped coverage (not capture completeness)” and hide the percentage when integrity is invalid.

Mortise in ATB View is a **custody row/link**, not a second investigation app.

Workspace copy “Evidence remains immutable on disk” (`web/app/workspace/page.tsx`) is rejected. Tamper-evident local files; not immutable storage.

Command palette and `/api/v1/investigation/*` ship with the 1.16 UI. They are not debris.

---

## 15. Mortise convergence

Mortise is private `pcguest/mortise`: tag **v0.5.0** (still historically `custos-*` in that tag) and later `main` renamed to `mortise-*` with `/verify/*`. ATB pin on that product: **1.14.3**. ATB tree is **1.15.4**. **F-MORTISE-01** / **TB-MORT-01**.

What Mortise already is (from H; do not duplicate in ATB): verify-on-ingest, WORM object store, signed `custos.receipt.v1`, timestamps, transparency proofs, witnesses, org-scoped API keys.

What ATB already is and must remain: capture, chain, local verify, profiles, CAS, incident, local viewer. `internal/mortise` is an optional HTTP client.

**Receipt wire ID stays `custos.receipt.v1`.** `--custos` aliases stay.

Local `/Users/paddyguest/custos` is a Phase-1 daemon with a different API. It is not Mortise and not ATB debris. Do not inventory it as cleanup. ATB docs/CHANGELOG that still describe custosd HTTP as current are GROUP 1 honesty.

Evidence-pack naming:

- **ATB pack** — portable evidence export (`internal/evidencepack`, compliance-pack CLI name retained as packaging label, not certification)
- **Mortise custody pack** — organisational receipt + tlog proof about that evidence

SSO / RBAC / WORM enforcement are Mortise operator capabilities. ATB docs may say “if the custody operator provides.” They must not read as ATB features or as proven Mortise SKUs without Mortise evidence (**TB-DOC-02**).

Shared content-address vs tenant isolation is a Mortise 1.16 concern (**TB-MORT-02**), not an ATB redesign.

Dirty `VerifierReport` extra fields would fail schema.2 consumers (`test/mortise/conformance_test.go`). Schema.3 is a Mortise compatibility event.

---

## 16. Tenon design system

Family grammar only. Tokens come from **existing ATB semantic colours**, not a new palette.

`web/tailwind.config.ts` already has `verified`, `warning`, `danger`, `unknown` mapped to `--state-*`. `web/app/globals.css` holds the CSS variables. Tenon/Mortise reuse those names.

| Product | Header question | Feel |
| --- | --- | --- |
| ATB View | What happened? | Flight recorder / incident console / evidence viewer |
| Mortise | What happened to the evidence? | Custody register / receipt / tlog |
| Tenon | What does the evidence allow the organisation to establish? | Assurance narrative; no third app chrome |

Four named statuses (Design I): VERIFIED / FAILED / INCONCLUSIVE / UNTRUSTED. Dirty investigation overview is only VERIFIED / FAILED today. Map the other two before freeze UI ships; do not use unnamed greens.

No decorative trust indicators. No 0–100 health. No observability graphs as the home view. Landing marketing chrome must not leak into `/view`.

Market research (F-MARKET-01) confirms: do not copy Langfuse/LangSmith **home** grammar. Copy Sentry/GitHub **finding** grammar and Elastic/Datadog **timeline-before-map** grammar. Visual language stays Tenon tokens (`verified` / `warning` / `danger` / `unknown`).

---

## 17. Security / trust review

**Official: FAIL if shipped unchanged.** Trust J is accepted as freeze-FAIL, not documented debt.

### Must-fix before PASS WITH DOCUMENTED DEBT

| ID | One line |
| --- | --- |
| **TB-UI-01** | Context heading “Context supplied to model” without `invocation_id` ↔ `prompt_digest` bind (`web/app/view/page.tsx`) |
| **TB-UI-02** | Empty lineage treated as no retrieval; capabilities overlay unused; Health / Trust Dashboard leftovers still ship |
| **F-CAS-01** / **TB-CAS-01** | `coverage_score` emitted on invalid chain |
| **F-MCP-01** / **TB-MCP-01** | Protocol 2026-07-28 advertised without `_meta` / `supportedVersions` / `resultType`; `BuildOperationEvent` unwired |
| **F-MCP-02** / **TB-MCP-02** | Plaintext query required on MCP RAG |
| **F-CONTRACT-01** | schema.2 `additionalProperties: false` vs new report fields |
| **F-PROC-01** | Dirty 1.16 unbranched on `main` |

### 1.16 code REFINE (Lead-locked)

| ID | One line |
| --- | --- |
| **F-OTEL-01** / **TB-OTEL-01** | `atb.event_type` allowlist; not taxonomy authority (`pkg/otel/translator.go`) |
| **TB-OTEL-02** | Privacy mode does not hash mapped prompt/query duplicates in `otel_attributes` |
| **F-LINEAGE-01** | Lineage ignores retrieval families |

### Held (do not “fix” into overclaims)

Hash chain ≠ capture completeness. `identity_evidence` not verified. MCP `clientInfo` is not identity. Reveal does not mutate the bundle (sidecar `.reveals`). Viewer role selector is presentation-only (`web/lib/roles.ts`). Timeline `causal_edge` is always false. `custos.receipt.v1` frozen. CAS not renamed to Capture.

### Overridden (do not reopen)

- **TB-CAS-04** wanted to zero RAG GC in Overall. Lead: keep GC=0.3 in `cas_score`; coverage path must not treat it as gating.
- **TB-PROF-02** wanted to flip approval-after-execute. Lead: document as `attestation_after_execute`; do not flip in v1.16.

---

## 18. Compatibility review

| Contract | Rule |
| --- | --- |
| Event types | No rename. Additive types only. Dual families stay |
| `verify.report.v1` | schema.2 is current on HEAD. schema.3 required before coverage fields |
| `cas_score` / `cas_grade` | Names frozen. `cas_score=0` on integrity fail frozen |
| `--format json` vs `--json` | `--format json` = stable `VerifierReport`; `--json` = diagnostic. Docs mix them (`docs/getting-started/quickstart.md` `residual_risk` as string vs object) |
| `--custos*` | Additive aliases until major |
| MCP tool names | `rag_retrieval_record`, `rag_index_record`, `verify`, `status`, `atb_init` stay |
| Encrypt magic / `.atb` NDJSON | Untouched |
| Help JSON | Omits live `serve`, `identity`; text help omits `intercept`. Honesty REFINE, not removal |
| SDKs | Independent Go / Python / TypeScript implementations; generated catalogues must stay in lockstep (`make check-generated`) |
| OpenAPI | `docs/api/openapi.yaml` `info.version` still `1.12.0` vs product `1.15.4` |
| Mortise pin | 1.14.3 vs ATB 1.15.4; schema.3 is a consumer event |
| Docker | 1.15.4 multi-arch is proven and frozen |

Historical-wire compatibility is intentional debt. “Clean” does not mean zero shims.

---

## 19. Implementation plan

**GROUP 0 is mandatory before any other commit.** Implementation has not started. This file does not start it.

### GROUP 0 — MOVE (blocker F-PROC-01)

Park uncommitted 1.16 work on `feat/v1.16.0-semantics`. Restore `main` to tagged `v1.15.4` (`53ad440`). Suggested:

```text
git switch -c feat/v1.16.0-semantics
# commit KEEP-for-1.16 files on that branch only
git switch main
git restore --worktree --staged .
```

Untracked KEEP: capabilities, contextlineage, `internal/mcp/evidence.go`, investigation API, CommandPalette, `web/lib/schemas/investigation.ts`.

Modified KEEP: event types/bindings, findings, MCP server, verify/CAS, API handlers/types, OTel translator, `schemas/event.v1.json` (+ sha256), Python pageindex, viewer/CSS/roles/schemas.

Treat all of the above as 1.16 product, not accident.

### GROUP 1 — DOCUMENT on clean `main` as unreleased 1.16 commits

After GROUP 0. Do not tag 1.15.4. Do not claim schema.3 or MCP 2026-07-28 complete.

| File | Fix |
| --- | --- |
| `docs/specification/bundle-v1.md` | `data_export` → `data.export.*` per YAML |
| `docs/specification/events.md` | Profile column + new context/MCP types |
| `docs/specification/verify-report.md` | schema.2 current (schema.3 only after GROUP 6) |
| `docs/maintainers/automation-contract.md` | `residual_risk` object; `--format json` vs `--json` |
| `docs/integrations/mcp.md` | Dual-era `2026-07-28` / `2024-11-05`; incomplete modern fields |
| `sdk/python/README.md` | Complete event table; keep families distinct |
| `docs/api/openapi.yaml` / template | `info.version` 1.15.4 |
| `README.md` | Docker Hub multi-arch; `pkg/api/v1` is viewer HTTP, not a product SDK |
| `docs/getting-started/quickstart.md` | `residual_risk` object shape |
| CHANGELOG historical custosd | Point at Mortise client + frozen receipt ID |

### GROUP 2A — DELETE glob

`sdk/python/pyproject.toml`: drop `atb_cli_stub*`.

### GROUP 2B — DOCUMENT scripts

`scripts/smoke-view.sh`, `scripts/verify-published-release.sh`: owners in maintainer docs. Do not delete.

### GROUP 3 — DOCUMENT generated TS

`sdk/typescript/src/eventTypes_generated.ts` is a generate-gate artefact. Do not delete.

### GROUP 4 — DEPRECATE / help honesty

Add `intercept` to text help. Document `identity` / `serve` as internal. Keep custos aliases, langchain shim, `cli.py` until major.

### GROUP 5 — CONSOLIDATE display + ship UI replace

One display event-family module. Kill Health / graph-first **as the shipped surface** (code may linger until replace). Trust three questions. Context heading. Findings empty-state on Incident. Tenon tokens from existing semantic colours.

### GROUP 6 — schema.3 (F-CONTRACT-01)

Freeze `verify.report.v1.schema.3` **before** coverage fields on public `VerifierReport`. Pin SHA in `pkg/custody`. F-CAS-01 omit/untrusted. Keep `cas_score=0` on integrity fail.

### GROUP 7 — AFTER FREEZE / major

Delete `sdk/python/atb/cli.py` only at a major. Keep `--custos` until major.

### J / L must-fix on the 1.16 branch (with GROUPS 5–6)

Wire or stop advertising MCP 2026-07-28 (**F-MCP-01**). Plaintext query opt-in (**F-MCP-02**). OTel `atb.event_type` allowlist (**F-OTEL-01**). Context bind/capabilities (**TB-UI-01/02**). Coverage omit/untrusted (**F-CAS-01**).

### Required regression tests (programme §9 + J/L)

Every semantic change ships with tests:

- event → capability mapping (`internal/capabilities`)
- PageIndex → `atb.event.rag_retrieval` adapter (not `ai.retrieval.executed`)
- context input/output digest binding; no “supplied to model” without bind
- compaction / assembly lineage (projection only; no CoT)
- MCP trace binding; `BuildOperationEvent` called or version claim dropped
- MCP cache semantics (no invented cache-as-evidence)
- asserted vs derived vs attested labels
- unassessable CAS dimensions; AssessmentCoverage
- integrity-invalid + coverage omitted/untrusted (**F-CAS-01**)
- critical obligation failure
- historical event and profile compatibility (dual families)
- Mortise ATB ingestion / `test/mortise` against schema.3
- privacy reveal sidecar (bundle unchanged)
- UI semantic-state rendering (three questions; no Health score)
- OTel allowlist; plaintext query opt-in
- `atb intercept` in text help; usage contract
- Python wheel has no `atb_cli_stub`

---

## 20. Freeze acceptance criteria

v1.15.4 Docker recovery is already complete. It is **not** this freeze.

ATB 1.16 may be called freeze candidate only when L’s **flip to PASS WITH DOCUMENTED DEBT** list is closed:

1. **F-PROC-01.** 1.16 is on `feat/v1.16.0-semantics` (or equivalent). `main` at tagged `v1.15.4` is clean.
2. **F-CONTRACT-01.** schema.3 cut, or new fields stripped from `VerifierReport`.
3. **F-CAS-01.** `coverage_score` omitted or `untrusted` when `integrity_valid=false`.
4. **TB-UI-01.** Context heading is not “supplied to model” unless bind is proven.
5. **TB-UI-02.** Empty lineage is not “no retrieval”; Health / Trust Dashboard is not the shipped surface.
6. **F-MCP-01 advertising.** Do not claim 2026-07-28 complete without `_meta`, `supportedVersions`, `resultType` (wire `BuildOperationEvent` or stop advertising).
7. **F-MCP-02.** RAG query plaintext is opt-in; default digest.
8. Remaining dual-family names, Custos aliases, `atb serve`, PageIndex-as-adapter, lineage-not-RAG, OTel subset, Mortise pin lag: **dated, owned debt** — not “later.”
9. `CHANGELOG.md` `[Unreleased]` and version markers describe 1.16, or the work does not claim freeze.

Programme §10 (intentional structure, inventoried contracts, coherent vocabulary, docs match reality, CI, clean-clone examples, Mortise consumes without duplicating, trust review) is evaluated **on the 1.16 branch after those items**, not on dirty `main`.

**True PASS** additionally requires: **F-CAS-02** (TC not 1.0 on missing timestamps); **F-SEM-01** one displayed vocabulary without wire merge; lineage consumes retrieval or Context cannot imply model-supply; Mortise consumes ATB 1.15.4+ / schema.3 without a parallel Custos product; gold gate + clean-clone examples; help JSON matches the command surface.

**F-MARKET-01** is closed (2026-09-01). Market validation does not add freeze blockers. GROUP 5 implements the already-locked UI plus P0 items listed in `docs/research/ATB_MARKET_UI_APPLICATION.md`.

Trust J + Freeze L remain **FAIL** until the documented-debt list is done in code, not in footnotes.

---

## 21. Deferred work

Hard stop after freeze criteria would pass. Remaining ideas:

- DSL v2
- Wire-merge of dual event families (never in 1.16; major+migration if ever)
- Flipping approval-after-execute to pre-exec HITL
- Zeroing RAG GC in historical `cas_score`
- Filling empty `required_fields` as runtime hard-fail
- Productising `atb identity` / `atb serve` in public help JSON
- Tenant-prefixed Mortise object keys / encrypt-then-hash (**TB-MORT-02**)
- Mortise SSO / billing / legal-hold screens
- Market P1/P2 (URL surface+seq, provenance badges, type filters) after GROUP 5 P0
- Vector DB, memory platform, agent runtime, SIEM, workflow orchestrator
- Tenon cloud dependency in ATB
- CoT schema
- Generic observability expansion
- Compliance scoring / certification productisation of CAS
- Deleting `--custos` or `cli.py` before a major
- Replacing hashed event identity

---

## 22. Explicit non-goals

- Inventing another architecture
- Semantic or UI work on the v1.15.4 Docker line
- Mixing dirty 1.16 into a 1.15.4 hotfix or tag
- Renaming anything on the wire
- Claiming MCP 2026-07-28 complete
- Claiming CAS as capture completeness or external assurance
- Treating empty lineage as proof of non-retrieval
- Saying “supplied to model” without bind
- Treating `/Users/paddyguest/custos` as Mortise
- Building a second investigation app inside Mortise
- Blurring ATB into Langfuse, LangSmith, or OTel
- Implementing this report in the same change as GROUP 0 (GROUP 0 is first and only first)

---

## Decision graph

| Finding | Evidence | Agents | Conflict | Decision | Reason | Release |
| --- | --- | --- | --- | --- | --- | --- |
| Dirty 1.16 on `main` | `v1.15.4-dirty`; untracked capabilities/lineage/investigation | A, K, L | Convenience commit vs freeze line | MOVE all dirty files to `feat/v1.16.0-semantics` first | 1.15.4 Docker line must stay clean | GROUP 0 / **F-PROC-01** |
| Dual event families | Four concepts × two wire types; disjoint evaluators | C, D, J | Docs speak as if aliases | KEEP distinct; overlay only | Historical bundles; no migration | 1.16 display / docs |
| `retrieval.performed` | `internal/capabilities` | C, E, K | Looks like a new type | KEEP derived overlay | Join without rewrite | 1.16 |
| CAS name Capture vs Completeness | Repo: Completeness (`cas.md`, `ComputeCAS`) | D vs programme brief | Brief said Capture | Completeness Assurance Score | Capture collides with non-claim | frozen name |
| `cas_score=0` vs coverage | Integrity fail still scores coverage 0.9 | D, J, L | Compatibility lie vs new field | Keep `cas_score=0`; omit/untrusted coverage | Old consumers; no false % | **F-CAS-01** + schema.3 |
| RAG GC=0.3 | `ragAnswerSubScores` | D, J | J wanted zero GC | Keep in `cas_score`; coverage must not gate on it | Compatibility | 1.16 coverage path |
| Approval-after-execute | `required_when` `at_or_after` | D, J | Incident wants preceding approval | Do not flip; document `attestation_after_execute` | Shipped profile rule | 1.16 docs |
| data_export docs | YAML `data.export.*` vs `bundle-v1.md` `ai.action.*` | C, K, L | Docs vs YAML | YAML wins; docs-only | BLOCKER-2 | GROUP 1 / **F-SEM-01** |
| MCP 2026-07-28 | Constant vs missing `_meta` / `supportedVersions` / `resultType` | E, J, L | Advertise vs implement | Stop advertising complete; KEEP stdio; wire `BuildOperationEvent` | Honesty | **F-MCP-01** |
| MCP plaintext query | Tool requires `query` | E, J, L | Privacy vs capture | Digest default; plaintext opt-in | Trust | **F-MCP-02** |
| `evidence.go` unused | Never called from `server.go` | E, K, L | Delete vs wire | KEEP and wire | Not dead | 1.16 NEXT |
| PageIndex | Python extra emits `atb.event.rag_retrieval` | E | Merge onto `ai.retrieval.*` | Adapter only | Dual families stay | 1.16 |
| Context “supplied to model” | `page.tsx` heading; lineage unbound | G, J, L | Copy vs bind rule | Kill phrase unless bind | Freeze FAIL | **TB-UI-01** |
| Empty lineage | “No context lineage was captured” | G, J | Absence vs non-retrieval | Capabilities / precise empty copy | Freeze FAIL | **TB-UI-02** |
| Viewer Health / Dashboard | `trust-score.ts`, Cypress, `loading.tsx` | G, I, K, L | Score vs three questions | Kill shipped surface; leftovers until replace | False certainty | 1.16 UI |
| Dirty IA | Incident…Trust nav | G | Revert to 1.15.4 rails | KEEP dirty IA | Investigation grammar | 1.16 |
| OTel `atb.event_type` | Translator uses it as type | C, J, L | Hint vs authority | Allowlist | Profile PASS from hostile spans | **F-OTEL-01** |
| Receipt ID `custos.receipt.v1` | `internal/mortise/client.go` | B, H | Rename to mortise.receipt | KEEP | Historical wire | frozen |
| Local custos | `/Users/paddyguest/custos` | H, K | Sibling “debris” | Not Mortise; not ATB cleanup | Different API | never |
| Mortise pin 1.14.3 | H clone of `pcguest/mortise` | H, J, L | Re-pin now vs ATB freeze | Dated debt; schema.3 is the consumer event | ATB freeze ≠ Mortise freeze | **F-MORTISE-01** |
| Market matrix | F-MARKET-01 research 2026-09-01 | F (closed), I, L | Copy LangSmith home vs Sentry finding-first | KEEP locked IA; adopt finding/timeline/inspector conventions; reject trace-first and scores | Market PASS WITH NON-BLOCKING REFINEMENTS | GROUP 5 (no new P0) |
| `atb_cli_stub*` | Zero files | A, K | Harmless glob | DELETE glob only | Six-point PASS | GROUP 2A |
| `cli.py` / langchain / `--custos` | Live shims | A, B, K | Delete for “clarity” | DEPRECATE until major | SemVer / wire | GROUP 7 |
| DSL v2 | Not required | D | Rebuild profiles | DEFER | Architecture complete | after freeze |
| Tenon tokens | ATB `--state-*` already exist | I | New palette | Reuse ATB semantic colours | One family | 1.16 UI |

---

## Convergence table

| Domain | Current | Problem | Final state | Action | Release |
| --- | --- | --- | --- | --- | --- |
| Repository | `main` = `v1.15.4-dirty` | 1.16 mixed into freeze line | Clean `main` @ 1.15.4; 1.16 on `feat/v1.16.0-semantics` | MOVE | GROUP 0 |
| CLI | Help omits `intercept`; hides `serve`/`identity`; `--custos` live | Dishonest public surface | Honest help; aliases kept | REFINE / DEPRECATE | GROUP 4 |
| SDKs | Go/Py/TS independent; generated TS unused export | Confusion vs generate gate | Document generated artefact; keep bindings | DOCUMENT | GROUP 3 |
| Evidence schema | Dual families; dirty context types | Docs alias; schema.3 not cut | Dual families + overlay; catalogue new types | KEEP / DOCUMENT | 1.16 |
| Profiles | YAML vs docs split; `attestation_after_execute` unnamed | Operators “fix” HITL | YAML wins; named attestation rule | DOCUMENT / REFINE | GROUP 1 + 1.16 |
| Capture Assurance (label **Completeness**) | Completeness in repo; coverage on invalid chain | Overclaim + contract break | Completeness; schema.3; omit/untrusted coverage | REFINE | GROUP 6 |
| MCP | Dual-era code; 2026-07-28 overclaim; unwired evidence | Compatibility lie | Dual-era docs; wire or stop advertising | REFINE | GROUP 1 + NEXT |
| RAG | Two retrieval types; GC=0.3 in Overall | Scoring/UI blind to PageIndex | Overlay display; GC compatibility; no gating on GC | KEEP / REFINE | 1.16 |
| PageIndex | Adapter extra | Treated as canonical | Adapter forever | KEEP | frozen posture |
| Context lineage | `ai.context.*` only; bad heading | False model-supply / non-retrieval | Honest heading; capabilities; optional retrieval join | REFINE | **TB-UI-01/02** |
| Metadata | Enum on context units only | Retrieval/MCP unlabelled | ASSERTED/DERIVED/ATTESTED on investigation surfaces | DOCUMENT / REFINE | 1.16 |
| Incident model | Deterministic findings | Empty-state overclaim | Findings caveat on Incident | REFINE | 1.16 UI |
| Findings | Bounded conclude/cannot | Unused on default surface | Same copy everywhere | CONSOLIDATE | 1.16 UI |
| Viewer | Dirty IA + Health leftovers + graph | Observability grammar | IA kept (market-validated); Health/graph killed; three questions; findings→evidence | KEEP / REFINE | 1.16 UI |
| Design system | ATB semantic tokens exist | Health 0–100; two-status overview | Four named statuses; reuse tokens | CONSOLIDATE | 1.16 UI |
| Documentation | CHANGELOG empty; MCP/verify/events drift | Docs ≠ tree | GROUP 1 then 1.16 CHANGELOG | DOCUMENT | GROUP 1 |
| CI | Green on 1.15.4 `main` | No 1.16 branch/CI | CI on feature branch after GROUP 0 | TEST | NEXT |
| Release | 1.15.4 Docker done; 1.16 untagged | Would tag dirty tree | 1.16 only after §20 | FREEZE later | after L flip list |
| Mortise integration | Client + pin 1.14.3; receipt ID frozen | Pin lag; schema.2 reject extras | Client stays; pin/schema.3 debt owned | KEEP / DOCUMENT | **F-MORTISE-01** |
| Tenon product grammar | README hierarchy mostly right; UI overclaims | Third-product confusion | One-minute questions on every header | DOCUMENT / UI copy | 1.16 |

---

## Roadmap

### NOW

1. GROUP 0 MOVE to `feat/v1.16.0-semantics`. Restore clean `main` @ `v1.15.4`.
2. GROUP 1 docs on clean `main` (unreleased 1.16 commits): YAML `data_export`, schema.2 as current, `residual_risk` object, MCP dual-era **without** 2026-07-28-complete, OpenAPI 1.15.4, Docker Hub paragraph.
3. GROUP 2A delete `atb_cli_stub*` glob.
4. Help-text: `intercept` in `printUsage`.

### NEXT (1.16 branch)

- GROUP 6 schema.3 + **F-CAS-01**
- Wire `BuildOperationEvent` **or** stop advertising 2026-07-28 (**F-MCP-01**)
- Plaintext query opt-in (**F-MCP-02**)
- OTel `atb.event_type` allowlist (**F-OTEL-01**)
- UI: keep IA; kill Health/graph as shipped surface; Trust three questions; **TB-UI-01/02**; display-family consolidate (GROUP 5)
- Capabilities visible on Context
- Version markers + CHANGELOG Unreleased for 1.16
- Programme §9 + J/L regression tests

### AFTER FREEZE

- GROUP 7: delete `sdk/python/atb/cli.py` at a major
- Mortise re-pin to freeze ATB / schema.3 (**F-MORTISE-01** true PASS)
- Help JSON `internal` field if productised
- F-CAS-02 TC unassessable if not already done
- Lineage consumes retrieval families (true PASS)

### DEFER

DSL v2; wire-merge of families; approval-after-execute flip; GC zeroing in Overall; Mortise tenant-key isolation; Tenon cloud; CoT schema; observability expansion. Market P1/P2 after GROUP 5 P0.

### DELETE

Only `atb_cli_stub*` glob now. Health leftovers only after UI replace ships. Nothing else from the never-delete list.

---

## Blockers (J + L)

Process and freeze:

| ID | Severity | Summary |
| --- | --- | --- |
| **F-PROC-01** | BLOCKER | Dirty 1.16 unbranched on `main` |
| **F-MARKET-01** | CLOSED | Research in `docs/research/`; no architecture blocker |
| **F-CONTRACT-01** | BLOCKER | Coverage fields vs schema.2 |
| **F-CAS-01** | BLOCKER | `coverage_score` on invalid chain |
| **F-CAS-02** | HIGH (true PASS) | Missing timestamps → TC=1.0 |
| **F-MCP-01** | BLOCKER | 2026-07-28 advertised incomplete |
| **F-MCP-02** | BLOCKER | Plaintext RAG query required |
| **F-OTEL-01** | HIGH | `atb.event_type` taxonomy authority |
| **F-SEM-01** | HIGH | `data_export` YAML / docs / inference split |
| **F-LINEAGE-01** | HIGH | Lineage ignores retrieval |
| **F-DOCS-01** | HIGH | CHANGELOG empty; events/MCP/verify docs stale |
| **F-MORTISE-01** | HIGH | Pin 1.14.3; schema.2 consumers; local Custos decoy |

Trust UI:

| ID | Severity | Summary |
| --- | --- | --- |
| **TB-UI-01** | BLOCKER | “Context supplied to model” unbound |
| **TB-UI-02** | BLOCKER | Empty lineage ≠ no retrieval; Health leftovers |
| **TB-UI-03** | HIGH | Incident empty-state weaker than Findings |
| **TB-MCP-01** | HIGH | Same as F-MCP-01 |
| **TB-MCP-02** | HIGH | Same as F-MCP-02 |
| **TB-OTEL-01** | HIGH | Same as F-OTEL-01 |
| **TB-CAS-01** | HIGH | Same as F-CAS-01 |

Closing **F-PROC-01, F-CONTRACT-01, F-CAS-01, TB-UI-01, TB-UI-02, F-MCP-01 advertising, F-MCP-02** is the Lead bar to **PASS WITH DOCUMENTED DEBT**. Until then: **FAIL**.
