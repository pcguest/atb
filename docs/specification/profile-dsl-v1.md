# Obligation Profile DSL — Specification v1

## Purpose

Obligation profiles define the evidence ATB expects to see for a declared AI workflow, then feed that evidence into profile pass/fail checks, Completeness Assurance Score (CAS) calculation, `verify.report.v1`, and residual-risk reporting. They support EU AI Act Article 12 logging review by making recorded lifecycle evidence explicit, and Article 17 quality-system review by making profile gaps and exclusions repeatable. ATB records and verifies evidence; it does not certify legal compliance.

## Profile File Format

The public obligation-profile DSL v1 format is YAML. A user profile file is parsed by `internal/profiles.ParseDSLProfile` and compiled into the internal `ProfileSchema` used by the built-in profiles and verifier.

The public DSL v1 input fields are:

```yaml
id: "org.example.support_triage"          # string, required, unique, must not collide with built-in profile IDs
version: 1                                # integer, optional, defaults to 1, must be >= 1
description: "Support triage workflow"    # string, optional, becomes workflow_class; defaults to the last dot segment of id
required_events:                          # []string, optional, each missing event is a critical failure
  - "ai.request.received"
  - "ai.model.invoked"
warning_events:                           # []string, optional, each missing event is a warning
  - "ai.response.sent"
cas_weights:                              # map[string]number, optional; present means CAS is enabled
  EC: 0.50                                # allowed keys: EC, FC, RC, TC, SC, XC, AC, GC
  FC: 0.25                                # provided values must sum to 1.0; omitted keys default to 0.0
  XC: 0.15
  AC: 0.10
```

Built-in profiles use a richer internal YAML schema stored under `internal/profiles/templates/`. The following real profile is `internal/profiles/templates/policy_decision.yaml`, annotated with field meaning:

```yaml
id: atb.profile.policy_decision
# string, required. Built-in profile identifier.

version: 1
# integer, required in internal templates. Must be >= 1.

workflow_class: policy_decision
# string, required. Human-readable workflow class in reports.

blind_spots:
  - "Limitation: does not verify policy rule set correctness. | Mitigation: record policy_version and policy_doc_hash with signatures (L3). | Residual: policy engine internal evaluation logic."
  - "Limitation: policy engine internal state is not attested. | Mitigation: source signatures on ai.policy.decision (L3). | Residual: engine state beyond recorded decision fields."
  - "Limitation: XC credit requires corroboration events. | Mitigation: atb corroborate (L4). | Residual: adapter not used."
# []string, optional in the struct. Included in verifier exclusions and notes.

weights:
  EC: 0.15
  FC: 0.10
  RC: 0.15
  TC: 0.10
  SC: 0.25
  XC: 0.10
  AC: 0.10
  GC: 0.05
# map[string]number, required by internal schema validation.
# Keys must be exactly EC, FC, RC, TC, SC, XC, AC, GC and sum to 1.0.

supports_cas: true
# boolean, optional, default false when absent. Enables CAS for this schema.

sc_mode: policy_decision
# string, optional. Selects source-commitment scoring mode used by the verifier.

required:
  - type: atb.bundle.manifest
    fields: []
    message: atb.bundle.manifest missing
    severity: critical
  - type: ai.request.received
    fields:
      - request_id
      - actor_id_hash
      - purpose_tag
    message: ai.request.received missing required fields
    severity: critical
  - type: ai.policy.decision
    fields:
      - policy_id
      - policy_version
      - decision
      - decision_reason_codes
      - subject_id_hash
      - action_id
    message: ai.policy.decision missing required fields
    severity: critical
# []EventRule, optional in the struct. Internal templates use it for critical obligations.

optional:
  - type: ai.action.precommit
    fields:
      - action_id
      - action_type
      - action_parameters_digest
    message: ai.action.precommit recommended to bind policy to a pending action
    severity: warning
    required_when:
      - when_type: ai.request.received
        at_or_after: true
        message: ai.action.precommit must occur at or after ai.request.received
# []EventRule, optional. Missing warning rules normally warn; required_when can promote them to critical failures.

relations:
  - name: policy_binds_action
    from: ai.policy.decision
    to: ai.action.precommit
    field: action_id
    message: "policy_binds_action: ai.policy.decision action_id does not match ai.action.precommit"
# []RelationRule, optional. Enforces cross-event field binding and optional predicates.
```

