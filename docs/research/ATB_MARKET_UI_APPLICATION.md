# ATB market UI application

**Status:** F-MARKET-01 applied to the frozen ATB model. No new architecture. No GROUP 0.  
**Date:** 2026-09-01  
**Companion:** `MARKET_UI_PATTERN_MATRIX.md`, `PROFESSIONAL_UI_GRAMMAR.md`.

Priorities:

- **P0** — trust correctness, serious usability, accessibility, or a professional convention whose absence makes a core workflow confusing, or inconsistency with the frozen grammar.
- **P1** — desirable in v1.16 if already in scope.
- **P2** — later polish.
- **REJECT** — not ATB.

P0 is intentionally short. Most “market” items already exist as Trust/UI freeze must-fixes (TB-UI-01/02, Health deletion). This file does not inflate that list.

---

## Pattern → ATB

| PATTERN | MARKET EVIDENCE | CURRENT ATB | DECISION | EXACT APPLICATION | COMPONENT(S) | PRIORITY |
| --- | --- | --- | --- | --- | --- | --- |
| Finding-first under an investigation | Sentry Issues; GitHub alerts; Splunk findings | Dirty IA Incident→Findings; 1.15.4 was graph-first | KEEP IA | Default surface Incident; Findings next; seq drill to Evidence | IncidentSurface, FindingTable, FindingDetail | **P0** (already Lead; implement in GROUP 5) |
| Timeline as investigation workbench | Elastic Timeline; Datadog waterfall default | Dirty Timeline exists; Relationships still mounts 420px graph first | KEEP Timeline > graph | List-first Timeline; graph collapsed on Relationships with anti-causality caption | EvidenceTimeline, RelationshipTable, RelationshipGraph | **P0** (usability + frozen grammar) |
| Split list + inspector | Weave 3-pane; Galileo inspector | Evidence already; Timeline rows not buttons | COPY | Timeline/Evidence rows activate inspector; Findings seq → Evidence | EvidenceTable, EvidenceInspector | **P0** (core workflow confusing if seqs are dead text) |
| Human label first, IDs mono | Langfuse root I/O; GitHub alert titles | event-labels exist; engineer density | COPY | Security/Auditor: human primary; Engineer: type primary | TimelineEvent, TechnicalIdentifier | P1 |
| JSON on demand | Sentry event JSON | Inspector JSON; Features.tsx wrong on reveal | COPY | L3 only; sidecar copy for reveal | RawEvidenceViewer | **P0** if landing still claims bundle mutation; else P1 for View |
| Named gap ≠ zero | Vanta Needs evidence; Sentry dashed; GitHub grey | Context empty = “no lineage”; CAS TC=1.0; Health 0 | COPY | Empty Context: capabilities vs lineage; Untrusted when integrity invalid; no Health | EmptyState, TrustSummary, ContextTree | **P0** (TB-UI-01/02, F-CAS-01) |
| Three named trust questions | Vanta control≠opinion; Lead lock | Health 92/100; ProfileCAS blob | KEEP Lead | Integrity / Coverage / Corroboration strip; no 0–100 | TrustSummary, IntegrityCheck, CoverageBreakdown, CorroborationStatus, CustodyStatus | **P0** |
| Integrity banner ≠ coverage | Sentry error vs missing instrumentation | Green banner reads as “all good” | ADAPT | Banner copy “Hash chain verified”; coverage on Trust/Incident rows | InvestigationHeader, VerificationBanner | **P0** (trust copy) |
| Command palette ⌘K | GitHub, Linear, dirty CommandPalette | Exists; weak a11y | COPY | Focus trap, arrows, aria-activedescendant; Navigate/Operate/Copy | CommandPalette | **P0** (a11y) |
| Persistent investigation context | LangSmith Details keeps thread; Weave keeps list | Surfaces swap; bundle in header | COPY | Header always: lockup, bundle path, integrity, ⌘K, presentation | InvestigationHeader, AppShell, PrimaryNav | P1 |
| URL encodes object | Langfuse filters in URL | Surface state in React | ADAPT | `?surface=&seq=` | AppShell | P1 |
| Optional structure graph | Weave Graph alt view; Datadog Map | Graph first on Relationships | ADAPT | Collapsed, captioned, keyboard users use table | RelationshipGraph | P1 |
| Provenance badges | Datadog dashed inferred; identity asserted | Missing on inspector | ADAPT | ASSERTED/DERIVED/ATTESTED | ProvenanceBadge | P1 |
| PageIndex as adapter card | Galileo retriever span *type*; not canonical | Capabilities unused | ADAPT | Retrieval cards from capabilities; badge PageIndex | ContextUnit | **P0** (TB-UI-02) |
| Trace-first home | LangSmith/Langfuse | 1.15.4 | REJECT | Do not restore | — | REJECT |
| Flame / service map / eval scores / playground / Loop | APM + eval suite | Health, StatsOverview | REJECT | Delete from shipped View | TrustScore* | REJECT |
| SIEM query language | Elastic/Splunk | — | REJECT | Type/family filter only | FilterBar | REJECT |
| Deduped trajectory | LangSmith | — | REJECT | Unique seqs | — | REJECT |
| Mortise investigation screens | — | ui.html custody-only | REJECT | Deep-link ATB | — | REJECT |
| Shareable filter language | Langfuse v4 | — | REJECT v1.16 | P2 maybe | — | P2 |
| Saved views | Braintrust | — | REJECT v1.16 | URL is enough | — | P2 |
| Waterfall duration bars | Datadog/Sentry | No span duration contract | REJECT default | Only if duration ATTESTED | — | REJECT |

