---
name: atb-trust-boundary
description: Use when a change could alter what ATB claims to prove or when reasoning about evidence authority. Covers the ATB epistemic boundary (integrity not truth), the public/private product boundary (ATB vs Mortise/Tenon), and the Jev advisory boundary. Trigger on: verification claims, "trust"/"score" wording, completeness/truth/causality language, Mortise/Tenon features in ATB, Jev influencing ATB verification.
---

# ATB Trust Boundary

## Purpose

Keep ATB an evidence/integrity layer. ATB proves integrity, ordering, and
captured relationships. ATB does **not** prove truth, completeness, semantic
correctness, causality, safety, or absence of uncaptured activity.

## When to invoke

- Any change to verification output, viewer "Evidence status", or findings wording
- Any change that could be read as a trust/health/completeness score
- Any proposal to put authority, enforcement, or orchestration into ATB
- Any proposal to make ATB verification depend on Jev or any model

## Bounded procedure

1. Identify the exact claim the change makes (sentence, field, label, finding).
2. Classify it: PROVEN (hash-chain/order/signature), OBSERVED (captured
   events/findings), or INTERPRETED (practitioner priority).
3. Reject any INTERPRETED claim presented as PROVEN.
4. Check product boundary:
   - ATB = public, deterministic evidence/integrity.
   - Mortise = private, execution/control/governance authority.
   - Tenon = private, longitudinal/system continuity.
   - Jev = probabilistic typed decision primitive, advisory only.
5. Enforce this skill-level guardrail (not a canonical claim): JEV MAY ANALYSE
   ATB EVIDENCE; ATB MUST NOT DEPEND ON JEV TO ESTABLISH THAT EVIDENCE IS VALID.

## Stop conditions

- Stop if the only way to proceed is to make ATB claim truth/completeness.
- Stop if a Mortise/Tenon capability is being pulled into public ATB.
- Stop if a model/Jev becomes required for deterministic verification.

## Canonical references

- docs/concepts/trust-model.md
- docs/concepts/evidence-model.md
- docs/verification-meaning.md
- AGENTS.md (product identity + architectural invariants)

## Tests / checks

- `make test-golden` (canonical parity)
- `make demo-incident` (finding + tamper semantics)
- Reviewer must be from a different model family than the implementer.

## Prohibited interpretations

- "Verified" does not mean true, complete, safe, or causal.
- A finding is not an action, verdict, or policy decision.
- A coverage/assessment figure is not a completeness guarantee.
