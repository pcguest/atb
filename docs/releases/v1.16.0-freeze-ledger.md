# ATB v1.16 Freeze Ledger

## Stable baseline
v1.15.4 tag: 53ad440cc872a685a1e1580e99b74b7e3c150a63 (peeled commit; annotated tag object 6843429c60dfbc8517f7be4e852a98614727d833)
main SHA: 53ad440cc872a685a1e1580e99b74b7e3c150a63

## Candidate
branch: feat/v1.16.0-semantics
freeze-review candidate SHA: a607f244012bb8ca0faf81209cace38f73e8093d (short a607f24; F-CLAIMS-01 copy-only on ea345fa). Gold recertified PASS on this SHA. Prior gold PASS ea345fad651f7585693e35c4adc983433870e2de INVALIDATED.
merge-base: 53ad440cc872a685a1e1580e99b74b7e3c150a63
recovered branch HEAD: 63702887afb131c0ce2abe2328aaf13047cffa14 (`chore(release): prepare v1.16.0`; interrupted release-prep commit before reconciliation)
recovered working tree: DIRTY — `VERSIONING.md` modified and 14 certification reports untracked; index clean

## Continuity transition
FREEZE REVIEW: PASS WITH DOCUMENTED DEBT
Freeze reviewer: release prep NOT self-authorised
Lead decision: release prep AUTHORISED
Interrupted agent session: background results lost and not trusted until reconstructed from Git and files
Recovery result: release-prep commit found; remaining release documentation and this ledger required reconciliation
Push: NOT AUTHORISED

## Locked decisions
Architecture: CLOSED
Market validation: PASS WITH NON-BLOCKING REFINEMENTS
F-MARKET-01: CLOSED
ATB v1.15.4 Docker recovery: COMPLETE / FROZEN
Groups completed: 0–6
Intentionally deferred: F-CAS-02, GROUP 7 (cli.py / --custos), market P1/P2, DSL v2, etc. as in the freeze brief.

## Accepted debt
- F-CAS-02 Temporal-consistency treatment when timestamps are missing (DATED DEBT)
- Legacy Health / TrustScore components may remain if unreachable from shipped View
- Market P1/P2: URL ?surface=&seq=, provenance badges, richer type filters
- GROUP 7: cli.py deletion, --custos removal
- View export: CLI-only (`atb export` / `atb incident export`); no shipped View control; `AuditorCompliancePanel` unmounted. DATED DEBT / View-parity for Subagent G. Public docs do not claim View export as shipped.

## Recovery
old local main SHA: 9afbd0ac9a6b8bae0bdf7c04c80cea9bddbaf0d3
safety ref: safety/pre-v116-main-recovery-20260902T1439 (still present; points at old local main)
reason local main had diverged: A+B — two unpushed honesty-doc commits (41006e243229c5003eb5a3cfce7b442368043922, 9afbd0ac9a6b8bae0bdf7c04c80cea9bddbaf0d3) were made on local main on 2026-09-01 after checkout from tagged v1.15.4, then merged into feat/v1.16.0-semantics via 13b8ff0; never pushed; already ancestors of the candidate; no unique main-only work
operation used: git branch -f main 53ad440cc872a685a1e1580e99b74b7e3c150a63 (while on feat/v1.16.0-semantics)
post-recovery verification: HEAD/main/v1.15.4^{} all 53ad440cc872a685a1e1580e99b74b7e3c150a63; git diff --exit-code v1.15.4..main = 0; candidate SHA unchanged f9c4d6eb029d81fc689e4c7597a85793d165bc72; merge-base main HEAD = 53ad440cc872a685a1e1580e99b74b7e3c150a63; both 53ad440 and 9afbd0a remain ancestors of the candidate

## Checkpoint A
status: PASS
SHA: f9c4d6eb029d81fc689e4c7597a85793d165bc72
evidence: V116_FREEZE_CANDIDATE_STATE.md. Initial inspection FAIL (local main 2 commits ahead of tag). Authorised recovery restored main to peeled v1.15.4. All expected equalities hold; candidate SHA unchanged; versions still 1.15.4.