---

## Screen-by-screen freeze validation

Lead-locked order. Architecture wins over nicer alternatives.

### INCIDENT

| Field | Specification |
| --- | --- |
| PRIMARY QUESTION | What happened? What matters? Is the evidence valid? How strong is coverage? |
| PRIMARY OBJECT | The bundle as an investigation (not a trace, not a dashboard) |
| SECONDARY OBJECTS | Top findings; integrity; profile-scoped coverage; custody row |
| DEFAULT DENSITY | Comfortable; human prose |
| PRIMARY ACTION | Open top finding or Trust if integrity failed |
| FILTERS | None (single artefact) |
| DETAIL BEHAVIOUR | Inline summary; no graph |
| EMPTY STATE | Use Findings caveat: does not prove complete capture or universal absence |
| ERROR STATE | Investigation could not load; retry |
| UNTRUSTED STATE | Integrity failed: pill + banner; **do not show coverage %** (F-CAS-01); Incident+Trust still reachable |
| KEYBOARD | Land on nav; Tab to findings; ⌘K |
| CLI PARITY | `atb verify --format json` → integrity/profile; `atb incident list` → findings preview. **DIRECT UI** for verify; **VISUAL** for summary |
| MARKET USED | Sentry issue header; Vanta “not auditor opinion”; Splunk investigation object |
| REJECTED | Health 92/100; urgency pies; live polling; LangSmith project dashboard |

### FINDINGS

| Field | Specification |
| --- | --- |
| PRIMARY QUESTION | What does ATB conclude from recorded evidence, and what can it not conclude? |
| PRIMARY OBJECT | Finding (bounded) |
| SECONDARY OBJECTS | event_seqs, session_id, severity text |
| DEFAULT DENSITY | Table/cards; engineer slightly denser |
| PRIMARY ACTION | Open supporting event in Evidence |
| FILTERS | Optional: flag name (P1) |
| DETAIL BEHAVIOUR | Expand can/cannot; seq is a control |
| EMPTY STATE | “No findings. This does not prove complete capture or universal absence.” |
| ERROR STATE | Integrity gate: cannot compute findings |
| UNTRUSTED STATE | Same as integrity fail — do not invent findings |
| KEYBOARD | List arrows; Enter opens Evidence at seq |
| CLI PARITY | `atb incident list\|report` **VISUAL REPRESENTATION**; export pack **COMMAND PALETTE** + CLI |
| MARKET USED | Sentry issues; GitHub alerts; Splunk queue |
| REJECTED | Assign/resolve workflow; risk score; anomaly rainbow toasts (`SessionAnomalies` merge here) |

### TIMELINE

| Field | Specification |
| --- | --- |
| PRIMARY QUESTION | In what recorded order did events occur? |
| PRIMARY OBJECT | TimelineEvent (seq) |
| SECONDARY OBJECTS | Human label, type, family accent, hash |
| DEFAULT DENSITY | Dense list; no duration bars |
| PRIMARY ACTION | Select seq → Evidence inspector |
| FILTERS | Family/type (P1); search via palette |
| DETAIL BEHAVIOUR | Highlight row; inspector on Evidence or inline peek P2 |
| EMPTY STATE | “No timeline events in this bundle.” |
| ERROR STATE | Integrity gate |
| UNTRUSTED STATE | Show recorded order still (it is the claimed chain) but banner says chain invalid — **do not present as verified history** |
| KEYBOARD | j/k or arrows; Enter to Evidence |
| CLI PARITY | `atb inspect` / `events` **VISUAL**; append remains CLI |
| MARKET USED | Elastic Timeline; Datadog waterfall-as-list; Weave time scrubber without latency |
| REJECTED | Flame graph; causal connectors; graph-first; parent edges as cause |

