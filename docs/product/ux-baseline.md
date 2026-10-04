# Product UX Baseline & Architecture Decision (Phase UX-0)

Status: **ARCHAEOLOGY / DESIGN CONTRACT** — no product code. Derived from live
code inspection of ATB (`origin/main` `0fa2da4`), Mortise (`origin/main`
`9ddb13d`), and Tenon (`07cf627`). Temporary archaeology; promote/replace when
phases land. Do not treat as canonical evidence/governance contract — link to
those instead.

---

## 1. Live state at time of writing

- ATB `origin/main` `0fa2da418d3e4715a9769a80c6628de1cada3fd9`; latest release
  tag `v1.16.0`. Open PRs: #41 (security, BLOCKED), #38 (manifest v3, BLOCKED),
  #39/#40 (CLEAN, stacked), plus stale #28 (DIRTY), #27/#22/#18 (BEHIND).
- Mortise `origin/main` `9ddb13d09ebb6bbefa0d3c85ec382c66acb6fe17`; **local
  `main` `fc2a27d` is not at origin/main**. Open PRs #7/#8 (CLEAN). ATB pin
  `github.com/pcguest/atb v1.16.0`. Active `go.work` (`use (. ../atb)`) present
  locally, gitignored.
- Tenon `07cf627`: **docs-only** (README, docs/, evaluation/ metadata, Python
  doc/contract validators). No module manifest, no runtime, no ActionRequest
  acceptor. Programme state Phase 0 "foundation-established". With no runtime,
  the programme-level `TENON_IMPLEMENTATION_GATE` stands at `NO_GO` (see D7 and
  §8); it is a programme gate, not a Tenon code artifact.
- Security: `braces` `GHSA-vfj7-8cjw-p6xm` remains the sole web HIGH root
  (dev/build-only, no upstream fix); #41 `Node Security` red, `Trivy FS` green.
- `ActionRequest` has **no HTTP create route** (Mortise H-MOR-01 open); H-MOR-02
  (ingest→governance resolution) appears fixed.

---

## 2. Product architecture decision (ADR summary)

**Context.** ATB ships one binary with an embedded, offline, static-export
viewer (`go:embed web/out`; `/`, `/view`, `/sessions`, `/workspace`). Mortise is
a Go daemon with server-embedded static HTML (`ui.html` auditor,
`governance.html`) and a JSON governance API. Tenon is non-runtime docs.

**Decision (proposed, for review).**

- **D1 — Separate concerns, not one app.** Public ATB presence (`/`), the ATB
  investigation application (`/view` + operational `/sessions`, `/workspace`),
  and the Mortise governance application are distinct products. Do not merge
  marketing/onboarding and evidence inspection merely because both are web.
- **D2 — ATB local viewer stays independently usable and offline.** No mandatory
  cloud calls, analytics, remote fonts, or Mortise dependency. The viewer must
  remain useful with only the bundled Go server.
- **D3 — Multiple deployments allowed; no premature infrastructure.** Keep
  embedded `/` as a lightweight product/onboarding entry point for now.
  Evaluate an independently deployed public site **only** in Phase UX-6, and
  only if operational IA has stabilised. Documentation stays
  repository/static-doc based.
- **D4 — Family resemblance, not release coupling.** Share *conceptual* tokens
  (status vocabulary, object-identity header, terminology) and copy them per
  repo. Do **not** create a shared package or monorepo in Phase UX-0. Reassess
  after UX-1/UX-3 (Candidate: A conceptual tokens only → B copied/versioned
  primitives → C shared package; prefer A/B).
- **D5 — Cross-product handoff is explicit and context-preserving.**
  ATB→Mortise: "Review in Mortise" only when a valid immutable
  `EvidenceReference` + configured destination + explicit contract exist;
  otherwise honest "Governance destination not configured". Mortise→ATB: "Open
  evidence in ATB" via the canonical locator
  (`atb://evidence/1/<head>?seq=<n>[&record=<hash>]` → `?focus=<locator>`),
  never a fabricated URL; location unavailable ≠ invalid evidence.
- **D6 — Ownership transition must be visible.** Moving ATB→Mortise reads as
  entering governance; Mortise→ATB as inspecting immutable evidence. Never
  present as undifferentiated tabs.
- **D7 — Tenon surface gated.** No Tenon UI until `TENON_IMPLEMENTATION_GATE`
  changes. `ActionRequest` surfaces must state "No execution has occurred."

---

