# Tenon pilot fixture

A deterministic, synthetic AI-agent session for the Tenon pilot path. It
answers the first evaluator question without external credentials:

> What did this AI agent do, was the privileged action approved, can the
> evidence be independently verified, and can it be placed into custody?

The fixture records:

| Evidence | Purpose |
| --- | --- |
| `ai.request.received`, `ai.action.precommit`, `ai.policy.decision`, `ai.action.executed`, `ai.action.committed` | profile/CAS evidence for the approved privileged action |
| `ai.human.approval` | profile-level approval evidence for the approved action |
| `atb.capture.scope` | explicit statement that this is synthetic digest-only fixture evidence |
| `atb.llm.request` / `atb.llm.response` | capture-shaped model exchange with body digests only |
| first `atb.tool.call` + `ai.action.error` | anomalous privileged action that was not approved and failed |
| `atb.human.approval` + second `atb.tool.call` | approved privileged tool action in the captured session |
| `atb.exchange.complete` / `atb.session.close` | closed session lifecycle evidence |

## Generate and review

```bash
go run ./examples/bundles/tenon-pilot/
atb verify \
  --bundle examples/bundles/tenon-pilot/tenon-pilot.atb \
  --profile atb.profile.privileged_tool_action \
  --format json
atb incident list --bundle examples/bundles/tenon-pilot/tenon-pilot.atb
atb incident report \
  --bundle examples/bundles/tenon-pilot/tenon-pilot.atb \
  --session sess-tenon-pilot-0001
```

The incident report should show an intact chain and `tool_without_approval`
for the first privileged tool call. The profile report should pass and include
CAS output for the approved action path.

Open the local viewer when built from source:

```bash
atb view \
  --bundle examples/bundles/tenon-pilot/tenon-pilot.atb \
  --profile atb.profile.privileged_tool_action
```

## Optional Mortise handoff

Submit only when a compatible Mortise instance is explicitly available:

```bash
ATB_MORTISE_TOKEN=<token> \
atb incident export \
  --bundle examples/bundles/tenon-pilot/tenon-pilot.atb \
  --session sess-tenon-pilot-0001 \
  --mortise-endpoint http://127.0.0.1:8088
```

This fixture is demonstration evidence, not production capture. It proves ATB
verification, incident reporting, profile/CAS evaluation, and the Mortise
client handoff shape without executing live provider calls or destructive
tools.