## Field Reference

Public DSL v1 fields:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | string | Yes | none | Unique profile identifier. Must not be empty and must not collide with a built-in profile ID. |
| `version` | integer | No | `1` | Profile schema version. Values below `1` are normalised to `1` by the DSL parser. |
| `description` | string | No | Last dot-separated segment of `id` | Human-readable workflow class used in reports. |
| `required_events` | array of string | No | empty array | Event types that must be present. Each missing type becomes a critical `missing_event` failure. Values must be non-empty and unique within the list. |
| `warning_events` | array of string | No | empty array | Event types recommended for review. Each missing type becomes a warning, not a critical failure. Values must be non-empty and unique within the list. |
| `cas_weights` | object of number | No | absent; CAS disabled | Enables CAS when present. Allowed keys are `EC`, `FC`, `RC`, `TC`, `SC`, `XC`, `AC`, `GC`; provided values must sum to `1.0` within `1e-9`; omitted keys default to `0.0`. |

Internal `ProfileSchema` fields used by built-in templates:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | string | Yes | none | Built-in or schema-backed profile identifier. |
| `version` | integer | Yes | none | Must be `>= 1`. |
| `workflow_class` | string | Yes | none | Human-readable workflow class in profile and verifier reports. |
| `blind_spots` | array of string | No | empty array | Limitation statements copied into exclusions and informational notes. |
| `weights` | object of number | Yes for internal schema validation | none | CAS weight vector. Keys must be exactly `EC`, `FC`, `RC`, `TC`, `SC`, `XC`, `AC`, `GC` and sum to `1.0`. |
| `supports_cas` | boolean | No | `false` | Marks whether the profile emits CAS. |
| `sc_mode` | string | No | empty string | Source-commitment scoring mode consumed by verifier source-binding logic. |
| `required` | array of `EventRule` | No | empty array | Critical event and field obligations. |
| `optional` | array of `EventRule` | No | empty array | Warning-level event and field obligations, with optional `required_when` conditions. |
| `relations` | array of `RelationRule` | No | empty array | Cross-event relation checks. |

`EventRule` fields:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | string | Yes | none | Event type matched against bundle records. Duplicate types are rejected within one section. |
| `fields` | array of string | No | empty array | Required payload fields on the first matching record. Missing fields generate findings. |
| `message` | string | No | generated fallback | Message used for missing-event or missing-field findings. |
| `severity` | string | Yes | none | Must be `critical` or `warning`. |
| `required_when` | array of rule | No | empty array | Supported only on optional rules. Each rule references another known event type and may require timestamp ordering. |

`required_when` fields:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `when_type` | string | Yes | none | Event type that triggers the conditional obligation. Must reference a known required or optional rule in the same profile. |
| `at_or_after` | boolean | No | `false` | When true, the target event must have an RFC 3339 timestamp at or after the condition event. |
| `message` | string | No | generated fallback | Finding message when the conditional event is absent. |

`RelationRule` fields:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | No | generated from `from`, `to`, and `field` | Stable relation identifier prefix. |
| `from` | string | No explicit validator | empty string | Source event type for relation evaluation. |
| `to` | string | No explicit validator | empty string | Target event type for relation evaluation. |
| `field` | string | No explicit validator | empty string | Payload field used to bind source and target records. |
| `message` | string | No | generated fallback | Finding message for relation violations. |
| `predicates` | array of `RelationPredicate` | No | empty array | Additional checks on linked records. |

`RelationPredicate` fields:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `side` | string | Yes when predicate exists | none | Must be `from` or `to`. |
| `field` | string | Yes when predicate exists | none | Payload field read from the selected side. |
| `equals` | string | Yes when predicate exists | none | Required string value, compared case-insensitively. |

## CAS Scoring

CAS is computed in `internal/verify.ComputeCAS`. Each profile supplies a sub-score map and a weight vector keyed by `EC`, `FC`, `RC`, `TC`, `SC`, `XC`, `AC`, and `GC`.