### CONTEXT

| Field | Specification |
| --- | --- |
| PRIMARY QUESTION | What context operations and units were recorded? What retrieval happened? What is actually bound to an invocation? |
| PRIMARY OBJECT | ContextUnit / ContextOperation / retrieval.performed capability |
| SECONDARY OBJECTS | Digests, PageIndex adapter fields, warnings |
| DEFAULT DENSITY | Cards/table of units + operations; retrieval capabilities always visible if present |
| PRIMARY ACTION | Expand unit; open source event in Evidence |
| FILTERS | Kind; operation type (P1) |
| DETAIL BEHAVIOUR | Nested inspector: AVAILABLE → RETRIEVED → SELECTED → COMPACTED → ASSEMBLED → BOUND TO INVOCATION. Title is **Context evidence**. “Supplied to model” **only** if `invocation_id` binds `prompt_digest` |
| EMPTY STATE | If capabilities: “Retrieval was recorded; structured lineage units were not.” If neither: “No context lineage units or retrieval capabilities.” Never “no retrieval happened.” |
| ERROR STATE | Integrity gate |
| UNTRUSTED STATE | Show records as claimed; provenance ASSERTED unless attested |
| KEYBOARD | Tab through cards; Enter to Evidence |
| CLI PARITY | Context events are append-only **CLI-ONLY BY DESIGN** to produce; View is **VISUAL REPRESENTATION** |
| MARKET USED | LangSmith trajectory-as-projection (display only); Phoenix blank-root warning; Galileo retriever span as *typed card* not canonical ontology |
| REJECTED | “Context supplied to model” unbound; full PageIndex tree dump; context-adherence score; flowchart-first (Galileo); CoT |

**Recommended combination:** **table of operations + cards of units + retrieval capability cards**. Tree only for `parent_unit_ids`. No flow diagram as default.

### RELATIONSHIPS

| Field | Specification |
| --- | --- |
| PRIMARY QUESTION | Which records share identifiers? |
| PRIMARY OBJECT | Relationship row (kind, seqs, evidence_value, strength semantic) |
| SECONDARY OBJECTS | Optional graph of the *same* rows |
| DEFAULT DENSITY | Table first |
| PRIMARY ACTION | Jump to seq in Evidence |
| FILTERS | Relationship kind (P1) |
| DETAIL BEHAVIOUR | Graph collapsed; caption: shared identifiers, not causation |
| EMPTY STATE | “No shared-identifier relationships were derived.” |
| ERROR STATE | Integrity gate |
| UNTRUSTED STATE | Derived overlay; still not causal |
| KEYBOARD | Table is the accessible path |
| CLI PARITY | None required; derived **VISUAL** |
| MARKET USED | Weave graph as *alternative*; Datadog map optional |
| REJECTED | Graph-first; animated parent causality; dagre-in-name-only |

### EVIDENCE

| Field | Specification |
| --- | --- |
| PRIMARY QUESTION | What is the canonical record at this seq? |
| PRIMARY OBJECT | Event record |
| SECONDARY OBJECTS | Hash, type, payload fields, reveal sidecar |
| DEFAULT DENSITY | Dense table + inspector |
| PRIMARY ACTION | Inspect; copy hash; reveal if authorised |
| FILTERS | Type/family (P1) |
| DETAIL BEHAVIOUR | Summary → fields → JSON. Identity **Asserted**. Reveal writes `.reveals` |
| EMPTY STATE | “Select an event to inspect.” |
| ERROR STATE | Load error finding |
| UNTRUSTED STATE | Still show bytes as recorded; integrity failed in chrome |
| KEYBOARD | List arrows; copy hash shortcut P1 |
| CLI PARITY | `atb inspect`, `encrypt/decrypt` **CLI-ONLY**; reveal **DIRECT UI** |
| MARKET USED | Weave 3-pane; Sentry JSON package |
| REJECTED | Family rainbow labels; clientInfo as actor; replay tool calls |

### TRUST

