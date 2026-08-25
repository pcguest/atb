# Obligation profiles and CAS

ATB uses obligation profiles to evaluate whether a bundle contains the minimum evidence expected for a workflow. Hash-chain verification answers whether recorded evidence was changed. Profiles and CAS answer whether enough of the workflow was recorded to support later review.

## CAS support matrix

| Profile ID | CAS supported | Notes |
|------------|---------------|-------|
| `atb.profile.privileged_tool_action` | Yes | Local completeness score over request, policy, execution, and commit evidence. Overall CAS falls to `0` when chain integrity fails. |
| `atb.profile.rag_answer` | Yes | Local completeness score over request, model, retrieval, and response evidence. Overall CAS falls to `0` when chain integrity fails. |
| `atb.profile.data_export` | Yes | Local completeness score over export authorisation and execution evidence. Overall CAS falls to `0` when chain integrity fails. |
| `atb.profile.policy_decision` | Yes | Local completeness score over request and policy decision evidence within the profile trust boundary. Overall CAS falls to `0` when chain integrity fails. |
| `atb.profile.human_override` | Yes | Local completeness score over approval, precommit, and execution evidence for override workflows. Overall CAS falls to `0` when chain integrity fails. |
| `atb.profile.background_automation` | Yes | Local completeness score over job scheduling, start, and completion evidence. Overall CAS falls to `0` when chain integrity fails. |

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

Blind spots / out of scope:

- Operator bypass of the ACP gate is not detectable from bundle events alone.
- Tool provider internal processing is not attested unless tool receipts are cryptographically verifiable.
- XC credit requires at least one valid `atb.corroboration.external` event.

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
- `ai.response.sent` — warning; becomes required (critical via `required_when`) when `ai.model.invoked` is present and must occur at or after it. Fields: `request_id`, `output_digest`

> **MCP PageIndex note:** The MCP bridge RAG tools (`rag_index_record`, `rag_retrieval_record`) emit `atb.event.rag_index` and `atb.event.rag_retrieval`, not `ai.retrieval.executed`. A bundle produced by those tools alone will not satisfy the retrieval evidence check in this profile. Emit `ai.retrieval.executed` from the surrounding workflow if you want full profile coverage.

Relation checks:

- `ai.response.sent` must bind to `ai.request.received` by `request_id` when both are present.

Blind spots / out of scope:

- Retrieval completeness beyond recorded corpus/version.
- Model provider internal execution fidelity.
- XC credit requires corroboration events.

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

> **GC note:** The GC sub-score for `atb.profile.rag_answer` is fixed at **0.3** regardless of bundle content. RAG workflows lack a full pre-commit gating chain, so GC cannot be computed from bundle events and is instead set to a static partial credit. This is intentional; do not rely on GC as a completeness signal for this profile.

### 3. Data Export (`atb.profile.data_export`)

Workflow class: `data_export`

CAS support: Yes

Required events:

- `atb.bundle.manifest`
- `ai.request.received` — fields: `request_id`, `actor_id_hash`, `purpose_tag`
- `ai.policy.decision` — fields: `policy_id`, `policy_version`, `decision`, `decision_reason_codes`, `subject_id_hash`, `action_id`
- `data.export.precommit` — fields: `action_id`, `action_type`, `action_parameters_digest`, `target_resource_id`, `intended_effect`
- `data.export.executed` — fields: `action_id`, `execution_outcome`, `tool_receipt_digest`

Optional evidence:

- `ai.human.approval` — warning; becomes required when `data.export.executed` is present. Fields: `approval_id`, `approver_id_hash`, `approval_outcome`, `justification_digest`, `action_id`

Relation checks:

- `ai.policy.decision` must bind to `data.export.precommit` by `action_id`.
- `data.export.executed` must bind to an allowing `ai.policy.decision` for the same `action_id`.

Blind spots / out of scope:

- Downstream recipient handling after export is not attested.
- Classification label correctness is not verified.
- XC credit requires corroboration events.

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

- `ai.action.precommit` — listed as optional but **effectively required** because `ai.request.received` is always present and triggers `required_when` (critical failure when absent). Must occur at or after the request. Fields: `action_id`, `action_type`, `action_parameters_digest`

Relation checks:

- `ai.policy.decision` must bind to `ai.action.precommit` by `action_id` when both are present.

Blind spots / out of scope:

