# Obligation Profiles & CAS

ATB uses **Obligation Profiles** and the **Completeness Assurance Score (CAS)** to evaluate the quality and completeness of recorded evidence. While the hash chain ensures **integrity** (that the record was not changed), profiles and CAS measure **completeness** (that the necessary events were recorded).

## Obligation Profiles

A profile defines which events and fields are required for a specific workflow. ATB includes six built-in profiles.

### 1. Privileged Tool Action (`atb.profile.privileged_tool_action`)
**Workflow:** High-impact actions such as shell command execution, database mutations, or cloud infrastructure changes.
**Risk:** Unauthorized or unrecorded high-impact changes that bypass security gates.

*   **Required Events:** `atb.bundle.manifest`, `ai.request.received`, `ai.action.precommit`, `ai.policy.decision`, `ai.action.executed`, `ai.action.committed`.
*   **Key Rules:**
    *   `ai.human.approval` is required when `ai.action.executed` is present and must occur at or after the execution timestamp.
    *   `ai.action.executed` must occur after `ai.policy.decision` for the same `action_id`.
*   **PASS:** All required events exist, fields are populated, and the sequence (precommit → policy → execute → commit) is causally ordered.
*   **FAIL/WARN:** Missing execution records, failed policy checks for executed actions, or missing human approval for high-impact tools.

### 2. RAG Answer (`atb.profile.rag_answer`)
**Workflow:** AI-generated responses that rely on retrieved context (Retrieval-Augmented Generation).
**Risk:** Hallucinations or lack of provenance for model-generated claims.

*   **Required Events:** `atb.bundle.manifest`, `ai.request.received`, `ai.model.invoked`, `ai.model.output`.
*   **Optional (Recommended):** `ai.retrieval.executed`, `ai.policy.decision`, `ai.response.sent`.
*   **PASS:** The model invocation and output are recorded, binding the prompt to the result.
*   **WARN:** Missing `ai.retrieval.executed` reduces the provenance signal for where the model found its context.

### 3. Data Export (`atb.profile.data_export`)
**Workflow:** Exporting or exfiltrating data records from the system.
**Risk:** Unmonitored or unauthorized data egress.

*   **Required Events:** Same as Privileged Tool Action.
*   **Key Rules:**
    *   `ai.human.approval` is required when `ai.action.executed` is present.
*   **PASS:** Full audit trail from request to commit, including the target resource and intended effect.

### 4. Policy Decision (`atb.profile.policy_decision`)
**Workflow:** Standalone policy evaluations or guardrail checks.
**Risk:** Unrecorded or opaque policy outcomes driving system behaviour.

*   **Required Events:** `atb.bundle.manifest`, `ai.request.received`, `ai.policy.decision`.
*   **PASS:** A clear decision (allow/deny) and reason codes are recorded for the subject.

### 5. Human Override (`atb.profile.human_override`)
**Workflow:** Manual intervention, override, or approval of an AI-driven action.
**Risk:** Unjustified or unrecorded bypass of automated controls.

*   **Required Events:** `atb.bundle.manifest`, `ai.request.received`, `ai.human.approval`, `ai.action.precommit`, `ai.action.executed`.
*   **PASS:** A signed approval record with a justification digest exists and is linked to the executed action.

### 6. Background Automation (`atb.profile.background_automation`)
**Workflow:** Scheduled tasks or autonomous agent execution without a direct human trigger.
**Risk:** "Ghost" executions where actions happen without a recorded request or trigger.

*   **Required Events:** Same as Privileged Tool Action.
*   **PASS:** The automation trace includes the full authorization and execution lifecycle.

---

## Completeness Assurance Score (CAS)

The CAS is a weighted score from 0.0 to 1.0 that measures how well the recorded evidence matches the profile's expectations.

### CAS Grades
*   **High (A):** ≥ 0.85 — Strong evidence coverage across all dimensions.
*   **Medium (B):** ≥ 0.60 — Sufficient evidence for review, with minor gaps.
*   **Low (C):** ≥ 0.30 — Significant evidence gaps; residual risk is high.
*   **Insufficient (F):** < 0.30 — Minimal or no reliable evidence for the workflow.

### Sub-Scores
| Code | Name | What it measures |
| :--- | :--- | :--- |
| **EC** | Event Coverage | Are all required event types present in the bundle? |
| **FC** | Field Completeness | Do the recorded events contain all required data fields? |
| **RC** | Relation Consistency | Are events correctly linked (e.g., via `action_id`)? |
| **TC** | Temporal Consistency | Are events recorded in a causally valid chronological order? |
| **SC** | Source Commitment | Are events signed by the originating source or policy engine? |
| **XC** | External Corroboration | Is there an RFC 3161 TSA anchor for the bundle? |
| **AC** | Anchor Coverage | Does the TSA anchor cover the most recent bundle state? |
| **GC** | Gating Completeness | Is the precommit-to-commit lifecycle fully represented? |

### Integrity vs. Completeness
*   **Integrity (The Hash Chain):** Proves that *what was recorded* has not been changed. Verification fails if a single byte is mutated.
*   **Completeness (CAS):** Measures *how much of the workflow* was recorded. A "passing" integrity check does not mean the audit trail is complete; it only means the (possibly incomplete) trail is authentic.

---

## Compliance Mapping

Obligation profiles help teams meet specific regulatory and framework requirements for AI logging and auditability.

| Requirement | Description | Relevant ATB Profile(s) |
| :--- | :--- | :--- |
| **EU AI Act Art. 12** | Logging of events throughout the AI system's life cycle. | All (EC/TC scores) |
| **ISO/IEC 42001 A.6.2.8** | Requirements for recording system activity and lifecycle events. | `background_automation`, `privileged_tool_action` |
| **SOC 2 CC6.1 / CC7.2** | Audit trail for logical access and system operations. | `policy_decision`, `human_override` |
| **NIST AI RMF** | Measurement and tracking of system behaviour and transparency. | `rag_answer`, `policy_decision` |