## Checkpoint B
status: PASS
SHA: a607f244012bb8ca0faf81209cace38f73e8093d
evidence: Semantic proof PASS on f9c4d6eb029d81fc689e4c7597a85793d165bc72 (V116_SEMANTIC_FREEZE_PROOF.md). F-CLAIMS-01 is copy-only and did not change semantic contracts. Gold recertified PASS on a607f24 (V116_GOLD_GATE_REPORT.md). Prior gold PASS on ea345fad651f7585693e35c4adc983433870e2de INVALIDATED because `web/components/Features.tsx` ships in the gold Next.js `output: "export"` webpack build (marketing landing `web/app/page.tsx`).

### Gold gate
status: PASS
SHA: a607f244012bb8ca0faf81209cace38f73e8093d
evidence: V116_GOLD_GATE_REPORT.md; log /tmp/atb-gold-a607f24.log
reason: Fresh `make gate-gold-release` for this SHA; exit 0; `✅ All gold release gates passed`. Log ancestry includes F-VIEW-01 `1208130`, F-VIEW-02 `9c6bbcc`, F-A11Y-01 `ea345fa`, F-CLAIMS-01 `a607f24` (Features.tsx sidecar copy; viewer.md Trust Dashboard removed). Live E2E `live-dashboard.cy.ts` 2/2 PASS (strict a11y included). `test:a11y` 1/1 PASS. web `npm audit --audit-level=high` PASS (1 low postcss-selector-parser only). Trivy HIGH/CRITICAL 0. gosec Issues 0. govulncheck clean. TS SDK audit 0 vulns. Hygiene/race/80.3% coverage PASS. Pins satisfied (Go 1.26.7). Firefox 154.0.1. Prior gold PASS on ea345fa INVALIDATED.

### Semantic proof (Subagent C)
status: PASS
SHA: f9c4d6eb029d81fc689e4c7597a85793d165bc72
report: V116_SEMANTIC_FREEZE_PROOF.md
cases PASS: 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15
cases FAIL: none
cases BLOCKED: none
material contract contradictions: none
notes: F-CAS-01 omit coverage_score on invalid chain confirmed on tampered rag_answer-pass copy (`cas_score=0`). MCP advertise-only 2024-11-05. PageIndex remains atb.event.rag_retrieval adapter. Python extra plaintext query is non-blocking G7, not a freeze FAIL. F-GOLD-01 (test/Cypress), F-GOLD-02 (x/crypto+grpc), F-GOLD-03 (investigation HTML encode), F-GOLD-04 (browserslist override 4.28.8), and F-GOLD-05 (dark muted token 64% L) did not change semantic contracts.

## Checkpoint C
status: PASS WITH DOCUMENTED DEBT
SHA: live View / Firefox / a11y / performance on ea345fad651f7585693e35c4adc983433870e2de (investigation `/view` not invalidated by F-CLAIMS-01 landing copy). Gold recertified PASS on a607f24.
evidence: Live View (Subagent D) PASS on ea345fa — V116_LIVE_INVESTIGATION_REPORT.md. Firefox/a11y (Subagent E) PASS. Performance (Subagent F) PASS WITH DOCUMENTED DEBT on ea345fa — V116_VIEW_PERFORMANCE_REPORT.md (10k-event UI smoothness not advertised; timeline not virtualised; graph unbounded). Debt is performance-only; not a Checkpoint C FAIL.