## 3. Route / surface maps

### ATB (Next.js 16 App Router, static export, `trailingSlash: true`)

| Route | Renders | Class |
|---|---|---|
| `/` | Homepage (Navbar/Hero/Features/CodeDemo/CurrentScope/Footer) | public product |
| `/view` | Single-page investigation: Incident(Run) / Findings / Timeline / Context / Relationships / Evidence / Trust | public forensic |
| `/sessions` | SessionList, SchemaStatus, ActorSessions | operational |
| `/workspace` | Agent workspace bundle table (agent-mode gated) | operational |

- Investigation state is **local to one page**: surface/`selectedSeq`/
  `selectedFinding`/`source` are `useState`, not routes. Only `?focus=<locator>`
  and `#session=<token>` are URL-encoded. `popstate`/`hashchange` resets
  selection.
- Trust surface keeps Integrity / Coverage / Corroboration separate; no score.

### Mortise (single Go daemon, embedded static HTML)

- UI shells: `GET /` (auditor), `/ui`, `/governance/ui`.
- Custody/ingest: `POST /ingest`, `/ingest/presign`, `/ingest/notify`.
- Verification: `POST /verify/bundle`, `/verify/receipt`, `GET /receipts/{id}/timestamp`.
- Receipts/custody: `GET /receipts`, `/receipts/by-hash`, `/receipts/{id}`, `/receipts/{id}/proof`.
- Log: `GET /checkpoint`, `/log/key`, `/log/consistency`, `/log/witnesses`, `/log/feed`, `POST /log/cosignature`.
- Governance: `GET/POST /governance/reviews`, `GET /governance/reviews/{id}`,
  `GET .../evidence`, `POST .../review`, `POST .../decision`,
  `GET .../action`. **No `POST .../action`.**

---

## 4. Terminology map (audit)

| Term | Canonical meaning | User-facing label | Must NOT imply |
|---|---|---|---|
| run / bundle / session | ATB capture container (on-disk bundle) vs live session | "Investigation" / "Bundle" (explain object) | that it is complete/true |
| evidence / record / event | exact canonical recorded unit | "Record" / "Evidence" | truth or completeness |
| finding | bounded ATB observation | "Finding" | an action recommendation |
| integrity | hash-chain/manifest validity | "Hash chain verified/failed" | truth, safety, correctness |
| coverage | whether expected signals were observed | "Coverage: Not assessed / …" | completeness |
| corroboration / custody | external custody evidence | "Corroboration: Absent from recorded evidence" | that absence = didn't happen |
| source | origin of captured material | "Source" | that a changed source is malicious |
| profile | capture profile id | "Profile (not selected)" needs explanation | an authorisation control |
| receipt | Mortise custody attestation | "Receipt" | ATB truth |
| review | append-only human governance act | "Review" | execution |
| interpretation / observation | bounded advisory model/evaluator output | "Interpretation (advisory)" | authority/approval |
| policy / rule | versioned deterministic basis | "Policy @version / Rule @version" | model judgement |
| decision | authorised outcome record | "Decision" | evidence mutation |
| approval / authorisation | authority state | "Approval: pending/approved/rejected" | execution |
| action request | non-executing request object | "Action request (not executed)" | that anything ran |
| execution | Tenon runtime (absent) | do not surface as available | that it happened |

Rule: improve **explanatory labels** before changing domain names.

---

## 5. Friction ledger (confirmed against code)

### ATB
- Weak global **object identity**; `run.atb/bundle.atb` dominates without
  explaining the object.
- **Profile: Not selected** unexplained; **Custody: Local only** lacks context.
- First-use orientation thin; nav items equally weighted.
- Timeline surfaces **technical metadata too early**; hashes compete with
  human-readable content.
- **Context** empty state occupies large area with little recovery/navigation.
- **Relationships** semantically careful but mechanically table-like.
- Evidence vs Timeline **duplicate list/detail** patterns
  (`SplitWorkspace` only used by Findings/Record).
- **Evidence status** semantics excellent but text-heavy.
- Some copy reads as implementation documentation.
- **Cross-product handoff invisible.**
- Deep links limited to `?focus`; selection not URL-encoded.
- UX-0 addendum: hardcoded surface taxonomy duplicates labels across page/
  tests/spec/cypress; unvirtualised long lists (documented debt,
  `docs/specification/viewer.md:319`).

