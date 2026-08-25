# `atb verify --format json` schema

Source of truth: `internal/verify/report.go` (`VerifierReport`, `VerifyReportVersion = "verify.report.v1"`).

This is the **stable automation contract**. For chain internals, nested `integrity`, corroboration bonus, and `effective_score`, use `atb verify --json` (diagnostic mode).

## JSON shape

```json
{
  "report_version": "verify.report.v1",
  "bundle_path": "run.atb/bundle.atb",
  "profile_id": "atb.profile.rag_answer",
  "profile_version": 1,
  "pass": true,
  "gate_result": {
    "pass": true,
    "chain_valid": true,
    "profile_pass": true,
    "anchor_required": false,
    "anchor_pass": null
  },
  "cas_score": 0.87,
  "cas_grade": "High",
  "sub_scores": {
    "EC": 1,
    "FC": 1,
    "RC": 1,
    "TC": 1,
    "SC": 0.8,
    "XC": 0,
    "AC": 0,
    "GC": 1
  },
  "critical_failures": [],
  "obligations": [
    {
      "id": "required:ai.request.received",
      "kind": "required_event",
      "event_type": "ai.request.received",
      "severity": "critical",
      "status": "pass",
      "message": ""
    }
  ],
  "required_warnings": [],
  "informational_notes": [],
  "exclusions": [
    "Limitation: retrieval completeness beyond recorded corpus/version. | Mitigation: ai.retrieval.executed with corpus metadata (L2). | Residual: corpus content not attested."
  ],
  "signatures": [],
  "provability_gaps": [],
  "residual_risk": {
    "level": "Low",
    "drivers": [],
    "recommended_next_evidence": []
  }
}
```

## Top-level fields

| Field | Type | Description |
|-------|------|-------------|
| `report_version` | string | Always `verify.report.v1`. |
| `bundle_path` | string | Bundle path used for evaluation. |
| `retrospective` | boolean | Optional. Set when evaluating a historical bundle outside live capture. |
| `profile_id` | string | Canonical profile ID when matched or selected. Empty when no profile result. |
| `profile_version` | integer | Profile schema version when a profile result is present. Omitted when no profile matched. |
| `pass` | boolean | Overall gate pass: chain valid, profile pass, and anchor pass when required. |
| `gate_result` | object | Decomposed gate outcome (see below). |
| `cas_score` | number | Weighted CAS overall (`CAS.Overall`). Omitted when profile does not support CAS. |
| `cas_grade` | string | `High`, `Medium`, `Low`, or `Insufficient`. |
| `corroboration_bonus` | number | Optional bonus applied when corroboration policy is active (`--with-anchor`). |
| `effective_score` | number | Optional `cas_score + corroboration_bonus`, capped at 1.0. |
| `sub_scores` | object | Map keyed by `EC`, `FC`, `RC`, `TC`, `SC`, `XC`, `AC`, `GC`. |
| `critical_failures` | array | Blocking failures. Each item: `id?`, `kind`, `detail`. |
| `obligations` | array | Structured obligation results with `id`, `kind`, `event_type`, `severity`, `status`, `message`. |
| `required_warnings` | array[string] | Non-blocking warnings from the profile. |
| `informational_notes` | array[string] | Non-blocking verification notes. |
| `exclusions` | array[string] | Profile blind-spot declarations (out-of-scope items). |
| `signatures` | array | Bundle and source signature provenance when present. |
| `provability_gaps` | array | Known provability limitations with mitigation guidance. |
| `residual_risk` | object | Residual risk assessment (see below). |

## `gate_result`

| Field | Type | Description |
|-------|------|-------------|
| `pass` | boolean | Combined gate: chain + profile + anchor (when required). |
| `chain_valid` | boolean | Hash chain integrity. |
| `profile_pass` | boolean | Selected profile obligation pass/fail. |
| `anchor_required` | boolean | Whether anchor verification was requested/required. |
| `anchor_pass` | boolean \| null | Anchor verification outcome when required; null otherwise. |

## `critical_failures[]`

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Optional stable failure identifier (e.g. `required:ai.model.invoked`). |
| `kind` | string | Failure category: `missing_event`, `missing_field`, `relation_violation`, `temporal_violation`, etc. |
| `detail` | string | Human-readable description. |

## `residual_risk`

| Field | Type | Description |
|-------|------|-------------|
| `level` | string | `Low`, `Medium`, `High`, or `Critical`. |
| `drivers` | array[string] | Optional risk drivers. |
| `recommended_next_evidence` | array[string] | Optional suggested instrumentation. |

## Remote verify extension

When using `atb verify --remote s3://...`, the response wraps `VerifierReport` with:

| Field | Type | Description |
|-------|------|-------------|
| `remote_uri` | string | S3 URI verified. |
| `key_hash_ok` | boolean | Object key hash matches computed head hash. |
| `key_hash_warn` | string | Warning when key hash mismatch detected. |

## Semantics

- `--format json` is profile-oriented and stable for CI/automation.
- A bundle can be internally intact (`chain_valid: true`) and still return `pass: false` when required evidence is missing.
- `cas_score` reflects base weighted completeness, not `effective_score` after corroboration bonus. Use `--json` for bonus details.
- CAS is diagnostic when `profile_pass` is false; it does not overturn FAIL.

## What to do next

- If `profile_id` is empty, rerun with `--profile <id>`.
- If `pass` is false, inspect `critical_failures` first.
- If `pass` is true but `residual_risk.level` is `Medium` or higher, review `required_warnings`, `sub_scores`, and the [CAS guide](../evidence/cas.md).
- Machine-readable contract: `atb verify --schema` prints frozen JSON Schema (`verify.report.v1.schema.1`); `atb verify --schema --schema-out path.json` writes it to disk.

## Schema versioning

`verify.report.v1.schema.1` is a strict custody contract. The embedded schema
sets `additionalProperties: false`, and conformance tests reject top-level
report fields that are not declared in that schema. Future additions to the
automation report require a new schema identifier (for example
`verify.report.v1.schema.2`, or `verify.report.v2` for semantic changes) and a
matching CHANGELOG entry.