### Live View investigation (Subagent D)
status: PASS (P0 rerun; overall Checkpoint C PASS WITH DOCUMENTED DEBT — performance)
SHA: ea345fad651f7585693e35c4adc983433870e2de
report: V116_LIVE_INVESTIGATION_REPORT.md
browser: Cypress Electron 138 (Chromium) 7/7 against live `atb view` ports 19010–19016; cursor-ide-browser MCP did not attach
independent verification: `atb verify --format json` PTA-pass `coverage_score=0.76` `coverage_grade="Moderate coverage"`; GET overview+trust same values; tamper Trust omits `coverage_score`
cases PASS: F-VIEW-01 coverage (Incident 76%, Trust Moderate coverage); F-VIEW-02 reveal (inspector shows auditor@example.com); F-CAS-01 tamper wall; Findings→Evidence; Timeline→Evidence; TB-UI-01; TB-UI-02; Trust three questions; command palette Ctrl+K
cases FAIL: none
cases BLOCKED: none
export: DATED-DEBT (not a freeze FAIL)
notes: Gold recertified PASS on a607f24 after F-CLAIMS-01. Overall Checkpoint C PASS WITH DOCUMENTED DEBT (performance).

### Firefox (Subagent E)
status: PASS
SHA: initial 34b87a1bbc54d0795a5c3a744450526f14305a0a; F-A11Y-01 rerun ea345fad651f7585693e35c4adc983433870e2de
report: V116_FIREFOX_A11Y_REPORT.md
browser: Mozilla Firefox 154.0.1 (`/Applications/Firefox.app/Contents/MacOS/firefox`); Cypress 15.16.0 `Browser: Firefox 154 (headless)`
evidence: Initial matrix on 34b87a1 (port 18991). F-A11Y-01 rerun rebuilt embed + `/tmp/atb-v116-fa11y01` on 18992; Cypress Firefox 1/1 PASS; real Firefox `--new-instance` PID 33830. Chromium MCP not used.
notes: Overall Checkpoint C PASS WITH DOCUMENTED DEBT (performance).

### Accessibility (Subagent E)
status: PASS
SHA: F-A11Y-01 rerun ea345fad651f7585693e35c4adc983433870e2de (prior FAIL on 34b87a1)
report: V116_FIREFOX_A11Y_REPORT.md
gold cited (not re-run on ea345fa): 34b87a1 `live-dashboard.cy.ts` strict a11y PASS and `a11y.cy.ts` PASS; F-GOLD-05 dark `--raw-text-muted` 64% L; `color-contrast` still enabled
cases PASS: F-A11Y-01 Escape restores Commands (`aria-label="Open command palette"`, not BODY); trap; arrows; aria-activedescendant. Prior matrix (Tab/Enter/Space/contrast/landmarks/overflow/zoom/narrow) still cited from 34b87a1.
cases FAIL: none on ea345fa
cases BLOCKED: none for F-A11Y-01
notes: Do not weaken axe. Overall Checkpoint C PASS WITH DOCUMENTED DEBT (performance).

### Performance (Subagent F)
status: PASS WITH DOCUMENTED DEBT (this is the Checkpoint C documented debt)
SHA: ea345fad651f7585693e35c4adc983433870e2de
report: V116_VIEW_PERFORMANCE_REPORT.md
browser: Cypress 15.16.0 Electron 138 (headless) against live `atb view` ports 19120 (2k) and 19121 (10k); cursor-ide-browser MCP did not attach
fixtures: gitignored deterministic `run.atb/v116-perf/large-2k.atb` (2001 records, 1.67 MiB) and `large-10k.atb` (10001 records, 4.18 MiB); shipped PTA/incident-demo remain the realistic small case (already live-proven on this SHA)
material unusable: no
unsafe: no
mechanisms: Evidence pagination 200 + load-more **present**; graph JSON lazy on Relationships + opt-in `<details>` **present**; investigation Timeline **not** virtualised (legacy `TraceTimeline` unused); graph **unbounded**; timeline/context/relationships APIs unpaginated and prefetched on valid integrity; `useMemo` partial; no `React.memo` on list rows. Absence is not a defect.
evidence: 2k Cypress 1/1 PASS in 5 s (Incident→Findings→Timeline→Context→Relationships+graph→Evidence load-more+inspector→Trust→palette). 10k Cypress 1/1 PASS in 101 s (Cypress scanned unbounded lists; no crash/OOM). APIs: 2k timeline 2.0 ms / 401 KiB, relationships 6.8 ms / 5250 rows; 10k timeline 12.9 ms / 2.0 MiB, relationships 43.6 ms / 26250 rows (dense shared-id stress). `atb view` RSS tens of MiB (22–63 MiB observed).
cases FAIL: none
cases BLOCKED: none (MCP browser absent; Cypress Electron substituted)
notes: Prior performance write-up on 34b87a1 is INVALIDATED as the candidate. Do not advertise 10k-event UI smoothness. Overall Checkpoint C PASS WITH DOCUMENTED DEBT (this subsection).