| Field | Specification |
| --- | --- |
| PRIMARY QUESTION | Integrity? Coverage under this profile? Independent corroboration? |
| PRIMARY OBJECT | TrustSummary (three questions) |
| SECONDARY OBJECTS | Obligations, dimensions, hashes, custody |
| DEFAULT DENSITY | Strip + grouped questions; L3 expand |
| PRIMARY ACTION | Run verify if no report |
| FILTERS | None |
| DETAIL BEHAVIOUR | L3: EC…GC rows, N/A if unassessable; no eight-bar hero |
| EMPTY STATE | “No profile report — run verification first.” |
| ERROR STATE | Verify failed to run |
| UNTRUSTED STATE | Integrity INVALID; coverage omitted or labelled untrusted; Grade Insufficient for cas_score compatibility |
| KEYBOARD | Expand/collapse L3 |
| CLI PARITY | `atb verify --format json` **DIRECT UI** mapping; `--json` stays diagnostic **CLI-ONLY BY DESIGN** for the nested shape |
| MARKET USED | Vanta control vs opinion; named checks not scores |
| REJECTED | Viewer Health; radial; “Current Score”; green meaning fully assured |

---

## CLI → View parity

| Capability | Classification | View treatment | Notes |
| --- | --- | --- | --- |
| `verify` (`--format json`) | DIRECT UI ACTION | Trust + banner + palette “Verify bundle” | Stable report |
| `verify --json` | CLI-ONLY BY DESIGN | Do not bind UI to this shape | Diagnostic |
| `incident list/report` | VISUAL REPRESENTATION | Findings + Incident | |
| `incident export` | COMMAND PALETTE + CLI | Palette “Export incident pack” | Destructive? confirm |
| `inspect` / `events` | VISUAL REPRESENTATION | Evidence + Timeline | |
| `view` | DIRECT | This app | |
| `init` / `append` / `snapshot` / `sign` / `anchor` | CLI-ONLY BY DESIGN | Not replay buttons | Capture is not the viewer’s job |
| `intercept` | CLI-ONLY BY DESIGN | Document; do not fake a proxy UI | |
| `mcp serve` / `serve` | INTERNAL/DEVELOPER | Palette omit or “internal” | |
| `identity set` | INTERNAL | Omit | |
| `push` / Mortise lodge | COMMAND PALETTE (P1) or CLI | Trust custody row if receipt present | No Mortise screens |
| `evidence pack` / `compliance pack` | COMMAND PALETTE | Export; not a score | |
| `encrypt`/`decrypt` | CLI-ONLY | | |
| Historical tool events | VISUAL ONLY | Never Replay | Lead lock |

Destructive actions (export overwrite, reveal): confirm dialog. Verify is safe/idempotent.

Keyboard: ⌘K palette; arrows on lists; Esc closes palette/drawer.

---

## Mortise crossover (validate, do not redesign)

| Concern | Shared Tenon | ATB only | Mortise only |
| --- | --- | --- | --- |
| Evidence identity (bundle hash, receipt id) | DigestDisplay, TechnicalIdentifier | Event seq | Receipt id, leaf index |
| Verification status | StatusBadge VERIFIED/FAILED | Hash chain of events | Attestation of receipt; inclusion |
| Hash display | DigestDisplay | Event hash | bundle_hash, content_hash |
| Provenance | ProvenanceBadge | Event metadata | Who submitted (org/API key) — not identity proof |
| Inspectors | Inspector shell | Event inspector | Receipt inspector |
| Tables | Table chrome | Findings/Evidence/Timeline | Custody log |
| Filters | SearchInput | Event family | Receipt id / bundle hash |
| Export | — | Evidence/compliance pack | Custody pack (receipt+proof+checkpoint) |
| Context lineage / findings / agent timeline / relationships | — | ATB | Do not copy |
| Retention / WORM retain-until / witnesses / tlog / fork alert | Status vocabulary only | — | Mortise |
| Trust three questions | Coverage/integrity words if rendering embedded `verify.report.v1` | Full Trust surface | Render ATB JSON fields; do not re-score |

---

## Component freeze

Create a component only if a screen above needs it. Shared = Tenon tokens/behaviour.