### Mortise
- Root page is an **engineering utility**, not a governance product.
- Raw **JSON textarea** is first interaction; **API base/key dominate** cognition.
- Invalid/empty JSON yields an implementation error; **lookup 404** exposed
  without recovery context.
- **Organisation context implicit**; no evidence-object/review/decision
  orientation on landing.
- No visible **ATB evidence handoff**; no hierarchy between ordinary governance
  workflow and specialised custody verification.
- Technical fields **not progressively disclosed**; large unused canvas at
  desktop width; weak product identity.
- No real browser/a11y harness (static `ux-gate.sh` + DOM-contract Go tests).

---

## 5a. Validated screen findings (UX-0 validation, independent practitioner)

A synthetic practitioner (P4 auditor / P2 governance reviewer) ran the real
surfaces (Mortise on `:9095`; source-built ATB viewer on `:18080`) plus code.
Findings were confirmed/refined against the Phase UX-0 archaeology and code
inspection above:

| # | Sev | Finding | Owner |
|---|---|---|---|
| V1 | HIGH | Mortise governance has **no UI path to start a review** (empty queue; no create-observation control). The §6 journey is therefore not reachable from the UI alone; it requires crafting `POST /governance/reviews` directly. | GOVERNANCE |
| V2 | HIGH | The **installed ATB binary without an embedded build serves only "UI not available"**; practitioners must rebuild from source to get the viewer. | EVIDENCE |
| V3 | MED | ATB Run **"Evidence coverage 80%"** can read as capture completeness; the profile-scoped bound lives only on Evidence status. | EVIDENCE |
| V4 | MED | Mortise queue **"Success"** (observation status) is the loudest signal and can be misread as evidence/governance validity. | GOVERNANCE |
| V5 | MED | ATB human-approval event renders "Human approval recorded" **without who** when only `approver_id_hash` is present; dual `ai.`/`atb.` wire types. | EVIDENCE |
| V6 | MED | Mortise auditor **"ATTESTATION VERIFIES" + "profile pass" + CAS grade** on one screen risks truth/compliance over-read; transparency path silently 404s on this deploy. | GOVERNANCE |
| V7 | MED | ATB Evidence/Timeline open **full canonical JSON + hashes at selection default** (forensic depth before orientation). | EVIDENCE |
| V8 | LOW | ATB surface switches do not retain **search/filter or nav return-path**; locator focus without `record=` reports `record_hash_matched:false` while still selecting seq. | EVIDENCE |

These **confirm** most of the Phase UX-0 archaeology and add V2/V4. None change the product-boundary
architecture; V1 is a product-coverage gap, not a doctrine change.

## 6. Current user journeys

- **ATB**: open bundle → orient on Run (integrity/findings) → investigate
  Timeline/Context/Relationships → inspect exact Evidence → read Evidence
  status → (missing) handoff to governance.
- **Mortise**: land on auditor → paste JSON / set API base+key → verify/lookup
  receipt → (governance reachable via separate `/governance/ui`, not the
  landing) → create review → append review → decision → read action.
- **Cross-product**: only the Mortise→ATB locator link is implemented; ATB→Mortise
  is absent.

---

## 7. Progressive-disclosure model (target)

- **L1 Orientation**: object, state, source, time, important issue, next action.
- **L2 Explanation**: relationships, provenance, policy, interpretation, review
  history, acquisition, evidence gaps, decision basis.
- **L3 Forensic**: exact ids, hashes, canonical JSON, signatures, schema
  versions, evaluator metadata, canonicalisation, lineage, raw receipt/evidence.

Current products frequently expose **L3 before L1**.

---

## 8. Gates

- `PRODUCT_UX_ARCHITECTURE_GATE` — this document (Phase UX-0 archaeology);
  independent review is pending a linked report.
- `ATB_INVESTIGATION_UX_GATE`, `ATB_FORENSIC_UX_GATE`,
  `MORTISE_GOVERNANCE_UX_GATE`, `MORTISE_REVIEW_UX_GATE`,
  `ATB_MORTISE_PRODUCT_JOURNEY_GATE`, `ATB_PUBLIC_WEB_GATE` — **NOT_STARTED**.

Engineering: `ATB_RELEASE_GATE` NO_GO · `RELEASED_INTEROP_GATE` NO_GO ·
`ATB_ACQUISITION_PRODUCTISATION_GATE` NEEDS_MORE_FOUNDATION ·
`TENON_READINESS_GATE` NO_GO · `TENON_IMPLEMENTATION_GATE` NO_GO.
