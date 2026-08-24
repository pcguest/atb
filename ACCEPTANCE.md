# ATB and Mortise acceptance evidence (v1.15.0)

> **Historical release evidence for v1.15.0. Not the current release status.**
> Commands, observed results, toolchain versions, and product names below are
> preserved as evidence of that release review.

Every load-bearing claim, the command that proves it, and the result observed on
this machine. Toolchain: `GOCACHE=$(pwd)/.gocache/go1.26.5 GOTOOLCHAIN=go1.26.5`
(go.mod requires go 1.26.5; the older 1.26.3 in earlier notes is stale). This is
evidence, not certification. The honest limits at the end are part of the claim.

## Gates

| Claim | Command | Result |
| --- | --- | --- |
| Cross-language verifiers agree | `make test-golden` | Go, Python (8 tests), TypeScript (8 tests) all pass; "Golden vectors verified across Go, Python, and TypeScript". |
| Full Go suite passes | `go test ./...` (ATB), then in the separate Mortise repository | ATB: 40 packages ok, 0 fail. Mortise: 9 ok, 0 fail. |
| Hygiene gate passes | `make hygiene-quick` | fmt, vet, staticcheck, full test-go, web lint and typecheck, govulncheck: exit 0. |
| Version markers agree | `ATB_SKIP_TAG_CHECK=1 bash scripts/check-versions.sh` | "ok: all version strings agree (1.15.0)", exit 0. The nine markers (Go, Python x2, TypeScript x2, web x2) are all 1.15.0. Registries are still 1.14.5 until publish. |

## Quickstart on a fresh checkout

| Claim | Command | Result |
| --- | --- | --- |
| A clean clone builds | `git clone --branch release/v1.15.0 ~/atb c && cd c && go build ./cmd/atb` | Builds `atb 1.15.0`, exit 0. (Before the embed-placeholder fix this failed with "pattern web/out/*: no matching files found"; that regression is now fixed and re-proven.) |
| Bundle, capture, verify | `atb bundle new; atb capture run -- ./a.sh; atb verify run.atb/bundle.atb` | Bundle initialised, capture ran, verify Integrity PASS, exit 0. |
| Verify exits 2 on tamper, 0 intact | mutate a non-manifest record, then `atb verify` | Intact: exit 0. Tampered: "tamper detected at event 2 (seq 2)", exit 2. |
| `go install` stub view page | `atb view` on a clean build | Serves the install-guidance page ("build from a checkout"), as documented. |

## Viewer reveal does not mutate evidence

| Claim | Command | Result |
| --- | --- | --- |
| Authoritative bundle unchanged by a reveal | `shasum -a 256` before and after a successful reveal | Identical: `32b24af0...c61c49` before and after. The reveal returned the value `auditor@example.com`. |
| Reveal recorded in an independent sidecar | `atb verify <bundle>.reveals` | Sidecar exists; Integrity PASS, exit 0. |

## Incident forensics from the bundle alone

| Claim | Command | Result |
| --- | --- | --- |
| Unapproved tool use is provable | `atb incident list` and `atb incident report` | `incident list` flags `tool_without_approval, action_failed`. `incident report` shows HIGH "Tool call with no preceding approval" at seq 2 and MEDIUM "Action error recorded" at seq 5. |
| Tampering is pinpointed | `atb verify` on a tampered bundle | "tamper detected at event 2 (seq 2): expected ... got ...". |

## The default-drift guard actually fires

| Claim | Command | Result |
| --- | --- | --- |
| Drift in an SDK default fails the gate | drift the Python `policy_id` default on a throwaway, run the golden test | FAIL: `'policy_id': 'remote.workflow' != 'local.workflow'` in `test_policy_decision_defaults_match_shared_fixture`. Reverted; tree clean. |

## Schema decision: reviewer_identities is earned, not WIP

`reviewer_identities` is kept. Evidence:

