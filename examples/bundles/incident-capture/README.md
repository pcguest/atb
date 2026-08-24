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

The deliberately incomplete
[`application-log.txt`](./application-log.txt) reports only a generic provider
failure and a normal session close. It omits the privileged tool name and the
absence of a matching recorded approval. That is representative of the logs an
investigator may be left with after the producing application is no longer a
trusted source.

The bundle's hash chain verifies (integrity intact), while the **session index
raises the `tool_without_approval` anomaly**. Integrity proves the presented
records have not been altered; the anomaly shows that no matching approval is
present in the preceding recorded evidence. It does not prove that no approval
existed outside the capture boundary.

## Review it

```bash
# Discover the sessions in the bundle:
atb incident list --bundle examples/bundles/incident-capture/incident-capture.atb

# Then report on one:
atb incident report \
  --bundle examples/bundles/incident-capture/incident-capture.atb \
  --session sess-incident-7731
```

Produces a session-scoped forensic report: **Integrity PASS**, anomalies
**`tool_without_approval`**, and the ordered event list (privileged
`atb.tool.call` → failed `ai.action.error`) with each row's record hash. Add
`--format json` for a machine-readable report.

Compare the sources directly:

```bash
cat examples/bundles/incident-capture/application-log.txt
atb incident report \
  --bundle examples/bundles/incident-capture/incident-capture.atb \
  --session sess-incident-7731
```

The application log is a claim made by the producing system. The ATB report is
a deterministic interpretation of portable records whose order and integrity
can be independently checked after the incident.

## Regenerate

```bash
go run ./examples/bundles/incident-capture/      # writes incident-capture.atb
```

The `.atb` artefact is gitignored; `make goldens` regenerates it along with the
other example bundles. Behaviour (integrity + the `tool_without_approval`
signal) is covered by `go test ./examples/bundles/incident-capture/`.

For the full capture → discover → review path, see the
[incident forensics walkthrough](../../../docs/guides/incident-forensics.md).
