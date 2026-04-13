# Obligation profiles and CAS

ATB uses obligation profiles to evaluate whether a bundle contains the minimum evidence expected for a workflow. Hash-chain verification answers whether recorded evidence was changed. Profiles and CAS answer whether enough of the workflow was recorded to support later review.

## Profile summary

| Profile ID | Workflow class | CAS support |
|------------|----------------|-------------|
| `atb.profile.privileged_tool_action` | `privileged_tool_action` | Yes |
| `atb.profile.rag_answer` | `rag_answer` | Yes |
| `atb.profile.data_export` | `data_export` | Yes |
| `atb.profile.policy_decision` | `policy_decision` | Yes |
| `atb.profile.human_override` | `human_override` | Yes |
| `atb.profile.background_automation` | `background_automation` | Yes |

## Built-in profiles

### 1. Privileged Tool Action (`atb.profile.privileged_tool_action`)

Workflow class: `privileged_tool_action`

CAS support: Yes

Required events:

- `atb.bundle.manifest`
- `ai.request.received` — fields: `request_id`, `actor_id_hash`, `purpose_tag`
- `ai.action.precommit` — fields: `action_id`, `action_type`, `action_parameters_digest`, `target_resource_id`, `intended_effect`
- `ai.policy.decision` — fields: `policy_id`, `policy_version`, `decision`, `decision_reason_codes`, `subject_id_hash`, `action_id`
- `ai.action.executed` — fields: `action_id`, `execution_outcome`, `tool_receipt_digest`
- `ai.action.committed` — fields: `action_id`, `commit_outcome`, `sink_receipt_digest`

Optional evidence:

- `ai.human.approval` — warning; becomes required when `ai.action.executed` is present. Fields: `approval_id`, `approver_id_hash`, `approval_outcome`, `justification_digest`, `action_id`

Relation checks:

- `ai.action.committed` must bind to `ai.action.precommit` by `action_id`.
- `ai.policy.decision` must bind to `ai.action.precommit` by `action_id`.
- `ai.action.executed` must bind to an allowing `ai.policy.decision` for the same `action_id`.

CAS weight vector:

| Code | Weight |
|------|--------|
| EC | 0.20 |
| FC | 0.15 |
| RC | 0.20 |
| TC | 0.05 |
| SC | 0.10 |
| XC | 0.10 |
| AC | 0.10 |
| GC | 0.10 |

### 2. RAG Answer (`atb.profile.rag_answer`)

Workflow class: `rag_answer`

CAS support: Yes

Required events:

- `atb.bundle.manifest`
- `ai.request.received` — fields: `request_id`, `actor_id_hash`, `purpose_tag`
- `ai.model.invoked` — fields: `model_provider`, `model_id`, `model_parameters_digest`, `prompt_digest`
- `ai.model.output` — fields: `output_digest`, `output_format`

Optional evidence:

- `ai.policy.decision` — warning. Fields: `policy_id`, `policy_version`, `decision`, `decision_reason_codes`
- `ai.retrieval.executed` — warning. Fields: `retrieval_query_hash`, `retrieval_corpus_id`, `retrieval_corpus_version`, `top_k`, `result_set_digest`
- `ai.response.sent` — warning. Fields: `request_id`, `output_digest`

Relation checks:

- `ai.response.sent` must bind to `ai.request.received` by `request_id` when both are present.

CAS weight vector:

| Code | Weight |
|------|--------|
| EC | 0.25 |
| FC | 0.15 |
| RC | 0.15 |
| TC | 0.10 |
| SC | 0.15 |
| XC | 0.05 |
| AC | 0.05 |
| GC | 0.10 |

### 3. Data Export (`atb.profile.data_export`)

Workflow class: `data_export`

CAS support: Yes

Required events:

- `atb.bundle.manifest`
- `ai.request.received` — fields: `request_id`, `actor_id_hash`, `purpose_tag`
- `ai.policy.decision` — fields: `policy_id`, `policy_version`, `decision`, `decision_reason_codes`, `subject_id_hash`
- `ai.action.precommit` — fields: `action_id`, `action_type`, `action_parameters_digest`, `target_resource_id`, `intended_effect`
- `ai.action.executed` — fields: `action_id`, `execution_outcome`, `tool_receipt_digest`
- `ai.action.committed` — fields: `action_id`, `commit_outcome`, `sink_receipt_digest`

