# Professional UI grammar

**Status:** F-MARKET-01. Conventions extracted from `MARKET_UI_PATTERN_MATRIX.md`.  
**Date:** 2026-09-01  
**Lead-locked IA:** Incident → Findings → Timeline → Context → Relationships → Evidence → Trust. Not replaced.

Classification used in this file:

- **A. PROFESSIONAL CONVENTION** — breaking it unnecessarily increases learning cost.
- **B. CATEGORY CONVENTION** — native to observability, SIEM, or eval; copy only if it maps to evidence.
- **C. PRODUCT-SPECIFIC** — interesting, not a default.
- **D. ANTI-PATTERN** — actively unsuitable for ATB.

---

## 1. Market consensus

Across Sentry, GitHub Security, Splunk ES, Elastic, Vanta, Weave, Braintrust, and Langfuse’s table chrome, professionals already know:

1. A **persistent left or top nav** of named jobs, not a single dashboard canvas.
2. A **dense table** as the corpus (issues, alerts, logs, traces, controls).
3. A **split inspector** (list | detail) rather than navigating away for every row.
4. **Human summary first**, machine identifiers second, **JSON on demand**.
5. **Status is a word plus colour**, never colour alone (GitHub grey “in PR”, Vanta “Needs evidence”, Sentry dashed gaps).
6. **⌘K / Ctrl+K** (web) or ⇧⌘P (editors) for command+search.
7. **Investigation context stays on screen** (header, breadcrumbs, remaining thread/list pane).
8. **Shareable URLs** for the current object (and, in telemetry tools, for filters).

ATB should feel like Sentry Issue Details + Elastic Timeline + Vanta evidence honesty, **not** like Langfuse-on-a-graph or Grafana-on-a-score.

---

## 2. Investigation conventions

**A — Issue/finding/alert first when the user’s question is “what is wrong?”**  
Sentry Issues, GitHub code scanning, Splunk findings, Elastic alerts all land on a clustered *problem object* before a trace waterfall. Why: a trace is how you *explain* a finding, not how you *triage*. ATB’s Incident → Findings matches this. Trace-first (LangSmith/Langfuse home) is **B** for live debugging.

**A — Bounded empty states.**  
Sentry missing instrumentation and Vanta “Needs evidence” refuse to treat absence as success. Why: investigators over-trust empty queues. ATB Findings copy already states non-proof of complete capture; Incident must use the same sentence.

**B — Ticket workflow (assign, resolve, SLA).**  
Splunk owner/status, Elastic cases. Why: SOC process. ATB is a local evidence viewer, not a case manager. Reject as product; do not block freeze.

**D — Chat-first investigation.**  
LangSmith Chat, Elastic AI chat, Braintrust Loop. Why they exist: query generation over huge corpora. Why ATB must not: unsound conclusions, hidden reasoning, playground feel.

---

## 3. Security conventions

**A — Analyst queue is a table of findings.**  
Splunk Mission Control, Elastic Alerts. Why: scan, filter, open. ATB Findings surface is the in-bundle analogue.

**A — Timeline is the workbench for “what happened in what order?”**  
Elastic Timeline is *the* investigation workspace; Splunk shows a timeline of findings as an *optional* overlay. Why: time is the natural axis for incidents. Validates ATB Timeline > graph.

**B — Query languages (KQL, SPL, EQL, TraceQL).**  
Why: multi-index hunt. ATB has one NDJSON bundle. A query language in v1.16 would be SIEM-shaped. **REJECT** for freeze. Filter-by-type and jump-to-seq are enough.

**B — MITRE / risk scores.**  
Splunk risk timeline. Why: ATT&CK mapping. ATB must not grow a risk score (Health relapse).

**D — Dashboards of pie-chart urgency as the home.**  
Splunk charts icon. Why they exist: SOC managers. ATB viewers are investigators of one artefact.

---

## 4. AI-tool conventions

**B — Trace tree of LLM/tool/retriever spans.**  
Langfuse, Phoenix, Weave, Galileo. Why: nested calls have parent_id. ATB events often lack a true call tree; MCP/PageIndex are adapters. Use **Timeline of recorded events**; indent only when `parent_span_id` is present.

**B — Session/thread grouping.**  
LangSmith threads, Langfuse sessions. Why: multi-turn products. ATB may filter by `session_id` but must not make Sessions the IA root (`/sessions` remains an index).

**C — Messages vs Details dual view.**  
LangSmith `M`/`D`, Galileo Messages vs flowchart. Why: conversation vs execution. ATB analogue is Context (units) vs Timeline (operations) — two surfaces, not a chat widget.

**D — Eval scores, playgrounds, “add to dataset”, LLM-as-judge.**  
LangSmith Evaluation, Braintrust experiments, Galileo metrics, Scale HITL rubrics. Why: improve the model. ATB boundary: no eval platform.

