# Incident capture demo

A deterministic, capture-shaped ATB bundle modelling an **agent incident** as
`atb intercept` would record it: an autonomous support agent invokes a
privileged tool (`delete_user_records`) with **no human approval**, and the
tool **fails**.

It exercises the capture and accountability event types end to end:

| Event | Role |
| --- | --- |
| `atb.llm.request` / `atb.llm.response` | the captured model exchange (bodies digested, not stored raw) |
| `atb.tool.call` | the privileged tool the model invoked — with no preceding approval |
| `ai.action.error` | the tool execution that failed (`error_class: failed`) |
| `atb.exchange.complete` / `atb.session.close` | capture lifecycle |

## Why it matters

The bundle's hash chain verifies (integrity intact), yet the **session index
raises the `tool_without_approval` anomaly** — the core oversight signal for
agent incidents. Integrity proves the record was not altered; the anomaly
proves a control was missing. The two are independent, and ATB surfaces both.

## Regenerate

```bash
go run ./examples/bundles/incident-capture/      # writes incident-capture.atb
```

The `.atb` artefact is gitignored; `make goldens` regenerates it along with the
other example bundles. Behaviour (integrity + the `tool_without_approval`
signal) is covered by `go test ./examples/bundles/incident-capture/`.