Optional evidence:

- `ai.human.approval` — warning; becomes required when `ai.action.executed` is present. Fields: `approval_id`, `approver_id_hash`, `approval_outcome`, `justification_digest`, `action_id`

Relation checks:

- `ai.action.committed` must bind to `ai.action.precommit` by `action_id`.
- `ai.policy.decision` must bind to `ai.action.precommit` by `action_id`.
- `ai.action.executed` must bind to an allowing `ai.policy.decision` for the same `action_id`.

CAS weight vector:

| Code | Weight |
|------|--------|
| EC | 0.15 |
| FC | 0.10 |
| RC | 0.15 |
| TC | 0.10 |
| SC | 0.10 |
| XC | 0.20 |
| AC | 0.10 |
| GC | 0.10 |

### 4. Policy Decision (`atb.profile.policy_decision`)

Workflow class: `policy_decision`

CAS support: Yes

Required events:

- `atb.bundle.manifest`
- `ai.request.received` — fields: `request_id`, `actor_id_hash`, `purpose_tag`
- `ai.policy.decision` — fields: `policy_id`, `policy_version`, `decision`, `decision_reason_codes`, `subject_id_hash`, `action_id`

Optional evidence:

- `ai.action.precommit` — warning. Fields: `action_id`, `action_type`, `action_parameters_digest`

Relation checks:

- `ai.policy.decision` must bind to `ai.action.precommit` by `action_id` when both are present.

CAS weight vector:

| Code | Weight |
|------|--------|
| EC | 0.15 |
| FC | 0.10 |
| RC | 0.15 |
| TC | 0.10 |
| SC | 0.25 |
| XC | 0.10 |
| AC | 0.10 |
| GC | 0.05 |

### 5. Human Override (`atb.profile.human_override`)

Workflow class: `human_override`

CAS support: Yes

Required events:

- `atb.bundle.manifest`
- `ai.request.received` — fields: `request_id`, `actor_id_hash`, `purpose_tag`
- `ai.human.approval` — fields: `approval_id`, `approver_id_hash`, `approval_outcome`, `justification_digest`, `action_id`
- `ai.action.precommit` — fields: `action_id`, `action_type`, `action_parameters_digest`, `target_resource_id`, `intended_effect`
- `ai.action.executed` — fields: `action_id`, `execution_outcome`, `tool_receipt_digest`

Optional evidence:

- `ai.action.committed` — warning. Fields: `action_id`, `commit_outcome`, `sink_receipt_digest`

Relation checks:

- `ai.human.approval` must bind to `ai.action.precommit` by `action_id`.
- `ai.action.executed` must bind to an approved `ai.human.approval` for the same `action_id`.

CAS weight vector:

| Code | Weight |
|------|--------|
| EC | 0.15 |
| FC | 0.10 |
| RC | 0.20 |
| TC | 0.20 |
| SC | 0.10 |
| XC | 0.10 |
| AC | 0.05 |
| GC | 0.10 |

### 6. Background Automation (`atb.profile.background_automation`)

Workflow class: `background_automation`

CAS support: Yes

Required events:

- `atb.bundle.manifest`
- `ai.job.scheduled` — fields: `job_id`, `job_type`, `trigger_source`, `scheduled_by_id_hash`
- `ai.job.started` — fields: `job_id`, `worker_id_hash`, `started_at`
- `ai.job.completed` — fields: `job_id`, `outcome`, `completion_reason`

Optional evidence:

- `ai.job.step` — warning. Fields: `job_id`, `step_index`, `step_type`, `step_outcome`

Relation checks:

- `ai.job.started` must bind to `ai.job.scheduled` by `job_id`.
- `ai.job.completed` must bind to `ai.job.started` by `job_id`.

CAS weight vector:

| Code | Weight |
|------|--------|
| EC | 0.15 |
| FC | 0.10 |
| RC | 0.15 |
| TC | 0.10 |
| SC | 0.05 |
| XC | 0.20 |
| AC | 0.05 |
| GC | 0.20 |

## Completeness assurance score

CAS is a weighted score from 0.0 to 1.0 that measures how well the recorded evidence matches the selected profile. All six built-in profiles support CAS.

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