## Checkpoint D
status: PASS
SHA: a607f244012bb8ca0faf81209cace38f73e8093d
evidence: V116_CLEAN_CONSUMER_PROOF.md SHA-alignment section. `git log ea345fa..a607f24` is only F-CLAIMS-01 (`docs/specification/viewer.md`, `web/components/Features.tsx`). Fresh clone `/tmp/atb-v116-d-a607f24` at a607f24; `make build`; quickstart `.atb`; verify `coverage_score=0.76`; View Incident 76% / Trust Moderate coverage (not 0%); tamper copy verify exit 2; CLI `atb export --format soc2 --with-verify` PASS. README/viewer.md do not tell a stranger to use Trust Dashboard or that reveal mutates the bundle. Prior ea345fa consumer PASS is history only.

## Checkpoint E
status: IN PROGRESS
SHA: a607f244012bb8ca0faf81209cace38f73e8093d
evidence: Claims hygiene (Subagent H) recertified PASS on a607f24 — V116_CLAIMS_HYGIENE_REPORT.md. Mortise boundary (Subagent I) PASS WITH DATED COMPATIBILITY DEBT on ea345fa — V116_MORTISE_CONFORMANCE.md. F-CLAIMS-01 is copy-only and does not reopen Mortise product. Do not treat Checkpoint E as PASS.

### Claims hygiene (Subagent H)
status: PASS (recertified after F-CLAIMS-01)
SHA: a607f244012bb8ca0faf81209cace38f73e8093d
report: V116_CLAIMS_HYGIENE_REPORT.md
investigation FAIL: ea345fad651f7585693e35c4adc983433870e2de (B1 Features reveal-mutates-bundle; B2 viewer.md Trust Dashboard / SessionAnomalies / 10k smoothness)
product edits this pass: none
blockers remaining: none
B1 closed: Masked Reveal Flow appends to `<bundle>.reveals` sidecar; authoritative bundle is not mutated
B2 closed: viewer.md Status/UI/Performance are Incident-first Trust (Integrity/Coverage/Corroboration); no present-tense Health / Trust Dashboard / stats strip / Profile/CAS / SessionAnomalies on `/view`; no 10k-event smoothness claim (documented debt)
new overclaim in those two files: none (Local Viewer timeline/graph/inspector and RoleSelector “role-aware panels” unchanged STALE, not FAIL)
material STALE (non-blocker, accepted): quickstart PyPI/npm `1.14.5` vs source `v1.15.2`; `web/README.md` Trust Score / polling; `web/docs/api/viewer-contract.md` Trust Dashboard shell; landing viewer as timeline/graph/inspector; workspace “immutable on disk”; RoleSelector “dashboard role”; overview empty-findings sentence weaker than View empty-state
notes: Overall Checkpoint E not PASS. Mortise evidence SHA remains ea345fa; claims SHA is a607f24.

