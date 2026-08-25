# CAS guide

CAS (Completeness Assurance Score) measures how much of the expected evidence ATB can see inside the selected obligation profile. It is a local scoring model over recorded bundle events — not a compliance decision, not an external audit opinion, and not proof that every material event in the real workflow was captured.

## Purpose

Use CAS to answer: *given this profile's trust boundary, how much of its
expected evidence is present in the recorded bundle?*

- **Integrity** (hash chain) proves that what was recorded has not been changed.
- **CAS** estimates profile-scoped evidence coverage within the recorded
  bundle.

A bundle can pass integrity checks and still return a low CAS. A bundle can fail profile obligations and still return a non-zero CAS (CAS is diagnostic in that case; it does not overturn FAIL).

If chain verification fails, overall CAS is forced to `0` and grade is `Insufficient`.

## Grade bands

| Grade | Threshold | Meaning |
|-------|-----------|---------|
| `High` | `>= 0.85` | Strong evidence coverage across weighted dimensions. |
| `Medium` | `>= 0.60` | Sufficient for review; minor gaps remain visible. |
| `Low` | `>= 0.30` | Significant gaps; residual risk is elevated. |
| `Insufficient` | `< 0.30` | Minimal or unreliable evidence for the workflow. |

Start with `critical_failures` when the score is low. That array names the blocking gaps (missing events, fields, relations, or timing checks). Fix those at the workflow call site, record a new bundle, and run `atb verify` again.

When `critical_failures` is empty but the score is still weak, inspect `sub_scores` and `required_warnings` for optional evidence such as retrieval records, anchoring, source signatures, or external corroboration.

## Sub-scores

Overall CAS is a weighted sum of eight sub-scores (each 0.0–1.0). Weights vary by profile; see [`profiles.md`](profiles.md) for per-profile weight vectors.

| Code | Full name | What it measures | Signals | Auditor interpretation |
|------|-----------|------------------|---------|------------------------|
| `EC` | Event Coverage | Required event types present | Count of required types found ÷ total required | Missing critical events indicate instrumentation gaps at call sites. |
| `FC` | Field Completeness | Required fields populated on present events | Per-record field coverage averaged across required events | Partial payloads reduce traceability even when event types exist. |
| `RC` | Relation Consistency | Cross-event ID binding | `action_id`, `request_id`, `job_id` links between related events | Broken links suggest mismatched or replayed records. |
| `TC` | Temporal Consistency | Causal event ordering | Timestamp order; execution window > 10m reduces score to 0.7 | Out-of-order or stale timestamps weaken causal narrative. |
| `SC` | Source Commitment | Governance binding of recorded events | RFC 3161 anchor (+0.40), manifest at seq 0 (+0.25), `ai.request.received` (+0.20), bundle signature (+0.10), plus profile `sc_mode` bonus (+0.15) | Higher SC indicates stronger binding to policy/retrieval context. |
| `XC` | External Corroboration | Evidence beyond the bundle | Anchor tier (absent 0, bad 0.1, digest-only 0.5, verified 1.0); raised by valid `atb.corroboration.external` events | Low XC means no independent corroboration was recorded, not necessarily a recording failure. |
| `AC` | Anchor Coverage | RFC 3161 TSA verification | 1.0 only when anchor is cryptographically verified; else 0.0 | Binary signal: verified timestamp token present or not. |
| `GC` | Gating Completeness | Control-plane path from intent to effect | Precommit → execute → commit chain (ACP profiles); **fixed at 0.3 for RAG answer** | For RAG, GC is static partial credit — do not use it as a gating signal. |

### Sub-score to failure mapping

| Sub-score | Typical `critical_failures` / warnings |
|-----------|----------------------------------------|
| `EC` | `missing_event: <type>` |
| `FC` | `missing_field: <type>.<field>` |
| `RC` | `relation_violation: <relation_name>` |
| `TC` | `temporal_violation: ...` or execution-window warnings |
| `SC` | Informational notes on missing policy signatures or anchors |
| `XC` | `informational_notes` recommending corroboration adapters |
| `AC` | `--with-anchor` failures or absent anchor events |
| `GC` | `missing_event: ai.action.precommit` (ACP gate violations) |

## JSON output

### Stable contract: `atb verify --format json`

```json
{
  "report_version": "verify.report.v1",
  "profile_id": "atb.profile.rag_answer",
  "pass": false,
  "cas_score": 0.18,
  "cas_grade": "Insufficient",
  "sub_scores": { "EC": 0.75, "FC": 0.9, "RC": 1, "TC": 1, "SC": 0.45, "XC": 0, "AC": 0, "GC": 0.3 },
  "critical_failures": [
    { "kind": "missing_event", "detail": "required event type not present: ai.model.invoked" }
  ],
  "residual_risk": { "level": "High", "drivers": ["missing critical event: ai.model.invoked"] }
}
```

`cas_score` is the base weighted sum (`CAS.Overall`). For corroboration bonus and `effective_score`, use `atb verify --json` (diagnostic mode). See the [verify report contract](../specification/verify-report.md).

## Examples by profile

### RAG answer — missing model invocation

A bundle with request and output but no `ai.model.invoked`:

```json
{
  "profile_id": "atb.profile.rag_answer",
  "pass": false,
  "cas_score": 0.18,
  "cas_grade": "Insufficient",
  "critical_failures": [
    { "kind": "missing_event", "detail": "required event type not present: ai.model.invoked" }
  ]
}
```

**Interpretation:** The chain may be intact, but the bundle lacks proof of the model call. Emit `ai.model.invoked` with `model_provider`, `model_id`, `model_parameters_digest`, and `prompt_digest`.

### Privileged tool action — complete happy path

A bundle with full precommit → policy → execute → commit chain typically returns `cas_grade: "Medium"` or `"High"` depending on anchoring and corroboration. `GC` reflects whether the ACP gating chain is complete.

### Data export — missing human approval after execution

When `data.export.executed` is present without `ai.human.approval`, the profile fails via `required_when`:

```json
{
  "profile_id": "atb.profile.data_export",
  "pass": false,
  "critical_failures": [
    { "kind": "missing_event", "detail": "ai.human.approval required when data exports execute" }
  ]
}
```

**Interpretation:** Export executed but approval evidence is absent. Record `ai.human.approval` with `approval_outcome: approved` before or after execution as your workflow requires.

### Policy decision — minimal pass

A bundle with manifest, request, and policy decision passes obligations. CAS reflects field completeness and source commitment (`sc_mode: policy_decision` gives SC bonus when policy events are present).

### Human override — relation binding

Obligations require `approval_id`/`action_id` binding between approval, precommit, and execution. RC drops when IDs diverge; denied `approval_outcome` lowers RC/GC even if obligations pass on ID match alone.

### Background automation — job lifecycle

Requires `ai.job.scheduled`, `ai.job.started`, and `ai.job.completed` linked by `job_id`. No `ai.request.received` is required for this profile.

## Custom profiles

DSL v1 profiles with `cas_weights` compute EC, FC, XC, and AC only. RC, TC, SC, and GC are fixed at `0.0`. Omitting `cas_weights` disables CAS entirely. See [`profiles.md`](profiles.md#custom-profiles-dsl-v1).

## What CAS does not mean

- The workflow is compliant with regulation or policy.
- An external auditor has attested to the bundle.
- ATB has proved that recording was complete in the real system.
- Retrieval completeness, model-provider internals, or post-delivery handling (see profile blind spots in [`profiles.md`](profiles.md)).