**D — Trajectory that deduplicates messages.**  
LangSmith trajectory. Why: readability. ATB rule: never deduplicate evidence.

**D — Context-adherence / quality metrics on retrieval.**  
Galileo. Why: RAG eval. ATB records retrieval; it does not score relevance.

---

## 5. Audit / assurance conventions

**A — Control/profile coverage is not an auditor’s opinion.**  
Vanta: control Ok/Needs evidence “does not represent an auditor’s assessment.” Why: legal and epistemic hygiene. ATB CAS and profile pass must keep the same disclaimer.

**A — Itemized evidence of what was actually tested.**  
Vanta Evidence table of resources. Why: “show your working.” ATB Trust L3 = critical_failures, dimension_assessments, hashes.

**A — Named gap states (Needs evidence, N/A, Flagged) instead of a single 0–100.**  
Why: 92/100 collapses missing, failed, and untested. Validates Lead three-question Trust.

**B — Auditor workflow (approve/flag).**  
Vanta auditor instance. Out of ATB. Mortise may later show receipt verification, not an audit opinion UI.

**D — “Certified compliant” / Trust Center as the investigation UI.**  
Marketing of Scale and GRC trust centres. ATB README already denies this.

---

## 6. Navigation conventions

**A — Persistent nav of named jobs.**  
GitHub tabs, Sentry sidebar, Vanta left nav, Weave project sidebar. Why: orientation. ATB’s seven-item investigation rail is this pattern applied to forensics.

**A — Breadcrumb: product › object › surface.**  
Why: back-stack without a browser fight. ATB: ATB View › bundle › Incident.

**A — Header holds identity + primary status + command.**  
Sentry issue header; Weave three panes keep the list. Why: don’t lose the artefact.

**B — Workspace/project switcher.**  
LangSmith projects, Phoenix projects. ATB is local-first one-bundle View; `/sessions` is enough. No org switcher in ATB.

**D — Dashboard as the default route for an investigation tool.**  
Grafana; ATB 1.15.4 Trust Dashboard. Why it fails: metrics are not evidence.

---

## 7. Search / filter conventions

**A — Filterable dense tables with sort.**  
Universal. ATB Findings/Evidence/Timeline.

**A — Command palette for actions and jumps.**  
GitHub ⌘K, VS Code ⇧⌘P, Linear ⌘K. Why: experts. ATB already chose ⌘K (web convention, not editor ⇧⌘P). Keep ⌘K; document it.

**B — Query language in the filter bar.**  
Langfuse v4, Phoenix expressions, Datadog facets. Why: high-volume telemetry. **P2** at most for ATB (filter type/family). No SPL.

**B — Saved views / bookmarks.**  
Langfuse saved views, Braintrust saved views. Why: repeated hunts. **P2** — URL of surface+seq is enough for freeze.

**A — Filters/object in the URL.**  
Langfuse serializes the query; Datadog shareable service URLs. Why: handoff. ATB should encode `surface` and `selectedSeq` (**P1**).

---

## 8. Detail / inspector conventions

**A — Split pane: corpus stays, detail updates.**  
Weave, Galileo right inspector, Sentry event sidebar, ATB dirty Evidence. Why: orientation.

**A — Tabs inside the inspector for I/O vs raw vs metadata.**  
Weave Call/Code/Summary. Why: progressive disclosure. ATB: summary → fields → JSON.

**C — Full-page detail.**  
GitHub alert page. Why: long prose (CWE help). ATB findings are short; split pane is better. Use full-page only if needed for Trust L3 print.

**D — Graph occupying the centre pane as the inspector.**  
1.15.4 ATB; Galileo session flowchart as first paint. Why it fails for ATB: structure ≠ recorded order; edges look causal.

---

## 9. Timeline / tree / graph conventions

What each is *for*:

| Representation | Good at | Bad at | ATB |
| --- | --- | --- | --- |
| **Event timeline (list)** | Order of records, sparse events, missing times as gaps | Nested call performance | **Primary** |
| **Waterfall** | Duration + ancestry of spans | Events without duration; implying cause | Only if duration ATTESTED (rare) |
| **Trace tree** | Parent/child calls | Semantic “related via id” | Indent if parent_span present |
| **Flame graph** | CPU/latency | Forensics of meaning | **REJECT** |
| **Dependency / service map** | Service topology, error rate on edges | Evidence identity | **REJECT** as home |
| **Relationship table** | Typed joins with evidence_value | Pretty pictures | **Primary for Relationships** |
| **Relationship graph** | Optional overview of the same joins | Default truth, causality | **Optional, collapsed, captioned** |

**A — Timeline (or waterfall) outranks topology maps** in every serious debugger (Datadog waterfall default, Elastic Timeline workspace, Weave tree default, Sentry waterfall after the issue).  
**A — Graphs that mean “parent call” or “service call” must be labelled as such.**  
**D — Graph as the default truth surface.**  
**D — Solid edges from “A happened then B.”** ATB `causal_edge` remains false.