### Mortise (Subagent I)
status: PASS WITH DATED COMPATIBILITY DEBT (Checkpoint E not overall PASS)
SHA: ea345fad651f7585693e35c4adc983433870e2de
report: V116_MORTISE_CONFORMANCE.md
mortise inspected: `pcguest/mortise` `main` `22f1d05e5935e3d3725daa08d5c9e8989a3c58db` still `require github.com/pcguest/atb v1.14.3`
cases PASS: .atb ingest bytes; verify.report.v1.schema.3 (`$id` + SHA `454d769167b27525e5bc79bd76e3621bf0ce4872f7d54af4cccaaa999d1cd9e5`); verification via `pkg/custody` (no re-score); profile fields on six `-pass.atb` fixtures; F-CAS-01 omit `coverage_score` on tampered RAG-pass copy; historical `custos.receipt.v1` on ATB clients and Mortise receipt; current Mortise naming (`--mortise`, `ATB_MORTISE_TOKEN`, `POST /ingest`); no duplicated JCS/hashing (`pkg/jcs` / `custody.Evaluate`)
cases FAIL: none
dated debt: F-MORTISE-01 Mortise pin v1.14.3 vs this SHA schema.3; GROUP 7 `--custos` aliases; Go lodge client ingest-only vs Py/TS `/verify/*`; Mortise e2e docs still describe intercept as unauthenticated
notes: Local `/Users/paddyguest/custos` is not Mortise. No Mortise product work. Overall Checkpoint E not PASS.

## Checkpoint F
status: PASS WITH DOCUMENTED DEBT
SHA: a607f244012bb8ca0faf81209cace38f73e8093d
evidence: ATB_V116_FREEZE_ADVERSARY.md + ATB_V116_FREEZE_REPORT.md. Independent freeze review on a607f24: HEAD identity matches candidate/main/peeled v1.15.4/safety ref; tracked tree clean; untracked V116_*.md / ATB_V116_*.md only. Semantic certified f9c4d6e (no contract change after); C live/firefox/a11y/perf ea345fa; Mortise ea345fa; gold/claims/consumer/adversary a607f24; F-CLAIMS-01 docs-only carry. No P0 missed. No FAIL IDs. Documented debt carried (F-CAS-02, performance, View export CLI-only, F-MORTISE-01, GROUP 7, claims STALE P1, leftover Health Cypress, F-ADV-DEBT-01/02 and 03–05). Overall freeze is not a clean PASS. Release-prep not executed (no version bump).

## Corrective commits
### F-GOLD-01
old SHA: f9c4d6eb029d81fc689e4c7597a85793d165bc72
new SHA: f71d4d4c2d487be0d676cc0c7ded1ecdd37a4ed0
DEFECT_ID: F-GOLD-01
FILES: cmd/atb/installed_binary_smoke_test.go, web/cypress/e2e/live-dashboard.cy.ts, web/cypress/support/e2e.ts
WHY: Gold smoke (and the gold-path live/a11y Cypress helpers) still required retired "Trust Dashboard" / generic Health HTML after View shipped Integrity, Coverage, and Corroboration. Stale test assertion; View Trust surface was not lost.
CHECKPOINTS INVALIDATED: Checkpoint B gold gate (semantic proofs still valid; no contract change)
CHECKPOINTS TO RERUN: gold gate only

### F-GOLD-02
old SHA: f71d4d4c2d487be0d676cc0c7ded1ecdd37a4ed0
new SHA: d8920d4d1ef07a6b87a704e4a348293cd461399a
DEFECT_ID: F-GOLD-02
FILES: go.mod, go.sum, tools/go.mod, tools/go.sum, THIRD_PARTY_NOTICES
WHY: Gold Trivy 0.73.0 HIGH/CRITICAL on CVE-2026-56854 (golang.org/x/crypto v0.53.0, fixed 0.55.0) and CVE-2026-84304 (google.golang.org/grpc v1.82.1, fixed 1.83.1 in go.mod and tools/go.mod). Minimum fixed versions only; notices regenerated. Required transitives: x/sys 0.46.0→0.47.0, x/sync 0.21.0→0.22.0 (via x/text 0.41.0), x/net 0.56.0→0.57.0, x/text 0.39.0→0.41.0, otel/metric/trace 1.43.0→1.44.0, genproto googleapis api/rpc 20260414→20260526.
CHECKPOINTS INVALIDATED: Checkpoint B gold gate
CHECKPOINTS TO RERUN: gold gate (semantic still valid; no contract change)

