# External design-partner pilot protocol

This document defines **how to run** the first external design-partner pilot of
the released `atb intercept` continuous-capture path (v1.18.0) and **how to judge
its outcome**. It is a protocol, not a product, and it is not evidence of
commercial value. Read it with
[Continuous capture with `atb intercept`](../capture/continuous-capture.md) and
the [pilot kit](continuous-capture-pilot-kit.md).

The pilot tests one question:

> Does a real practitioner, in a real organisation, have a sufficiently painful
> problem that this released capability materially improves — and will their
> organisation accept the operational, security and commercial trade-offs
> required to use it?

The answer may be no. That is a valid and useful result.

## What this protocol does not do

- It does not manufacture a pilot. If no genuine external participant is
  available, the result is `DESIGN_PARTNER_PILOT_RESULT = NOT_RUN`.
- It does not substitute a synthetic practitioner for a human one, and does not
  treat a synthetic result as independent validation.
- It does not treat a completed workflow, or positive usability feedback, as
  commercial proof.
- It does not defend the product during interviews.

## Roles

- **Design partner** — the external organisation.
- **Participant practitioner** — the person doing the work. Must be independent:
  not an ATB contributor, not the person who built the capture path, and not
  coached through the interface.
- **Observer** — records the session and any intervention. May answer
  environment/setup questions only. Every intervention must be recorded (what was
  asked, what was answered). Coaching invalidates the affected task.

## Participant criteria

Preferred profiles: AI security engineer, incident responder, digital forensic
investigator, security assurance practitioner, AI governance reviewer,
platform/security engineer.

Required:

- works with consequential AI/agent workloads;
- can supply a safe, restartable test workload with a known observation boundary;
- has authority to evaluate and to speak to deployment, security and procurement.

Excluded: anyone who has built or reviewed this capture path, or who has seen the
expected answers in this protocol.

## Bounded workload scenario

Use **one** workload and **one** bundle. Do not expand to a connector catalogue.
The workload should have: meaningful agent activity; at least one model
invocation; at least one tool operation; one consequential but safe unexpected
operation; known expected behaviour; a known observation boundary; and the
ability to restart.

Run this sequence:

    normal activity
      -> unexpected tool operation
      -> continued activity
      -> capture interruption or degradation
      -> recovery
      -> investigation
      -> export / handoff

State explicitly which paths are **unsupported** (for example, clients that pin
certificates or bypass `HTTPS_PROXY`). Do not manufacture universal coverage.

## Operator tasks (A–K)

Give the participant the incident objective, not the solution path. Record for
each task: attempted, outcome, assistance required, and the evidence they relied
on.

| Task | The participant must | Probe |
| --- | --- | --- |
| A — Orient | identify the object under investigation, what was captured, and the evidence scope | — |
| B — Integrity | explain what the integrity result establishes | does the participant infer integrity = truth? |
| C — Capture boundary | identify when observation began, what was observed, known gaps, and unavailable history | does the participant infer no record = did not happen? |
| D — Incident | locate the unexpected operation | — |
| E — Evidence | retrieve the exact supporting record | — |
| F — Timeline | explain the recorded sequence without claiming causality | does the participant infer sequence = causality? |
| G — Relationships | explain what relationships do and do not establish | does the participant infer relationship = causation? |
| H — Recovery | determine the observable recovery state (restarted/degraded) and whether replay can be established from the available output | does the participant infer replay = no loss? |
| I — Export | produce an independently inspectable evidence package | — |
| J — Verification | verify it independently of the running capture process | — |
| K — Handoff | where Mortise is in scope, reach the same evidence from governance | does the participant infer decision = execution? |

## Evidence collection template

A pilot is only useful if it produces a bounded, comparable evidence package.
Collect the following, excluding secrets. A blank template is in
[pilot evidence and interview](pilot-evidence-and-interview.md).

- **Environment** — ATB version; binary checksum; OS/platform; workload
  description; capture configuration (no secrets); capture start boundary; source
  incarnation; relevant versions.