Evidence required before ATB draws a **semantic** edge: shared identifier already in `supportedRelationships` (`action_id`, `request_id`, `approval_id`, `policy_id`, `retrieval_id`, `invocation_id`, envelope `trace_id`). Strength `semantic`. Caption: not causation.

---

## 10. Trust conventions

**A — Separate “the check failed” from “the check was not run.”**  
Vanta Needs evidence vs failing test; GitHub grey vs red; Sentry dashed vs error span. Why: 0% and “failed” are different legal/operational statements. Maps to Integrity INVALID vs Coverage not assessed vs Corroboration not present.

**A — No single health number as the answer.**  
Eval platforms and APM use scores because they *are* scoring quality or latency. Assurance products that stay honest (Vanta’s disclaimer) refuse to equate coverage with opinion. Validates rejection of Viewer Health 92/100.

**A — Named checks in a header strip.**  
Sentry: issue counts + environments. Vanta: Ok / Needs evidence. ATB Lead strip: Integrity, Profile, Coverage, Obligations, Corroboration, Anchor, Custody.

**Vocabulary (Lead-locked + this research):**

| Term | Use | Distinct from |
| --- | --- | --- |
| **VERIFIED** | Named cryptographic or profile check passed | Coverage % |
| **FAILED** | Named check ran and did not pass | Missing |
| **INCONCLUSIVE** | Check could not be completed (mixed signatures, partial XC) | Failed |
| **UNTRUSTED** | Result exists but the chain is invalid; do not use coverage % | Failed |
| **NOT ASSESSED** | No profile / dimension unassessable | Failed |
| **NOT PRESENT** | Optional evidence absent | Failed |
| **ASSERTED** | Producer claim | Verified identity |
| **PARTIAL** | Some independent support, not fully verified | Verified |

These are distinct enough if the UI always pairs the word with the *named check* (“Hash chain verified”, not a lone green). **INCONCLUSIVE** absorbs “mixed signatures.” Do not add a fifth colour.

**D — Green banner meaning “the whole investigation is healthy.”** Integrity banner must stay integrity-only.

---

## 11. Technical-data conventions

**A — Mono for IDs, hashes, types; proportional for prose.**  
Universal in Weave, Sentry JSON, GitHub SHAs. ATB Inter + JetBrains Mono.

**A — Truncate + copy hash.**  
Sentry event id, GitHub SHAs, Weave call ids. ATB `HashValue`.

**A — JSON package / download in the inspector chrome.**  
Sentry event JSON. ATB L3 + `atb incident export`.

**A — Timestamps in ISO, local display optional.**  
Do not hide missing timestamps by scoring 1.0.

---

## 12. Accessibility conventions

**A — Keyboard path that does not depend on canvas.**  
GitHub palette, VS Code palette, Weave tree arrows; Elastic/Sentry lists. Graph is enhancement (G freeze).

**A — Colour not sole status.**  
WCAG; GitHub dual-encoding. ATB semantic tokens.

**A — Focus visible; dialog semantics for palettes.**  
GitHub command palette docs. Dirty CommandPalette still lacks `aria-activedescendant` — **P0 a11y**.

**A — `prefers-reduced-motion`.** Already in ATB `globals.css`. Keep; tamper pulse must honour it.

---

## 13. Patterns ATB should deliberately reject

| Pattern | Why neighbours have it | Why ATB must not |
| --- | --- | --- |
| Trace-first home | Live debug | Principal object is evidence/incident |
| Graph-first session | Agent storytelling | Edges look causal; not ATB’s relation model |
| Flame graphs / service maps | Latency and topology | Not recorded-evidence questions |
| Health 0–100 / eval scores | Quality loops | False certainty; CAS is coverage |
| Playground / prompt iteration | Improve the model | Out of boundary |
| NL investigation agent | Query generation | Unsound + CoT risk |
| Deduplicated trajectory | Readability | Never deduplicate evidence |
| SIEM query language | Multi-source hunt | One bundle |
| Urgency pies / live polling | Fleet ops | Not a console |
| “Certified / audit trail” slogans | GRC marketing | Epistemic non-claims |
| Chat as nav | Consumer AI | Playground |
| Mortise investigation clone | Convenience | Custody vs what-happened |
| Context “supplied to model” unbound | AI-tool sloppiness | Lead + Trust FAIL |

---

## Why the frozen IA survives

Neighbours that **investigate incidents** (Sentry, Splunk, Elastic, GitHub Security) are finding-first then temporal then raw. Neighbours that **debug agents** (LangSmith, Langfuse, Weave, Galileo) are trace-first then tree/graph then I/O. ATB is the first category with cryptographic evidence. Using the second category’s home screen is the 1.15.4 mistake. The dirty 1.16 rail is the first category. Market research **validates** it; it does not replace it.