### F-GOLD-03
old SHA: d8920d4d1ef07a6b87a704e4a348293cd461399a
new SHA: 223c30bc7b768919fac02e17913d380053dafa69
DEFECT_ID: F-GOLD-03
FILES: pkg/api/v1/investigation.go, pkg/api/v1/investigation_test.go
WHY: Gold gosec v2.27.1 G705 XSS MEDIUM on investigation report writes. JSON now MarshalIndent at the handler so the sanitizer is visible; markdown is html.EscapeString before Write. Incident/Findings/Trust copy unchanged except HTML specials cannot appear raw in the downloaded report.
CHECKPOINTS INVALIDATED: Checkpoint B gold gate (semantic proofs still valid; no contract change)
CHECKPOINTS TO RERUN: gold gate only

### F-GOLD-04
old SHA: 223c30bc7b768919fac02e17913d380053dafa69
new SHA: 813da078ba1dfa6b0a046f97dd6655cd4e1f17ec
DEFECT_ID: F-GOLD-04
FILES: web/package.json, web/package-lock.json
WHY: Gold `cd web && npm audit --audit-level=high` failed on browserslist@4.28.1 GHSA-c83g-rgw3-j3cx / GHSA-73wf-gq98-2v4g HIGH. Transitive pin via existing npm overrides to 4.28.8 (advisory range <=4.28.6). postcss-selector-parser@6.1.2 GHSA-w9m9-85wc-3x92 remains low and does not fail --audit-level=high. ATB version unchanged. TS SDK audit had 0 vulnerabilities.
CHECKPOINTS INVALIDATED: Checkpoint B gold gate (semantic proofs still valid; no contract change)
CHECKPOINTS TO RERUN: gold gate only