- **Capture** — start; status; observations; checkpoint; recovery/replay state;
  known gaps; degradation; final bundle.
- **Incident** — the seeded or encountered unexpected operation; exact evidence
  reference; relevant relationships and context; source/observation limitations.
- **Verification** — offline verification result; bundle identity; evidence
  locator; export result.
- **Governance** — where Mortise is used: immutable `EvidenceReference`,
  resolution result, visible capture boundary, and confirmation that ATB evidence
  was not mutated.
- **Practitioner** — tasks attempted, outcome, confusion, semantic mistakes,
  assistance required, and evidence relied upon.
- **Commercial** — existing workflow, current alternative, perceived value,
  operating burden, security/privacy objections, deployment objections,
  integration objections, willingness to continue, champion, and pay/procure.

Do not conflate usability with willingness to buy.

## Success and failure gates

`DESIGN_PARTNER_PILOT_RESULT = PASS` only if **all** hold:

- a real external participant used the released v1.18.0;
- a bounded real workload was captured;
- capture was successfully operated;
- at least one meaningful investigation was completed;
- gaps and limitations were correctly understood;
- evidence was independently verified;
- no material semantic misunderstanding was caused by the product;
- no unresolved pilot-blocking security defect;
- the practitioner identifies meaningful value;
- the participant is willing to continue evaluation.

Purchase is **not** required for `PASS`.

`MIXED` — the system technically works but value is unclear, burden is high, the
workflow does not fit, major integration changes are requested, or champion
interest is weak.

`FAIL` — the workflow cannot be completed, evidence semantics are materially
misunderstood, security/privacy blocks use, capture reliability is inadequate,
the product solves no meaningful problem, or the participant would not continue.

## Commercial evidence ladder

Keep these distinct; do not infer payment intent from positive usability.

    C0  internal engineering proof
    C1  independent practitioner completes the workflow
    C2  practitioner states a meaningful problem was solved
    C3  organisation agrees to another / expanded pilot
    C4  organisation commits resources or integration effort
    C5  organisation demonstrates procurement / payment intent
    C6  paying production deployment

Do not claim product-market fit from C1 or C2. `COMMERCIAL_PROOF_GATE` stays
`NOT_TESTED_EXTERNALLY` until external evidence exists, and advances no faster
than `EARLY_SIGNAL` -> `DESIGN_PARTNER_VALIDATED` -> `PROCUREMENT_SIGNAL` ->
`PAID_VALIDATION`.

## Pre-pilot corrections

The following are known gaps in the current kit that should be corrected **before
a kit is sent**, so the participant is not misled:

1. **Source incarnation and evidence.** The capture documentation states the
   incarnation is *not* in the canonical evidence envelope; in fact it is
   embedded in the hashed `acquisition.source_record_id`. Correct the
   documentation (or the behaviour) so the participant is not told a
   non-secret-but-operator-supplied token does not enter portable evidence.
2. **Kit acquisition.** The kit guide describes a kit produced from a tagged
   release, but the release publishes a binary, not a kit, and the kit generator
   is not referenced by the guide, Makefile or CI. State the exact download and
   kit-assembly path.
3. **Signature / provenance.** The guide documents SHA-256 checksums only.
   Document how to verify release provenance (and do not imply a signature over
   the binaries that does not exist).
4. **Replay visibility.** The operator cannot tell from `capture status` whether
   a restart replayed durable observations. Either surface it or document that it
   is not observable offline.
5. **Removal completeness.** Removal instructions omit `~/.atb/identity-map.yaml`,
   which can contain raw API keys.

These are recorded as findings in the programme report; they are not fixed by
this protocol.

## Stop conditions

Stop and do not run a pilot if any hold: no genuine external participant; the
participant is coached; the workload cannot be bounded; the security/privacy
questionnaire (see the companion document) cannot be answered by the operator; or
an open **pilot-blocking** evidence-integrity/security defect exists (a defect
that would prevent safe or honest evaluation).
