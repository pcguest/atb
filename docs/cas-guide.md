# CAS guide

CAS is the Completeness Assurance Score. It measures how much of the
expected evidence ATB can see inside the selected profile. It is about
recorded evidence completeness within that declared workflow boundary. It
is not a compliance decision.

Current `cas_grade` values are:

- `Insufficient` (`< 0.30`): very little of the expected evidence is present.
- `Low` (`>= 0.30`): some evidence is present, but important gaps remain.
- `Medium` (`>= 0.60`): the workflow is reasonably evidenced, with some gaps still visible.
- `High` (`>= 0.85`): the recorded evidence is strong for the selected profile.

If your score is low, start with `critical_failures`. That array tells you
which required event, field, relation, or timing check failed. Fix those
gaps at the workflow call site, record a new bundle, and run `atb verify`
again.

CAS does not mean:

- the workflow is compliant
- an external auditor has attested to the bundle
- ATB has proved that recording was complete in the real system

It is a local scoring model over what ATB can see in the bundle.

Example: a RAG answer bundle returns `cas_score: 0.18` and
`cas_grade: "Insufficient"`, with:

```json
{
  "critical_failures": [
    {
      "kind": "missing_event",
      "detail": "required event type not present: ai.model.invoked"
    }
  ]
}
```

That means the chain may still be intact, but the bundle is missing a
critical record of the model call itself. To improve the score, emit
`ai.model.invoked` with the required fields such as `model_provider`,
`model_id`, `model_parameters_digest`, and `prompt_digest`, then verify
again. If `critical_failures` is empty but the score is still weak, look
at missing optional evidence such as retrieval, anchoring, signatures, or
external corroboration.