- Does not verify policy rule set correctness.
- Policy engine internal state is not attested.
- XC credit requires corroboration events.

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
- `ai.action.executed` must bind to an approved `ai.human.approval` for the same `action_id` (ID binding enforced; `approval_outcome=approved` is checked in CAS RC/GC, not as a hard obligation failure).

Blind spots / out of scope:

- Plain approver IDs remain caller-asserted. Optional `identity_evidence` can
  bind the event to a digest of an external IdP assertion, but ATB does not
  validate the assertion or operate the identity system.
- Justification quality is not assessed.
- XC credit requires corroboration events.

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

Blind spots / out of scope:

- Scheduler integrity is not attested.
- Real-world job effect vs recorded outcome.
- Policy authorisation not required by this profile.
- XC credit requires corroboration events.

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

## Custom profiles (DSL v1)

ATB supports user-defined obligation profiles via a minimal YAML DSL. A DSL profile compiles to the same internal structures used by built-in profiles and goes through the same verification and CAS machinery.

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Unique profile identifier. Must not collide with a built-in profile ID. |
| `description` | No | Human-readable label; used as `workflow_class` in reports. Defaults to the last dot-segment of `id`. |
| `version` | No | Integer >= 1; defaults to 1. |
| `required_events` | No | List of event type strings. Each absent event type is a critical failure. |
| `warning_events` | No | List of event type strings. Each absent event type is a warning, not a critical failure. |
| `cas_weights` | No | Map of sub-score keys (`EC`, `FC`, `RC`, `TC`, `SC`, `XC`, `AC`, `GC`) to weights summing to 1.0. Omitting a key sets it to 0.0. Omitting the field entirely disables CAS for this profile. |

### Example

```yaml
id: "org.example.custom_support"
description: "Custom support escalation workflow"
required_events:
  - "ai.request.received"
  - "ai.action.precommit"
  - "ai.action.executed"
warning_events:
  - "ai.action.committed"
cas_weights:
  EC: 0.50
  FC: 0.25
  XC: 0.15
  AC: 0.10
```

See [`docs/profiles/examples/custom-support.yaml`](profiles/examples/custom-support.yaml) for a copy-ready file.

### Loading a custom profile

```
atb verify --bundle bundle.atb --profile ./profiles/custom-support.yaml
```

### CAS for custom profiles

When `cas_weights` is defined, CAS is computed using the same pipeline as built-in profiles. Sub-scores EC and FC are derived from the schema's required event rules. XC and AC are derived from the anchor result. RC, TC, SC, and GC cannot be computed generically and are fixed at 0.0 for custom profiles.

When `cas_weights` is omitted, no CAS score is produced.

### Non-goals for DSL v1

The following capabilities are out of scope for DSL v1:

- Cross-bundle correlation or arbitrary predicates.
- Per-event required-field checks (no `fields` list on event rules).
- Temporal ordering constraints beyond the existing event model.
- `required_when` conditional rules.
- Relation rules (cross-event ID binding checks).
- `sc_mode` or blind-spot declarations.

The complete YAML contract is documented in the
[obligation profile DSL specification](../specification/profile-dsl-v1.md).

## Completeness assurance score

CAS is a weighted score from 0.0 to 1.0 that measures how well the recorded evidence matches the selected profile. All six built-in profiles support CAS.

CAS is a local scoring model over the events recorded in the bundle and evaluated inside the selected profile's trust boundary. It answers how much of the expected evidence ATB can see for that workflow, not whether the surrounding system was instrumented completely.

CAS is not an external audit opinion, not proof of overall system compliance, and not proof that every material event in the real workflow was captured. It is a bounded completeness signal over recorded evidence. If chain verification fails, the overall CAS is forced to `0`.

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
- Profile-scoped evidence coverage, CAS: estimates how much of the selected
  profile's expected evidence is present in the recorded bundle.

## Compliance mapping

| Requirement | Description | Relevant ATB profile(s) |
|-------------|-------------|-------------------------|
| EU AI Act Art. 12 | Logging of events throughout the AI system life cycle. | All profiles, especially where `EC` and `TC` are relevant |
| ISO/IEC 42001 A.6.2.8 | Requirements for recording system activity and lifecycle events. | `atb.profile.background_automation`, `atb.profile.privileged_tool_action` |
| SOC 2 CC6.1 / CC7.2 | Audit trail for logical access and system operations. | `atb.profile.policy_decision`, `atb.profile.human_override` |
| NIST AI RMF | Measurement and tracking of system behaviour and transparency. | `atb.profile.rag_answer`, `atb.profile.policy_decision` |
