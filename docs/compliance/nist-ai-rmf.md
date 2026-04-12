<!-- Practitioner reference only. Not a formal NIST AI RMF assessment.
Sub-category references are to NIST AI RMF 1.0 (January 2023).
Review against the current RMF version before citing in submissions. -->

# NIST AI RMF and ATB

## Purpose

NIST AI RMF organises AI risk management across four functions:
GOVERN, MAP, MEASURE, and MANAGE. This document maps ATB verification
output to the functions and sub-categories most relevant to
traceability, auditability, and incident response.

It does not constitute a formal RMF assessment or gap analysis. It is a
practitioner reference for teams that need to explain ATB's role in an
RMF-aligned programme.

## CAS sub-score to RMF function mapping

| CAS Sub-score | Full name | Primary RMF function | Sub-category | What a low score signals to an auditor |
| --- | --- | --- | --- | --- |
| `EC` | Event Coverage | `MEASURE` | `MES-2.5` | Gaps in event capture suggest incomplete logging coverage. |
| `FC` | Field Completeness | `MEASURE` | `MES-2.5` | Missing required fields prevent reconstruction of decision context. |
| `RC` | Relational Consistency | `MEASURE` | `MES-2.6` | Unbound events suggest semantic laundering risk. |
| `TC` | Temporal Consistency | `MANAGE` | `MAN-2.2` | Ordering violations may indicate post-hoc log construction. |
| `SC` | Source Commitment | `GOVERN` | `GOV-1.7` | Missing signatures mean component attribution cannot be verified. |
| `XC` | External Corroboration | `MANAGE` | `MAN-4.1` | Low reconciliation signals possible bypass of recorded controls. |
| `AC` | Anchor Coverage | `MANAGE` | `MAN-2.2` | Absence of TSA anchor weakens existence-at-time evidence for disputes. |
| `GC` | Gating Completeness | `GOVERN` | `GOV-2.2` | Missing precommit chain means high-impact actions may have executed without recorded authorisation. |

## Profile to RMF mapping

| Profile | RMF functions supported | RMF gaps / out of scope |
| --- | --- | --- |
| `atb.profile.rag_answer` | `MEASURE` (output traceability), `MAP` (model and retrieval inventory) | Does not cover `GOVERN` functions for the model provider or `MAP` functions for data provenance upstream of retrieval. |
| `atb.profile.privileged_tool_action` | `GOVERN` (authorisation controls), `MANAGE` (incident response evidence), `MEASURE` (action traceability) | Bypass paths outside the ATB gate are not covered. |
| `atb.profile.data_export` | `GOVERN` (export authorisation), `MANAGE` (data movement evidence) | Downstream recipient handling and retention verification are out of scope. |
| `atb.profile.policy_decision` | `GOVERN` (policy version traceability), `MEASURE` (decision auditability) | Correctness of policy text itself and identity attribute truth are not verified. |
| `atb.profile.human_override` | `GOVERN` (human oversight evidence), `MANAGE` (override audit trail) | Physical identity verification and approver comprehension are not attested. |
| `atb.profile.background_automation` | `MANAGE` (job execution traceability), `MEASURE` (output integrity) | External side effects outside the worker and SaaS actions are not captured. |

## What ATB does not cover

ATB does not provide `MAP` function coverage for AI system inventory or
risk classification. It does not provide `GOVERN` function coverage for
organisational policy governance or human oversight programme design. It
provides evidence that can support those functions when an operator has
already established the governance structure and instrumented their
workflows. ATB is evidence infrastructure, not a governance programme.
