# `atb verify --format json` schema

Source of truth: `internal/verify/report.go` (`VerifierReport`).

## JSON shape

```json
{
  "bundle_path": "run.atb/bundle.atb",
  "profile_id": "atb.profile.rag_answer",
  "pass": true,
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
  "required_warnings": [],
  "informational_notes": [],
  "exclusions": [],
  "residual_risk": "Low"
}
```

## Fields

- `bundle_path` (`string`): bundle path used for evaluation.
- `profile_id` (`string`): canonical profile ID when a profile matched or was selected. Empty when no profile result is present.
- `pass` (`boolean`): whether the selected or matched profile passed. When no profile result is present, this remains `false`.
- `cas_score` (`number`, optional): overall CAS score when the profile supports CAS.
- `cas_grade` (`string`, optional): `High`, `Medium`, `Low`, or `Insufficient`.
- `sub_scores` (`object`, optional): CAS sub-score map keyed by `EC`, `FC`, `RC`, `TC`, `SC`, `XC`, `AC`, and `GC`.
- `critical_failures` (`array`): blocking failures. Each item contains `kind` and `detail`.
- `required_warnings` (`array[string]`): non-blocking warnings from the evaluated profile.
- `informational_notes` (`array[string]`): non-blocking notes from verification.
- `exclusions` (`array[string]`, optional): declared blind spots for the matched profile.
- `residual_risk` (`string`): `Low`, `Medium`, `High`, or `Critical`.

## Semantics

- `--format json` is the stable automation contract for `atb verify`.
- It is profile-oriented. If you need chain internals such as
  `integrity.chain_valid`, use `atb verify --json` instead.
- A bundle can be internally intact and still return `pass: false` when
  the selected profile is missing required evidence.

## What to do next

- If `profile_id` is empty, rerun `atb verify --profile <id> --format json` with the workflow profile you expect.
- If `pass` is `false`, inspect `critical_failures` first. Those are the missing or inconsistent items that blocked the result.
- If `pass` is `true` but `residual_risk` is still `Medium` or `High`, use `required_warnings`, `informational_notes`, and the CAS guide in [`../cas-guide.md`](../cas-guide.md) to decide what to instrument next.
