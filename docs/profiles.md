# Obligation profiles and CAS

ATB uses obligation profiles to evaluate whether a bundle contains the minimum evidence expected for a workflow. Hash-chain verification answers whether recorded evidence was changed. Profiles and CAS answer whether enough of the workflow was recorded to support later review.

## Profile summary

| Profile ID | Workflow class | CAS support |
|------------|----------------|-------------|
| `atb.profile.privileged_tool_action` | `privileged_tool_action` | Yes |
| `atb.profile.rag_answer` | `rag_answer` | Yes |
| `atb.profile.data_export` | `data_export` | No |
| `atb.profile.policy_decision` | `policy_decision` | No |
| `atb.profile.human_override` | `human_override` | No |
| `atb.profile.background_automation` | `background_automation` | No |

## Built-in profiles

### 1. Privileged Tool Action (`atb.profile.privileged_tool_action`)

Workflow class: `privileged_tool_action`

CAS support: Yes

Required events:

- `atb.bundle.manifest`
- `ai.request.received`
- `ai.action.precommit`
- `ai.policy.decision`
- `ai.action.executed`
- `ai.action.committed`

Optional evidence:

- `ai.human.approval` is warning-level evidence and becomes required when `ai.action.executed` is present.

Relation checks:

- `ai.action.committed` must bind to `ai.action.precommit` by `action_id`.
- `ai.policy.decision` must bind to `ai.action.precommit` by `action_id`.
- `ai.action.executed` must bind to an allowing `ai.policy.decision` for the same `action_id`.

### 2. RAG Answer (`atb.profile.rag_answer`)

Workflow class: `rag_answer`

CAS support: Yes

Required events:

- `atb.bundle.manifest`
- `ai.request.received`
- `ai.model.invoked`
- `ai.model.output`

Optional evidence:

- `ai.policy.decision`
- `ai.retrieval.executed`
- `ai.response.sent`

Relation checks:

- `ai.response.sent` must bind to `ai.request.received` by `request_id` when both are present.

### 3. Data Export (`atb.profile.data_export`)

Workflow class: `data_export`

CAS support: No

Required events:

- `atb.bundle.manifest`
- `ai.request.received`
- `ai.policy.decision`
- `ai.action.precommit`
- `ai.action.executed`
- `ai.action.committed`

Optional evidence:

- `ai.human.approval` is warning-level evidence and becomes required when `ai.action.executed` is present.

Relation checks:

- `ai.action.committed` must bind to `ai.action.precommit` by `action_id`.
- `ai.policy.decision` must bind to `ai.action.precommit` by `action_id`.
- `ai.action.executed` must bind to an allowing `ai.policy.decision` for the same `action_id`.

### 4. Policy Decision (`atb.profile.policy_decision`)

Workflow class: `policy_decision`

CAS support: No

Required events:

- `atb.bundle.manifest`
- `ai.request.received`
- `ai.policy.decision`

Optional evidence:

- `ai.action.precommit`

Relation checks:

- `ai.policy.decision` must bind to `ai.action.precommit` by `action_id` when both are present.

### 5. Human Override (`atb.profile.human_override`)

Workflow class: `human_override`

CAS support: No

Required events:

- `atb.bundle.manifest`
- `ai.request.received`
- `ai.human.approval`
- `ai.action.precommit`
- `ai.action.executed`

Optional evidence:

- `ai.action.committed`

Relation checks:

- `ai.human.approval` must bind to `ai.action.precommit` by `action_id`.
- `ai.action.executed` must bind to an approved `ai.human.approval` for the same `action_id`.

### 6. Background Automation (`atb.profile.background_automation`)

Workflow class: `background_automation`

CAS support: No

Required events:

- `atb.bundle.manifest`
- `ai.job.scheduled`
- `ai.job.started`
- `ai.job.completed`

Optional evidence:

- `ai.job.step`

Relation checks:

- `ai.job.started` must bind to `ai.job.scheduled` by `job_id`.
- `ai.job.completed` must bind to `ai.job.started` by `job_id`.

## Completeness assurance score

CAS is a weighted score from 0.0 to 1.0 that measures how well the recorded evidence matches the selected profile. It is currently emitted only for `atb.profile.privileged_tool_action` and `atb.profile.rag_answer`.

### CAS grades

- High, `>= 0.85`: strong evidence coverage across all dimensions.
- Medium, `>= 0.60`: sufficient evidence for review, with minor gaps.
- Low, `>= 0.30`: significant evidence gaps; residual risk is high.
- Insufficient, `< 0.30`: minimal or no reliable evidence for the workflow.

### Sub-scores

| Code | Name | What it measures |
|------|------|------------------|
| `EC` | Event Coverage | Are all required event types present in the bundle? |
| `FC` | Field Completeness | Do the recorded events contain all required data fields? |
| `RC` | Relation Consistency | Are events correctly linked, for example by `action_id` or `request_id`? |
| `TC` | Temporal Consistency | Are events recorded in a causally valid chronological order? |
| `SC` | Source Commitment | Are events bound to their originating source and governance context? |
| `XC` | External Corroboration | Is corroborating evidence present beyond the bundle itself? |
| `AC` | Anchor Coverage | Does RFC 3161 anchoring cover the relevant bundle state? |
| `GC` | Gating Completeness | Is the control-plane path fully represented from intent to committed effect? |

### Integrity vs. completeness

- Integrity, the hash chain: proves that what was recorded has not been changed.
- Completeness, CAS: measures how much of the workflow was recorded.

## Compliance mapping

| Requirement | Description | Relevant ATB profile(s) |
|-------------|-------------|-------------------------|
| EU AI Act Art. 12 | Logging of events throughout the AI system life cycle. | All profiles, especially where `EC` and `TC` are relevant |
| ISO/IEC 42001 A.6.2.8 | Requirements for recording system activity and lifecycle events. | `atb.profile.background_automation`, `atb.profile.privileged_tool_action` |
| SOC 2 CC6.1 / CC7.2 | Audit trail for logical access and system operations. | `atb.profile.policy_decision`, `atb.profile.human_override` |
| NIST AI RMF | Measurement and tracking of system behaviour and transparency. | `atb.profile.rag_answer`, `atb.profile.policy_decision` |
