---
name: jev-decision-review
description: Use when semantic ambiguity survives deterministic gates and independent review, and a typed probabilistic observation would help. Covers the local Jev/System One decision plane (Score/Noul/Choice), execution statuses, and the escalation policy. Trigger on: ambiguous root cause, reviewer/implementer disagreement, claim leakage, scope drift, "should we escalate", running the Jev decision plane.
---

# Jev Decision Review

## Purpose

Use Jev (TypeSafe System One) as an advisory typed decision primitive over
bounded development packets. Jev observes semantic ambiguity; it never overrides
deterministic evidence, never merges, never bypasses tests.

## When to invoke

- Deterministic gates pass but behaviour may be wrong
- Multiple plausible hypotheses remain
- Reviewer and implementer disagree
- Claim leakage / scope drift / ambiguous terminology
- Deciding whether stronger review or a human is required

Do NOT invoke for deterministic failures, clear defects, typos, or simple
preferences.

## Bounded procedure

1. Build a DecisionPacket: task_id, change_scope, evidence_refs,
   deterministic_gate_summary, diff_summary, architecture_context, typed questions.
2. Ask a small library of typed questions (Noul / Choice / Score). Never ask
   vague questions like "is this good?".
3. Use the Jev decision plane via an optional external tool (not maintained in
   this repository). Record a DecisionObservation: execution_status, model
   identity, question schema, typed answers, probabilities, confidence,
   latency/usage, decision = ADVISORY_ONLY.
4. execution_status is explicit: SUCCESS / UNAVAILABLE / ERROR / NOT_RUN /
   NOT_RUN_NO_DIFF. SUCCESS means a parseable typed observation was returned;
   it does NOT mean engineering approval.
5. Threshold/policy belongs to the caller, not to Jev. No universal threshold.

## Stop conditions

- Stop if Jev output would be treated as authority, approval, or a gate.
- Stop if a Jev failure would invalidate deterministic engineering evidence.
- Stop if the question is deterministic (use the tests instead).

## Canonical references

- docs/concepts/trust-model.md (why Jev is not ATB trust)

## Tests / checks

- CREDENTIAL_AVAILABLE / CREDENTIAL_UNAVAILABLE only; never log key/prefix/length
- Record disagreement honestly; remove question families that add no value

## Prohibited interpretations

- JEV SUCCESS ≠ PASS ≠ APPROVED ≠ VERIFIED.
- Jev is ADVISORY_ONLY and is not part of ATB verification or canonicalisation.
- Do not tune a test until Jev agrees with us.
