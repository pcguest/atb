---
name: evidence-governance-ux
description: Use when changing any human-facing ATB or Mortise surface (viewer, embedded UI, governance UI, CLI output that presents evidence/governance state). Acts as a GATE over UX changes, not an autonomous design agent. Trigger on: viewer/app/router changes, governance review/decision/action UI, evidence-status copy, cross-product ATB↔Mortise handoff, terminology visible to users, empty/loading/error states, progressive disclosure, data density, accessibility of the operational product.
---

# Evidence / Governance UX Gate

A gate applied to every material human-facing change. This skill does **not**
design for you and does **not** decide product scope. It forces the change to
state exactly which layer owns the screen and to pass a bounded checklist.

## Layer ownership (must be explicit and non-collapsible)

- **EVIDENCE (ATB)** — what was recorded. Capture, canonicalise, hash, sign,
  verify, relate, locate, inspect, export, reference. ATB **never** independently
  claims truth, correctness, safety, completeness, compliance, intent,
  approval, or authority to act.
- **GOVERNANCE (Mortise)** — what the organisation makes of the evidence and
  what it decides. Reference immutable ATB evidence, organise review, expose
  explicit policy/rule versions, record authority and decisions, produce a
  **non-executing** ActionRequest. Mortise never rewrites ATB evidence, never
  treats interpretation as evidence or model output as authority, never claims
  execution occurred, never silently infers approval.
- **ACTION (Tenon)** — controlled execution and integration. **Not implemented.**
  Do not build UI that implies executable functionality.

Product sentences (use them as the north star for copy):

> ATB tells you what was recorded.
> Mortise tells you what the organisation makes of the evidence and what it decides.
> Tenon carries out what was authorised and returns evidence of what happened.

## The 18 gate questions

Every material UX change MUST answer all of these. "N/A" is allowed only with a
stated reason.

1. What **exact user question** does this screen answer?
2. Which **layer owns it**? (EVIDENCE / GOVERNANCE / ACTION)
3. Is **source evidence preserved** (not mutated, not summarised away)?
4. Can **evidence and interpretation be distinguished immediately**?
5. Is **policy/version visible** where relevant?
6. Is **authority visible** where relevant?
7. Is **progressive disclosure** appropriate (L1 orientation → L2 explanation → L3 forensic)?
8. Can an investigator **maintain continuity** (selection, return path, context)?
9. Are **deep links stable** (and honest when a target is unavailable or invalid)?
10. Are **statuses semantically honest** (no trust scores; distinct states not merged)?
11. Is it **accessible** (WCAG 2.2 AA-oriented; keyboard-complete; status not colour-only)?
12. Is **density appropriate** for professional use (fit at laptop 100%, not 80% zoom)?
13. Are **destructive/consequential actions explicit**?
14. Are **empty/loading/failure/partial states useful** and distinct?
15. Can **exact/raw information still be inspected** (Level 3 preserved)?
16. Does **cross-product navigation preserve context** at the ownership transition?
17. Are **AI/model outputs visibly bounded** (advisory; never approval)?
18. Is the design **free from dark patterns or approval nudging**?

## Prohibited patterns

- Collapsing **integrity / coverage / corroboration** into a single score.
- Any "trust score", "safe", "true", "AI verified", "definitely malicious", percentage-confidence-as-verdict.
- Presenting an **interpretation** (model/evaluator observation) as an approved outcome.
- Any surface that implies an **ActionRequest executed** (there is no executor).
- Adding an **"Approve" / action** control to ATB.
- Revealing private fields by **hover/accident**, or without explaining the consequence.
- Treating **location unavailable** as **integrity failed**, or **not recorded** as **did not occur**.
- Treating **relationship** as **causation**, or **successful parse** as **approved/correct/safe**.
- Exposing raw hash-chain/RFC/storage detail as the **first** thing a user sees.
- Fabricating an **ATB viewer URL** when a locator cannot be formed.

## Required state vocabulary (must be distinct, never one "Error")

LOADING · EMPTY · AVAILABLE · PARTIAL · NOT_RECORDED · NOT_SUPPORTED ·
NOT_CONFIGURED · UNAUTHORISED · FORBIDDEN · NOT_FOUND · INVALID ·
INTEGRITY_FAILURE · NETWORK_FAILURE · SERVER_FAILURE

ATB: "No structured context was recorded" ≠ "This evidence format does not
support structured context."
Mortise: "Receipt not found" ≠ "You are not authorised to view this receipt."
"Evidence location unavailable" ≠ "Evidence integrity failed."

## Regression invariants (safety-critical; prefer centralised copy or tests)

integrity ≠ truth · source change ≠ malicious tampering · identity assertion ≠
verified identity · model observation ≠ authority · interpretation ≠ evidence ·
decision ≠ evidence mutation · ActionRequest ≠ execution · location ≠ integrity ·
not recorded ≠ did not happen · relationship ≠ causation · successful parse ≠
approved/correct/safe · local custody ≠ independent custody.

## Output

```
UX_GATE: GO | NEEDS_WORK | BLOCK
OWNER_LAYER: EVIDENCE | GOVERNANCE | ACTION
QUESTIONS: <1..18 answers, terse>
FINDINGS: <bounded, each with screen + severity>
INVARIANTS_AT_RISK: <list or none>
```

`BLOCK` when the change would cross a layer boundary, fabricate state, imply
execution, or collapse a distinction above. `NEEDS_WORK` when the layer is
correct but questions 3–18 have gaps. `GO` only when all 18 are satisfied.