| Component | SHARED TENON | ATB ONLY | MORTISE ONLY |
| --- | --- | --- | --- |
| AppShell | ✓ chrome, accent bar, ⌘K slot | Investigation body | Custody column body |
| PrimaryNav | Pattern | 7-item IA rail | Optional receipt list later |
| InvestigationHeader | — | ✓ | — |
| FindingTable | — | ✓ | — |
| FindingDetail | — | ✓ | — |
| EvidenceTimeline | — | ✓ | — |
| TimelineEvent | — | ✓ | — |
| ContextTree | — | ✓ parent_unit_ids only | — |
| ContextUnit | — | ✓ | — |
| ContextOperation | — | ✓ | — |
| RelationshipTable | — | ✓ | — |
| RelationshipGraph | — | ✓ optional | — |
| EvidenceTable | table pattern | ✓ events | receipts table uses same pattern |
| EvidenceInspector | inspector pattern | event | receipt inspector |
| TrustSummary | — | ✓ | subset: pass/coverage from embedded report |
| IntegrityCheck | status row | chain | attestation |
| CoverageBreakdown | — | ✓ | read-only ATB fields |
| ObligationList | — | ✓ | — |
| CorroborationStatus | — | ✓ | — |
| CustodyStatus | status row | link/absent | retain-until, WORM honesty |
| RawEvidenceViewer | JSON viewer | event JSON | receipt/proof JSON |
| CommandPalette | ✓ | ATB commands | Mortise commands |
| FilterBar | ✓ | type/family | hash/id |
| SearchInput | ✓ | ✓ | ✓ |
| TechnicalIdentifier | ✓ | ✓ | ✓ |
| DigestDisplay | ✓ | ✓ | ✓ |
| StatusBadge | ✓ | ✓ | ✓ |
| ProvenanceBadge | ✓ | ✓ | optional on submitter |
| EmptyState | ✓ | bounded ATB copy | “No receipts in org scope” |
| BoundedConclusion | — | can/cannot | Mortise footer non-certification |

Do **not** create: TrustScoreRadial, StatsOverview, SessionAnomalies (merge), AuditorCompliancePanel, ExecutiveSummaryPanel, FlameGraph, ServiceMap, ChatSurface, EvalScore, HealthGauge.

---

## Final gate

1. **Does the existing ATB IA survive market validation?** Yes. Incident-forensics products are finding-then-timeline-then-raw. Agent-debug products are trace-then-tree. ATB is the first family.
2. **Does Incident-first survive?** Yes (Sentry/Splunk/Elastic investigation object).
3. **Does Finding-first investigation survive?** Yes (Sentry issues, GitHub alerts, Splunk findings).
4. **Does Timeline > Graph survive?** Yes (Elastic Timeline workspace; Datadog/Weave default to waterfall/tree; graph is alternative). No blocker to invert this.
5. **Does the three-question Trust model survive?** Yes (Vanta control≠opinion; named gaps vs scores). No replacement score.
6. **Does progressive disclosure survive?** Yes (LangSmith M/T/D; Weave tabs; Sentry issue→event→JSON).
7. **Does the command-palette approach survive?** Yes (GitHub ⌘K). Keep; fix a11y.
8. **Which existing ATB UI conventions should be removed?** Viewer Health / Trust Dashboard / graph-first centre / “supplied to model” unbound / rainbow family labels / SessionAnomalies banner / compliance panels / live polling / landing copy that reveal mutates the bundle.
9. **Which professional conventions should be adopted?** Finding→evidence drill; list+inspector; named gap states; hash copy; JSON on demand; persistent header; ⌘K; table-first Relationships; integrity≠coverage in chrome.
10. **Which attractive competitor conventions must NOT be adopted?** Trace-first home; Galileo flowchart-first; flame/service maps; eval scores/playgrounds; Loop/Chat investigators; SIEM query languages; trajectory dedup; urgency dashboards; GRC certification UI.
11. **What is genuinely P0 before v1.16 freeze?** Only items that are already freeze-FAIL or make the locked grammar unusable/inaccessible: TB-UI-01/02 (Context honesty + capabilities), kill Health/graph-as-shipped-surface, three-question Trust + no coverage % on invalid chain, Findings seq→Evidence, Timeline list-first, CommandPalette keyboard a11y, Incident empty-state = Findings caveat. **No new P0 from this research** beyond those.
12. **Is F-MARKET-01 CLOSED?** Yes, as a research gap. Implementation remains GROUP 5 on the 1.16 branch.

**MARKET VALIDATION: PASS WITH NON-BLOCKING REFINEMENTS**

No BLOCKER against Lead-locked architecture. Refinements are P1 URL state, provenance badges, type filters, Mortise command palette later.
