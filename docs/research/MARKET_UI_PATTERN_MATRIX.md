# Market UI pattern matrix

**Status:** F-MARKET-01 research. Not an architecture proposal. Not a feature request.  
**Date:** 2026-09-01  
**Method:** Primary product documentation (official docs, current help centres, current release notes). Secondary commentary used only where it describes practitioner workflow already visible in those docs.  
**Lead-locked IA (unchanged):** Incident → Findings → Timeline → Context → Relationships → Evidence → Trust.

Each row is an *interaction grammar* extracted from a reconstructed user path, not a marketing summary.

Columns: PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON ACROSS MARKET? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK

---

## How to read DECISION

- **COPY CONVENTION** — professional convention; breaking it raises learning cost.
- **ADAPT** — useful if mapped onto evidence objects, not competitor objects.
- **REJECT** — observability/eval/SIEM grammar that would blur ATB.

---

## Tier 1 — closest AI-product neighbours

### LangSmith

**Primary object:** Run (span analogue) grouped into a Trace; Traces grouped into a Thread. Trajectory is a *projection* that flattens messages.  
**User path:** production failure or eval miss → Observability project → thread/trace table → Messages / Turns / Details views → run inspector (inputs, outputs, metadata, child runs).  
**Sources:** [Observability concepts](https://docs.langchain.com/langsmith/observability-concepts) and [Configure threads](https://docs.langchain.com/langsmith/threads) (retrieved 2026-09-01).

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| LangSmith | Project → Threads tab | **Conversation/thread as second grouping over traces** | Multi-turn sessions are unreadable as a flat run list | Same metadata (`thread_id`) on every child so filters and token counts stay coherent | Yes (Langfuse sessions, Phoenix sessions, Galileo sessions) | ATB already has session_id on capture events; Incident is the investigation object, not a live thread | **ADAPT** | Keep session as a *filter and finding field*, not a nav root. `/sessions` stays an index. | Thread-first nav would demote Incident |
| LangSmith | Thread: Messages / Turns / Details (`M` `T` `D`) | **Three disclosure layers: human conversation, per-turn cards, execution detail** | Chat view hides why; tree view hides what was said | Named views with keyboard shortcuts; surrounding thread stays visible in Details | Partial (Galileo Messages vs flowchart) | Maps to ATB progressive disclosure, not to a chat product | **ADAPT** | Level 1 Incident/Findings (human). Level 2 Timeline/Context. Level 3 Evidence JSON. Do **not** ship a Messages chat surface | Chat-first would become an LLM playground |
| LangSmith | Details view | **Keep surrounding context while inspecting a run** | Lost place when drilling into a span | Side panel + thread remains | Yes (Weave 3-pane, Sentry issue+event) | Investigation context must survive surface changes | **COPY CONVENTION** | Persistent header: bundle path, integrity pill, selected seq. Surfaces swap the body only | Losing bundle identity between Timeline and Evidence |
| LangSmith | Trajectory vs Trace | **Flattened message list is a projection, not the evidence** | Nested runs overwhelm a reader who only wants the exchange | Docs say trajectory *deduplicates* messages and *removes nesting* | Product-specific naming | ATB: “never deduplicate evidence” | **REJECT** as storage/UI merge | Trajectory-like *display* of conversation units is allowed only as a view over distinct seqs | Deduplicating two equal digests into one evidence row |
| LangSmith | Evaluation / Playground | **Dataset → experiment → scores** | Improve the agent | That is the product | Category (eval platforms) | Out of ATB boundary | **REJECT** | No experiments, no LLM-as-judge, no playground | Scope expansion |
| LangSmith | 180-day SaaS retention | **Telemetry is ephemeral unless promoted to a dataset** | Cost control | Explicit TTL | Category | Opposite of ATB local-first evidence | **REJECT** | Bundles persist until the operator deletes them. Mortise is optional custody | Implying ATB traces expire |

### Langfuse

**Primary object:** Observation nested in a Trace; optional Session over traces.  
**User path:** Tracing table → click trace → **trace tree + agent graph** → observation detail (I/O, metadata). Filter bar query serializes to URL.  
**Sources:** [Core concepts](https://langfuse.com/docs/observability/data-model), [What does a good trace look like?](https://langfuse.com/docs/observability/best-practices), [Filter search bar](https://langfuse.com/docs/observability/features/filter-search-bar) (v4; retrieved 2026-09-01).

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Langfuse | Trace open | **Tree is default; graph is a sibling view of the same trace** | Nested tool/LLM steps | Parent_id nesting is real call structure, not correlation | Yes (Phoenix, Weave) | ATB relations are semantic ID joins, not parent_id trees | **ADAPT** tree as Timeline indent *only if* parent_span is present; **REJECT** graph as default | Relationships list first; graph collapsed | Graph implying causality |
| Langfuse | Root observation I/O | **Table shows human I/O from the root, raw JSON in metadata** | Reviewers scan the table without opening every row | “Not a raw JSON blob of function arguments” in the table | Yes | Human label first, type/hash second | **COPY CONVENTION** | Evidence list: human `eventDisplayLabel`; JSON in inspector | Root-I/O pattern must not hide dual event families |
| Langfuse | Filter search bar | **One-line query ↔ sidebar filters; full query in URL** | Share exact view | Same filter AST, two editors | Emerging (Phoenix expressions, Datadog facets) | ATB View is one bundle, not a multi-trace warehouse | **ADAPT** | Encode `surface` + `seq` in URL (shareable investigation). Do **not** invent a trace query language in v1.16 | Query language = SIEM |
| Langfuse | Scores on observations | **Quality scores on the same object as the span** | Eval in the trace UI | Product is observability+eval | Category | CAS is coverage under a profile, not a quality score | **REJECT** | Never put eval scores on Timeline rows | Health-score relapse |
| Langfuse | Sessions table facets | **Environment, tags, tokens, cost, bookmark** | Ops filtering | Facets for high-volume telemetry | Category | Cost/token columns are observability | **REJECT** as default columns | Optional engineer density may show token fields *if recorded*; not L1 | Live cost dashboard |

### Arize Phoenix

**Primary object:** Span; Traces tab = root spans; Sessions = traces sharing `session.id`.  
**User path:** Project → Spans/Traces table → filter expression (`status_code == 'ERROR'`) → click row → trace tree → span Events tab (exceptions). Dashboard for latency percentiles is a *separate* surface.  
**Sources:** [Signals, spans, traces, sessions](https://arize.com/docs/phoenix/tracing/concepts-tracing/otel-openinference/signals), [Filter expressions](https://arize.com/docs/phoenix/tracing/how-to-tracing/filter-expressions), [2026-08-17 release notes](https://arize.com/docs/phoenix/release-notes/08-2026/08-17-2026-trace-filters-and-analytics-sql), [High-signal traces cookbook](https://arize.com/docs/phoenix/cookbook/tracing/identify-high-signal-traces) (retrieved 2026-09-01).

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Phoenix | Traces vs Spans tabs | **Same table chrome, different grain** | Roots vs every operation | Toggle, not a different app | Yes | ATB grain is event seq inside one bundle | **ADAPT** | Evidence table is the span analogue. Do not add a second “traces” product | Dual grain in one bundle is seq vs finding |
| Phoenix | Filter bar | **Missing rollups are 0, never null** | Aggregates always defined | Explicit in 2026-08-17 notes | Product-specific | ATB must **not** score missing timestamps as 1.0 (F-CAS-02) | **REJECT** the 0-for-missing trick for trust scores | Missing order evidence → unassessable / not 1.0 | False completeness |
| Phoenix | Root I/O on list | **Blank list columns if root lacks input/output even when children are rich** | Honest empty vs hidden data | Documented pitfall | Useful warning | Empty lineage vs RAG events | **ADAPT** | Empty Context must not mean no retrieval (TB-UI-02) | Same class of bug Phoenix already warns about |
| Phoenix | Dashboard P50–P99 | **Metrics page is not the investigation page** | Tail latency | Separate surface | Category (APM) | ATB has no metrics product | **REJECT** | No percentile home | Grafana-clone |

### Braintrust

**Primary object:** Trace (root span) on Logs; Experiments share the same data shape.  
**User path:** Logs table → row → nested spans + I/O; SQL/filters/saved views; Loop agent; extract prompt to playground.  
**Sources:** [Observe your application](https://www.braintrust.dev/docs/observe), [Instrument](https://www.braintrust.dev/docs/instrument) (retrieved 2026-09-01).

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Braintrust | Logs page | **Dense searchable table of traces as the home of “observe”** | Scan production | Table-first, not graph-first | **Professional convention** | ATB Evidence/Findings should be table-first | **COPY CONVENTION** | Findings table, Evidence table, Timeline rows | Graph-first 1.15.4 layout |
| Braintrust | Logs ≡ experiments shape | **Promote a log into an eval dataset** | Improve loop | Same schema | Category | ATB bundles are not eval datasets | **REJECT** | Export remains forensic pack, not “add to dataset” | Eval platform |
| Braintrust | Loop agent on logs | **NL agent over telemetry** | Ask questions without SQL | Product feature | Emerging | Hidden reasoning / assistant-as-investigator | **REJECT** | No investigation chatbot | CoT / unsound conclusions |
| Braintrust | `bt view logs` CLI | **CLI parity for the same table** | Experts stay in terminal | Same objects | Partial | ATB CLI is source of truth | **ADAPT** | Palette + View for verify/inspect; CLI-only for intercept internals | Replay buttons |

### Weights & Biases Weave

**Primary object:** Call nested in a Trace; Threads group traces.  
**User path:** Sidebar Traces → **three panes** (list \| tree \| op details) → switch Tree / Code / Flame / Graph; scrubbers (timeline, peers, siblings, stack); `Cmd/Alt`+arrows.  
**Sources:** [Navigate the Weave Trace view](https://docs.wandb.ai/weave/guides/tracking/trace-tree), [Ops, Calls, Traces](https://docs.wandb.ai/weave/guides/tracking/tracing) (retrieved 2026-09-01).

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Weave | Traces page | **List + structure + inspector (3-pane)** | Never lose the corpus while inspecting | Classic expert layout (Sentry event, IDE) | **Professional convention** | ATB Evidence already list+inspector; Relationships should not steal the centre | **COPY CONVENTION** | Evidence: list \| inspector. Timeline: list, not 3-pane graph | 1.15.4 centre graph |
| Weave | Default = tree; Graph is an *alternative* | **Graph is opt-in structure view** | Parent/child of calls | Docs: graph “to understand structure”; flame “performance over time”; tree default | Yes (Datadog waterfall default vs map) | Validates Timeline > Graph | **COPY CONVENTION** | Graph optional on Relationships; never default truth | Graph-first |
| Weave | Scrubbers | **Same tree, multiple traversal axes (time, peers, siblings, stack)** | Nested agents | Time is one axis among several | Product-specific | ATB timeline is *recorded order*, not duration waterfall | **ADAPT** | Seq up/down on Timeline; do not add latency scrubbers unless duration is ATTESTED | Pretending ATB has span durations |
| Weave | Call vs Code vs Scores tabs | **Raw I/O separate from scores** | Don’t mix eval into the call | Tabs | Partial | Trust ≠ event inspector | **ADAPT** | Inspector: summary, fields, JSON. Trust is its own surface | Scores on every event |
| Weave | Keyboard tree walk | **Modifier+arrows to move in the tree** | Power users | Documented | Emerging | Keyboard investigation | **ADAPT** | j/k or arrows on Timeline/Evidence lists (GROUP 5 a11y) | Mouse-only lists |

### Galileo

**Primary object:** Session containing Traces containing Spans (LLM / retriever / tool).  
**User path:** Log stream → group by Session → session **flowchart** of traces → select node → **right inspector** for I/O and metrics; Messages tab for conversational list; “Condense Steps” toggle.  
**Sources:** [Using sessions](https://docs.galileo.ai/concepts/logging/sessions/using-sessions), [Custom LLM-as-a-Judge metrics](https://docs.galileo.ai/concepts/metrics/custom-metrics/custom-metrics-ui-llm) (retrieved 2026-09-01).

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Galileo | Session flowchart | **Graph as the first open of a session** | See tools as nodes | Visual agent story | Product-specific | Attractive, wrong default for ATB | **REJECT** as default | Optional Relationships graph | Galileo-clone |
| Galileo | Right-edge inspector | **Select node → facts on the right** | Detail without leaving the diagram | Split pane | **Professional convention** | Evidence inspector | **COPY CONVENTION** | EvidenceSurface list+inspector | Full-page only |
| Galileo | Retriever span type | **Retrieval is a first-class span with query in / docs out** | RAG debugging | Typed spans | Category (AI tools) | ATB retrieval.performed overlay | **ADAPT** | Context: retrieval cards from capabilities; not a quality “context adherence” metric | Galileo context-adherence score |
| Galileo | Condense Steps | **Hide spans that are “less relevant”** | Noise | Product toggle | Product-specific | Must not hide evidence | **REJECT** | Filter by family, never drop seqs from the bundle view | Deduplicating evidence |

### Scale AI / Scale GenAI Portfolio

**Primary object:** Evaluation item / annotation task; monitoring of production with scores and “full trace transparency” as a *supporting* capability.  
**User path:** Dataset → evaluation tasks (auto + HITL) → scores → optional production monitor. Marketing leads with **trust gap / safety / scores**, not incident investigation.  
**Sources:** [Enterprise evaluation](https://scale.com/evaluation/enterprise), [SGP platform](https://scale.com/genai-platform), [Next-gen evals overview](https://docs.gp.scale.com/docs/v5/next-gen-evaluation/overview) (retrieved 2026-09-01).

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Scale | Trust Feedback Loop | **Evaluate → improve → monitor as the product loop** | Deploy with “confidence” | Enterprise eval buyer | Category | Opposite loop to ATB forensics | **REJECT** | ATB does not close a quality loop | Compliance theatre |
| Scale | HITL on hard cases | **Human review as a labelled evaluation task** | Auto-eval is insufficient | Workforce + rubrics | Category | ATB human.approval is evidence, not a scoring workforce | **ADAPT** copy only | Show approval events as evidence; no review-queue product | Mortise review queue (Ring 4) |
| Scale | “Full audit trail” marketing | **Audit trail as a claim on the eval platform** | Buyer language | Vague | Anti-pattern if unearned | ATB must not copy the slogan | **REJECT** | Keep “integrity of records presented” | Overclaim |

---

## Tier 2 — investigation / incident / security grammar

### Sentry

**Primary object:** **Issue** (clustered errors). Trace is supporting context.  
**User path:** Issues list (`is:unresolved`) → Issue Details (message, users, stack, breadcrumbs) → Trace Preview → full Trace waterfall → span inspector. Missing spans shown as **dashed** “broken subtrace” / “missing instrumentation”. JSON package download in the event sidebar.  
**Sources:** [Issue details](https://docs.sentry.io/product/issues/issue-details/), [What to prioritize](https://docs.sentry.io/guides/issues-errors/), [Trace view](https://docs.sentry.io/concepts/key-terms/tracing/trace-view/) (retrieved 2026-09-01).

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Sentry | Issues home | **Issue-first, not trace-first** | “What is wrong?” before “what ran?” | Clusters noise; trace is one click away | **Professional convention** for incident tools | ATB Finding-first under Incident | **COPY CONVENTION** | Incident → Findings → seq drill to Evidence | Trace-first 1.15.4 |
| Sentry | Issue Details header | **Human error message first; counts second; actions in header** | Assign/resolve without losing the title | Header anatomy | Yes (GitHub alerts) | Finding title + bounded can/cannot | **ADAPT** | FindingCard header; no assign/resolve workflow in v1.16 | Ticket system |
| Sentry | Trace Preview on the issue | **Abbreviated structure, then “open full trace”** | Context without drowning | Progressive | Yes | Relationships preview optional | **ADAPT** | Finding lists event_seqs; click opens Evidence | Mini-graph as causality |
| Sentry | Waterfall | **Chronology + ancestry (child under parent)** | Duration and order of instrumented work | Left ops, right duration | Category (APM) | ATB often lacks duration; parent_span ≠ cause | **ADAPT** chronology; **REJECT** duration-as-truth | Timeline = recorded order. Indent only with attested parent | Causal edges |
| Sentry | Dashed missing instrumentation | **Unknown/incomplete shown as a distinct visual, not as zero** | Honesty about capture holes | Explicit gap | **Professional convention** for trust | Empty lineage, unassessable CAS | **COPY CONVENTION** | Untrusted / not assessed / “lineage not recorded” | Empty = all-clear |
| Sentry | Event JSON download | **Raw package on demand from the inspector chrome** | Forensics / support | Sidebar, not centre | Yes | L3 raw JSON | **COPY CONVENTION** | Evidence inspector + export pack | JSON as L1 |

### Splunk Enterprise Security (Mission Control)

**Primary object:** **Finding** / investigation in an analyst queue.  
**User path:** Mission Control queue → finding → investigation (evidence gathering) → optional timeline of when findings were *generated* → contributing events. Detections create findings from raw events.  
**Sources:** [Mission Control overview 8.6](https://help.splunk.com/en/splunk-enterprise-security-8/user-guide/8.6/mission-control/overview-of-mission-control-in-splunk-enterprise-security) (updated 2026-07-21), [Analyst queue 8.4](https://help.splunk.com/en/splunk-enterprise-security-8/administer/8.4/mission-control/manage-analyst-workflows-using-the-analyst-queue-in-splunk-enterprise-security) (retrieved 2026-09-01).

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Splunk ES | Analyst queue | **Findings and investigations share one dense queue** | SOC triage | Urgency/status/owner facets | Category (SIEM) | ATB has one bundle, not a queue of tickets | **ADAPT** | Findings surface is the queue analogue *inside* an incident | Mission Control product |
| Splunk ES | Investigation := structured evidence gathering | **Investigation is not the raw event list** | Process vs data | Named object | Yes (Elastic cases) | ATB Incident is that object | **COPY CONVENTION** | Incident overview = investigation header | Alerting product |
| Splunk ES | Timeline of finding *generation* | **Time of detection ≠ time of activity** | When the SOC saw it | Separate from event time | Category | ATB timeline is event seq in the bundle | **ADAPT** | Do not mix CAS computation time into the event timeline | Confusing clocks |
| Splunk ES | Pie charts of urgency | **Dashboard chrome over the queue** | Fleet metrics | SOC manager | Category | Health 0–100 | **REJECT** | No urgency pies on Incident | Observability home |
| Splunk ES | Risk score aggregation | **Numeric risk as a ranking device** | Prioritise entities | SIEM | Category | Single score | **REJECT** | Three Trust questions | CAS-as-risk |

### Elastic Security

**Primary object:** Alert → **Timeline workspace** (query + events) → Case (documentation). Visual event analyzer is a *process tree*, not a generic blob graph.  
**User path:** Alerts table → Investigate in Timeline → KQL/EQL/ES\|QL → attach Timeline to a case.  
**Sources:** [Investigate](https://www.elastic.co/docs/solutions/security/investigate), [Timeline](https://www.elastic.co/docs/solutions/security/investigate/timeline), [Cases](https://www.elastic.co/docs/solutions/security/investigate/security-cases) (retrieved 2026-09-01).

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Elastic | Timeline as workspace | **Timeline is the investigation workbench, not a decoration** | Correlate alerts and raw events | Central, queryable | **Professional convention** in IR | Validates Timeline outranking graph | **COPY CONVENTION** | Timeline surface is L2 primary temporal tool | Graph as workbench |
| Elastic | EQL sequences | **Ordered patterns across categories, explicitly queried** | “Then this then that” as a *question*, not a drawing | User states the sequence hypothesis | Category | ATB must not draw causal edges from seq | **ADAPT** | Relations = ID joins; sequence is Timeline order only | EQL product |
| Elastic | Process tree analyzer | **Tree when the relationship is parent process** | Execution chains | Typed relation | Category | Only if parent_span/process is in evidence | **ADAPT** | Optional indent; no invented trees | Fake process tree |
| Elastic | Attach timeline to case | **Preserve queries/filters as evidence of the investigation itself** | Handoff | Investigation meta-evidence | Category | ATB investigation is over a frozen bundle | **REJECT** for v1.16 | Bundle *is* the artefact; no case database | Workflow orchestrator |
| Elastic | AI chat to generate queries | **Assistant writes KQL** | Speed | Optional | Emerging | Unsound ATB conclusions | **REJECT** | No query-writing agent | Hidden reasoning |

### Datadog APM

**Primary object:** Trace visualized as **Waterfall (default)**, Flame, Span list, or Map.  
**User path:** Trace Explorer → sample → Waterfall → span metadata. Map is service-level overview; inferred services get **dashed** outlines.  
**Sources:** [Trace view](https://docs.datadoghq.com/tracing/trace_explorer/trace_view/) (retrieved 2026-09-01).

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Datadog | Waterfall default, Map optional | **Time/ancestry first; topology second** | Find slow/error span | Duration encoded in the bar | Category | No duration-centric ATB home | **ADAPT** | Timeline list = waterfall-without-bars. Map ≡ optional Relationships graph | Service map as Incident |
| Datadog | Inferred service dashed | **Uncertainty has a visual grammar** | Incomplete telemetry | Not drawn as fact | Yes (Sentry dashed) | Untrusted / asserted | **COPY CONVENTION** | Provenance badges; dashed optional graph nodes if type is inferred | Solid edges for semantic joins |
| Datadog | Flame graph | **Performance over time** | CPU/latency | Wrong tool for forensics of *what was recorded* | Category | **REJECT** | No flame charts in ATB View | Observability |

### Grafana Tempo

**Primary object:** Trace in **Explore** (waterfall/flame). Dashboards are a *secondary* embedding.  
**User path:** TraceQL or trace ID in Explore → waterfall. Docs note that putting traces *on a metrics dashboard* is awkward (manual traceId variable).  
**Sources:** [Grafana traces visualization](https://grafana.com/docs/grafana/latest/visualizations/panels-visualizations/visualizations/traces/) (retrieved 2026-09-01).

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Grafana | Explore vs dashboard | **Investigation ≠ dashboard** | Dashboards are metrics-first | Trace panels are bolted on | **Anti-pattern for ATB home** | 1.15.4 Trust Dashboard | **REJECT** dashboard-home | Kill Trust Dashboard / Health | Grafana-clone |

### GitHub Security (code scanning)

**Primary object:** **Alert** (finding).  
**User path:** Security tab → Code scanning list (filters) → alert page (why, where, help, affected branches) → code. Status on non-default branches is **grey** (“in pull request”), not green/red.  
**Sources:** [Assessing code scanning alerts](https://docs.github.com/en/code-security/how-tos/manage-security-alerts/manage-code-scanning-alerts/assess-alerts) (retrieved 2026-09-01).

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| GitHub | Alert list → detail | **Finding-first with evidence location** | Developer remediation | List+detail, filters | **Professional convention** | Findings → Evidence seq | **COPY CONVENTION** | FindingCard seq controls | Alert-as-vulnerability scoring |
| GitHub | Grey branch status | **Not-on-default ≠ failed** | Avoid false red | Colour+text | Yes | Not assessed / inconclusive | **COPY CONVENTION** | Unknown token for not assessed | Red for missing optional |
| GitHub | Command palette ⌘K | **Navigate + command modes** | Power users | `>` command mode; scoped | **Professional convention** | Dirty CommandPalette | **COPY CONVENTION** | ⌘K; groups Navigate / Operate / Copy; arrow keys (G freeze) | Palette as chat |

---

## Tier 3 — assurance / governance / custody-adjacent (retained only)

### Vanta

**Primary object:** Control; evidence and tests *support* the control. Control status **Ok / Needs evidence** is *not* the auditor’s opinion.  
**User path:** Controls table → control → mapped tests/documents → Evidence tab (what was checked, resource table, pass/fail per resource). Auditor sees a restricted evidence view (approve / flag / N/A).  
**Sources:** [Controls page](https://help.vanta.com/en/articles/11345373-controls-page) (updated 2026-07-23), [Automated test evidence](https://help.vanta.com/en/articles/11345529-automated-test-evidence), [Audit evidence statuses](https://help.vanta.com/en/articles/11345427-audit-evidence) (retrieved 2026-09-01).

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Vanta | Control status vs auditor assessment | **Operational coverage ≠ audit opinion** | Stop fake certification | Explicit disclaimer | **Professional convention** for assurance | CAS / profile pass vs “certified” | **COPY CONVENTION** | Trust limitations; coverage is profile-scoped | CAS as SOC2 |
| Vanta | Needs evidence | **Missing evidence is a first-class status, not a zero score** | Honest gap | Named state | Yes | Unassessable dimensions; empty Context | **COPY CONVENTION** | Not present / not assessed | 0% coverage as “nothing happened” |
| Vanta | Evidence tab itemizes resources | **Show the rows the test actually evaluated** | Why did it fail? | Table of subjects | Yes | Obligation list + missing events | **ADAPT** | Trust L3: critical_failures list | GRC product |
| Vanta | Auditor flag / N/A | **Workflow on evidence** | Audit process | Out of ATB runtime | Mortise/Tenon defer | **REJECT** for ATB; **DEFER** Mortise | No legal-hold/review queue | Ring 4 |

Drata is the same control→test→evidence grammar; no additional ATB pattern beyond Vanta. Not expanded.

### Sigstore / Rekor (custody-adjacent, not a SaaS console)

**Primary object:** Transparency-log entry + inclusion proof.  
**User path:** Sign artefact → log leaf → **verify inclusion against checkpoint**. UI is CLI/keytool, not a dashboard.  
**Source:** pattern already frozen for Mortise (`mortise-keytool verify-inclusion`); official model at [Sigstore docs](https://docs.sigstore.dev/) (retrieved 2026-09-01).

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Rekor | Inclusion proof | **Proof object + public checkpoint, not a green shield** | Independent verify | Cryptographic | Category (tlogs) | Mortise inspector | **ADAPT** Mortise-only | ATB Trust: “External custody present” + link; no fake tlog UI in ATB | ATB pretending WORM |

**Not retained:** Lakera / runtime prompt-injection consoles (guardrail product, not investigation). Protect AI model-security catalogues (inventory, not forensics). They do not teach ATB View grammar.

---

## Command / expert surfaces (cross-cutting)

| PRODUCT | SCREEN / WORKFLOW | PATTERN | USER PROBLEM SOLVED | WHY IT WORKS | COMMON? | ATB RELEVANCE | DECISION | ATB APPLICATION | RISK |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| GitHub | ⌘K palette | Search vs `>` commands | Find or act without hunting menus | Scoped to location | **Professional convention** | CommandPalette | **COPY CONVENTION** | Keep; add a11y | Chat |
| VS Code | ⇧⌘P commands | Every action reachable by name | Expert density | Discoverability | Convention for tools | CLI→View mapping | **ADAPT** | Palette lists View actions; CLI-only stays CLI-only | Exposing intercept as Replay |

---

## Workflow reconstructions (summary)

| Product | Sees first | Primary object | Second level | Raw detail | Disclosure depth | Back | Filters | Uncertainty |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| LangSmith | Project traces/threads | Trace / thread | Run | Metadata JSON | 3 views | Thread remains | Tags/metadata | Retention TTL |
| Langfuse | Trace table | Trace | Observation tree | Metadata | Tree then graph | Table | URL-serialized query | — |
| Phoenix | Spans/Traces table | Span / root | Tree | Events tab | Table→tree | Table | Expressions (missing→0) | Blank root I/O |
| Braintrust | Logs table | Trace | Nested spans | I/O | Table→trace | Table | SQL/saved views | — |
| Weave | Trace list | Trace | Tree (graph optional) | Call tabs | 3 panes | List pane | Name regex | — |
| Galileo | Session table | Session | Flowchart | Right inspector | Graph+messages | Session | Grouping toggle | Condense hides |
| Sentry | Issues | Issue | Event + trace preview | JSON package | Issue→event→trace | Issues list | Search queries | Dashed gaps |
| Splunk ES | Analyst queue | Finding | Investigation | Search | Queue→finding→events | Queue | Time/urgency | Unknown urgency |
| Elastic | Alerts | Alert | Timeline workspace | Event fields | Alert→timeline→case | Alerts | KQL | — |
| Datadog | Trace explorer | Trace | Waterfall | Span attrs | Waterfall/map | Explorer | Facets | Dashed inferred |
| GitHub | Security alerts | Alert | Code + help | SARIF/code | List→alert | List | Tool/branch | Grey non-default |
| Vanta | Controls | Control | Tests/evidence | API/resource table | Control→evidence | Controls | Search | Needs evidence ≠ auditor fail |

---

## Products that teach Tenon family grammar

Successful families (GitHub: Issues vs Security vs Actions; Grafana: dashboards vs Explore; Datadog: APM vs Logs vs Security) share **tokens, tables, inspectors, and status colour** but **change the header question and the primary object**. That validates Tenon: ATB investigation rail vs Mortise custody column vs no third Tenon app.