- Wired: `atb verify --format json` populates it (`cmd/atb/verify.go` calls `ReportFromVerifyWithBundle`). Also consumed by trust reports, compliance packs, evidence packs, custody export, and the viewer API.
- Tested: `TestReportFromVerifyWithBundle_IncludesReviewerIdentityEvidence` passes; emitter-contract, compliance-pack, incident, and schema-frozen tests pass.
- Strict additive: `git diff v1.14.5 HEAD -- pkg/custody/schema/verify.report.v1.schema.json` shows exactly two changes, the `$id` `.1` to `.2` and the new optional `reviewer_identities` array. No existing field or constraint changed, and it is not added to the top-level `required` list. `report_version` stays `verify.report.v1`.
- Frozen: the schema is pinned by SHA-256 in `pkg/custody/schema_test.go` (`ed25fdbc...`).
- Honest by construction: the field's `verification` value is the constant `caller_provided_unverified`.

Honest limit on this item: there is no JSON-schema instance validator in-tree (no `jsonschema` dependency, not added in an acceptance pass). Coverage of both shapes is at the producing layer (with-field via `ReportFromVerifyWithBundle`, without-field via `ReportFromVerify` plus `omitempty`), and the schema document is hash-frozen.

## Mortise conformance and the end-to-end path

| Claim | Command | Result |
| --- | --- | --- |
| Mortise verifies bundles against the contract | `go test ./internal/verifyingest/...` | ok. |
| Full custody path works | `scripts/e2e-atb-mortise.sh` | PASS: ATB verify, ingest, Ed25519 receipt, RFC 6962 inclusion proof, witness cosignature, offline verify-inclusion (six checks), credential-free monitor. |
| Conforming CLI push (live) | `atb incident export --mortise-endpoint` against a real `mortise-ingestd` | "lodged bundle with Mortise: receipt sha256-... (bundle hash ...)", daemon logged `POST /ingest 201`. |
| Article 12 evidence pack exports | `atb compliance pack --regime eu-ai-act` | Wrote a 16-file pack with `verify.report.json`, `MANIFEST.json`, `SHA256SUMS`. |

## Repository and history hygiene

| Claim | Command | Result |
| --- | --- | --- |
| No secrets in history | `gitleaks detect --log-opts=--all` on a rewritten clone | "no leaks found", 886 commits. |
| History rewrite is safe | `git filter-repo --path .atb-agent --path-glob '.gocache*' --invert-paths` on a mirror | 121.45 MiB to 4.13 MiB; gosec, `.gocache*`, `.atb-agent` gone; 36 branches and 25 tags survive; rewritten tree builds. Not run on the real repo. |
| Disclosure contact is resolved | `grep -rn 'replace-me.example'` | Security disclosure on both repos points to GitHub private vulnerability reporting; no email is exposed on a security surface. The conduct report uses the same private channel; the Mortise site sales CTA uses the proton stopgap. The placeholder remains only in this runbook and these notes, which describe the change. The personal address otherwise stays only in package author metadata, by design. |

## Mortise site static audit

Self-contained `site/index.html`: 0 em dashes, no ampersand glyphs, no external
assets or JavaScript (`grep` for `<script`, `src=`, CDN, fonts: none), `lang="en-GB"`,
viewport set. No certification claims; the only "certificate/compliant" words are
the honest-limit negations. The open-core table matches the ATB README content
exactly. Accessibility gaps to note: no `<main>` landmark and no skip-to-content
link; the verification ticks rely on adjacent text for meaning. No headless
browser was available to screenshot. Preview with
`python3 -m http.server --directory site 8000`. Above the fold: the Article 12
pill, the headline "Provable evidence of what your AI agents did", the lead
paragraph, and two buttons. Below: the problem, what Mortise does, the
"what a third party can verify without trusting Mortise" table, how it fits, the
open-core boundary table, the honest-limits panel, and the design-partner
contact.

## UI and UX heuristic audit: closed

The full audit is in `UI-HEURISTIC-AUDIT.md`. Every finding is closed in code, or
judged by description and recorded as residue. Findings closed at these commits:

| Finding | Closed at | Evidence |
| --- | --- | --- |
| A1 reveal copy contradicted the sidecar fix | 18049fe | Tooltip and caption now say reveals write to the independent `.reveals` sidecar; the authoritative bundle is byte-unchanged, proven above. |
| A2 dead DashboardShell with a duplicate landmark | b243d69 | File deleted; the orphaned sidebar state it was the only consumer of is pruned. |
| A3 tamper banner gave no locus | eb80049 | The FAIL banner now shows the failing record, the chain length, the bundle path, and the `atb verify` re-check command, taken from the verification response. |
| A4 inspector used hardcoded slate utilities | 18049fe | Every `slate-*` is replaced by a semantic token (`bg-card`, `border-border`, `text-foreground`, `text-muted-foreground`, `bg-muted`, `hover:border-ring`); zero `slate-*` remain, so the inspector tracks the theme in both modes. |
| A5 duplicate Inspector heading | 18049fe | The inner heading is removed; the rail header is the single one. |
| A7 raw-data rails read as broken for non-engineer roles | 9006425 | The timeline, the graph, and the inspector state they are available in the Engineer role; the layout structure is unchanged across roles. |
| A8 raw lowercase verification value | f2780d5 | Shown as Valid or Invalid with the banner lock and shield icons. |
| A9 Viewer Health read as a third integrity verdict | f2780d5 | A caption states it is a composite of verification and recency, not the integrity result. |
| B1 first-person-plural CTA | fc316cb | Reworded to "Discuss a design partnership". |
| B2 no main landmark or skip link | fc316cb | Content wrapped in `<main id="main">` with a skip-to-content link. |
| B3 tick glyphs without a text alternative | fc316cb | Glyphs marked `aria-hidden`; meaning is in the adjacent text. |
| B4 table headers without scope | fc316cb | `scope="col"` added to both headers. |

Contrast was judged on the real compiled colours, not by eye. In the FAIL banner the
text red-300 (rgb 252 165 165) on red-950 (rgb 69 10 10) is 8.5 to 1, and red-200
(rgb 254 202 202) on the same ground is 11.2 to 1. Both exceed WCAG 2.1 AA, which
is 4.5 to 1 for normal text and 3 to 1 for large text. The light-theme question
(A4) was settled the same way: a `grep` confirms no `slate-*` remains in the
inspector, so every surface resolves through the semantic tokens that flip with
the theme.

### Residue judged by description, still wanting a human glance

These three are cosmetic on a localhost tool, or a single Mortise-fold judgement.
None bears on trust. Each was assessed from the CSS and the runtime, not fixed
blind.

- A6 narrow-window layout: there are no responsive breakpoints, so below roughly
  768px the centre column is squeezed and `overflow-hidden` clips. Rule 4 as
  written holds, because the layout is one structure across all roles. Whether a
  responsive stack is worth adding wants a glance.
- TraceGraph label legibility: nine nodes on a fixed three-column grid at 260px
  spacing, labels 18 to 26 characters, `fitView` on. Overlap is unlikely because
  the spacing exceeds the node width, but it cannot be measured without a render.
- Mortise above-the-fold hierarchy: the hero leads with a four-sentence paragraph
  and carries no artefact; the first artefact is the verify table further down.
  Whether that reads as too much at the real fold wants a glance.

## Honest limits (verbatim, unchanged)

From the ATB README:

> ATB proves integrity of what was recorded. It does not prove universal
> capture, model correctness, actor identity, or legal compliance by itself, and
> it never certifies compliance.

From the Mortise site:

> Mortise produces evidence. It does not issue certificates, and nothing here is
> a certification or a legal opinion.

> Equivocation resistance is bounded by the number and independence of the
> witnesses actually deployed. It is a detection property, not an absolute.

## Open items (yours, irreversible; see LAUNCH-RUNBOOK.md)

Set the security mailbox; push branches; tag `v1.15.0` once; make repos public;
decide Path A or B on the history rewrite; publish the SDKs; deploy the site.