When the hash chain is invalid, CAS is forced to:

```text
overall = 0
effective_score = 0
grade = "Insufficient"
```

When the hash chain is valid:

```text
overall = sum(weights[key] * sub_scores[key])
effective_score = overall
grade = gradeFromScore(overall)
```

The grade bands are:

| Grade | Condition |
|-------|-----------|
| `High` | `score >= 0.85` |
| `Medium` | `score >= 0.60` and `< 0.85` |
| `Low` | `score >= 0.30` and `< 0.60` |
| `Insufficient` | `score < 0.30` |

When `EvaluateBundle` receives a `CorroborationPolicy`, it computes a capped bonus from verified evidence:

```text
bonus = min(anchor_bonus + signature_bonus + snapshot_bonus, max_bonus)
effective_score = min(1.0, overall + bonus)
grade = gradeFromScore(effective_score)
```

The default policy is:

| Component | Value |
|-----------|-------|
| Anchor verified | `0.05` |
| Bundle signature verified | `0.03` |
| Snapshot verified | `0.02` |
| Maximum bonus | `0.10` |

Sub-score meanings are:

| Code | Implemented source |
|------|--------------------|
| `EC` | Required event coverage: present required event types divided by total required event types. |
| `FC` | Required field completeness averaged across present required event records. |
| `RC` | Profile-specific relation consistency, such as matching `action_id`, `request_id`, or `job_id`. DSL profiles without internal schema rules receive `0.0`. |
| `TC` | Profile-specific timestamp ordering and execution-window checks. DSL profiles without internal schema rules receive `0.0`. |
| `SC` | Source commitment from `computeSC`, including manifest, request, signature, anchor, and profile `sc_mode` signals. DSL profiles without internal schema rules receive `0.0`. |
| `XC` | External corroboration and anchor-classification signal, raised by valid `atb.corroboration.external` events. |
| `AC` | Anchor coverage, `1.0` only when RFC 3161 anchoring is verified. |
| `GC` | Profile-specific gating completeness. `atb.profile.rag_answer` is fixed at `0.3`; generic DSL profiles receive `0.0`. |

The provability ladder is implemented as verifier gaps derived from integrity and sub-scores:

| Layer | Implemented trigger | Evidence required to close |
|-------|---------------------|----------------------------|
| `L1` | Invalid hash-chain integrity | `integrity.chain_valid` is true. |
| `L2` | `EC`, `FC`, `RC`, or `TC` below `0.70` | Required event types, required fields, relation links, and temporal checks reach the closure criteria in `provability_gaps.go`. |
| `L3` | `SC` below `0.70`, or policy-signature warnings | Policy/source signatures and source-binding evidence raise `SC` to the documented threshold. |
| `L4` | `XC` or `AC` below `0.70` | External corroboration or verified RFC 3161 anchor evidence is present. |
| `L5` | `GC` below `0.70`, or retrospective-capture provenance | Capture/gating path evidence is recorded live enough to close the gating or retrospective limitation. |

## Compatibility Rules

For public DSL v1 profile files:

- Adding a new optional field that older parsers ignore is non-breaking.
- Removing an existing field is breaking.
- Renaming an existing field is breaking.
- Changing a field's type is breaking.
- Changing whether a field is required or optional is breaking.
- Changing CAS weights is breaking for that profile's score contract.
- Changing the meaning of `required_events` or `warning_events` is breaking.
- Adding a new required event to an existing profile is breaking because it can turn a previously passing bundle into a failing bundle.
- Adding a new warning event is non-breaking for gate pass/fail, but score-impacting when CAS weights or sub-score computation change.
- Changing duplicate-event, empty-event, built-in-collision, or CAS-weight validation is breaking when it changes whether an existing profile file loads.

For internal built-in `ProfileSchema` templates:

- Changing `id`, `version`, `workflow_class`, `required`, `optional`, `relations`, `weights`, `supports_cas`, or `sc_mode` changes verifier behaviour and requires release-note review.
- Adding, removing, or changing `blind_spots` changes explanatory output but not the hash-chain contract.
- Event payload fields remain part of profile evaluation, not the canonical bundle hash schema.

## Version History

- v1 — formalised 29 May 2026.