### F-GOLD-05
old SHA: 813da078ba1dfa6b0a046f97dd6655cd4e1f17ec
new SHA: 34b87a1bbc54d0795a5c3a744450526f14305a0a
DEFECT_ID: F-GOLD-05
FILES: web/app/globals.css
WHY: Live Cypress-axe color-contrast failed on 11 Incident-surface nodes. Dark `--raw-text-muted` (52% L, #768493) on `--card` (#161c25) was 4.48:1, below WCAG AA 4.5:1 for small text. Raised dark muted to 64% L (~6.6:1 on card). Command palette, kbd, card labels, and finding dt/dd all used that shared token. Not a stale Health/Trust Dashboard wait.
CHECKPOINTS INVALIDATED: Checkpoint B gold gate (semantic proofs still valid; no contract change)
CHECKPOINTS TO RERUN: gold gate only

### F-VIEW-01
old SHA: 34b87a1bbc54d0795a5c3a744450526f14305a0a
new SHA: 120813091d477494b32b74acbbbec8732ad298b9
DEFECT_ID: F-VIEW-01
FILES: cmd/atb/view.go, cmd/atb/view_test.go, pkg/api/v1/investigation_test.go, web/app/view/page.tsx, web/app/view/page.test.tsx
WHY: Live Incident showed Evidence coverage 0% on verified PTA-pass (`atb verify --format json` `coverage_score=0.76`) while Trust said Not assessed. `startupProfileSummary` copied CAS overall but not `IntegrityValid`/`CoverageScore`/`CoverageGrade`; clone then treated zero IntegrityValid as untrusted and omitted coverage; zod/UI rendered omitted as 0%. Startup now copies CAS coverage when integrity is valid; Incident renders a percentage only when `coverage_grade` is present.
CHECKPOINTS INVALIDATED: Checkpoint B gold gate; Checkpoint C live View. F-CAS-01 omit-on-invalid-chain unchanged (semantic still valid).
CHECKPOINTS TO RERUN: gold gate; live View coverage scenario

### F-VIEW-02
old SHA: 120813091d477494b32b74acbbbec8732ad298b9
new SHA: 9c6bbccabac103b76e7b3454dcff23a5a225026f
DEFECT_ID: F-VIEW-02
FILES: web/components/dashboard/EventInspector.tsx, web/components/dashboard/EventInspector.test.tsx
WHY: Privacy reveal is a shipped View inspector flow (`Click to Reveal`, POST `/api/v1/privacy/reveal`). Successful reveal wrote the sidecar but inspector JSON stayed `[REDACTED]` because overlay keys used `data.email` against `event.data`. Bind now stores the stripped payload path. History note: 29cda1aa68a5115030516d1abd409b0285bdeb67 accidentally committed unrelated CommandPalette files and was reverted by 11a7b0531246ef1b9e940878512c3180e24bab10 — net zero on palette; this SHA is the actual inspector bind.
CHECKPOINTS INVALIDATED: Checkpoint B gold gate; Checkpoint C live View
CHECKPOINTS TO RERUN: gold gate; live View privacy-reveal scenario

### F-A11Y-01
old SHA: 9c6bbccabac103b76e7b3454dcff23a5a225026f
new SHA: ea345fad651f7585693e35c4adc983433870e2de
DEFECT_ID: F-A11Y-01
FILES: web/app/view/components/CommandPalette.tsx, web/app/view/components/CommandPalette.test.tsx
WHY: Escape closed the command palette but left document.activeElement on BODY, not the Commands trigger (`aria-label="Open command palette"`). Save the opener on open; restore it on Escape/close. Existing focus trap, arrow-key navigation, and aria-activedescendant unchanged. Vitest: CommandPalette.test.tsx 2/2 PASS. Checkpoint C is not marked PASS.
CHECKPOINTS INVALIDATED: Checkpoint C accessibility
CHECKPOINTS TO RERUN: Checkpoint C accessibility (Firefox keyboard: Escape restores focus to Commands). Gold axe not invalidated.

### F-CLAIMS-01
old SHA: ea345fad651f7585693e35c4adc983433870e2de
new SHA: a607f244012bb8ca0faf81209cace38f73e8093d
DEFECT_ID: F-CLAIMS-01
FILES: web/components/Features.tsx, docs/specification/viewer.md
WHY: Public claims contradicted shipped behaviour. Landing Features said privacy reveal appends an audit event to the bundle; actual is `<bundle>.reveals` sidecar (F-VIEW-02). Public viewer spec present-tense shipped Trust Dashboard chrome (`SessionAnomalies`, stats strip, Profile/CAS panel) on `/view` and required 10k-event smoothness; frozen View Trust is Integrity/Coverage/Corroboration and performance debt forbids advertising 10k smoothness. Copy-only. Investigation behaviour unchanged. Non-blocker STALE left untouched.
CHECKPOINTS INVALIDATED: Checkpoint B gold gate (Features.tsx is in the gold Next.js export via marketing `web/app/page.tsx`); Checkpoint E claims. Checkpoint C live View not invalidated (landing is not investigation `/view`; viewer.md is docs). Semantic proofs still valid.
CHECKPOINTS TO RERUN: gold gate; claims hygiene

## Release-prep checkpoint
EXECUTED (local)

## Release preparation

freeze candidate:
a607f244012bb8ca0faf81209cace38f73e8093d

release-prep SHA:
PENDING (the interrupted `63702887afb131c0ce2abe2328aaf13047cffa14` commit will be superseded by the reconciled amend)

scope:
version markers, generated OpenAPI version artefact, CHANGELOG, VERSIONING, and freeze-ledger bookkeeping only

local recertification:
PENDING

## Security delta

baseline CodeQL open High alerts:
18

baseline date observed:
11 July

security delta report:
V116_CODEQL_SECURITY_DELTA.md

result:
FAIL (read-only audit established inherited release-blocking defects; remediation must be a separate security commit)

new High:
0

inherited High:
18

unresolved:
0

true positives:
11 (#2 and #8-#17; #8-#17 are duplicate sinks of one HTTP-controlled path defect)

false positives / bounded non-security flows:
7 (#1, #3-#7, #18)

hosted exact-SHA CodeQL:
PENDING

## Hosted CI
NOT STARTED

## PR
NOT STARTED

## Merge
NOT STARTED

## Publication
NOT STARTED
