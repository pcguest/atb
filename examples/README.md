# ATB examples

Use this directory for public, product-facing examples only.

## Quickstart

A minimal privileged_tool_action workflow that produces a verifiable bundle with a CAS score. Run with `bash run.sh`.

Path: [examples/quickstart/](./quickstart/)

## SDK examples

- [Python LangChain example](./python/langchain_bot.py)
- [Python LangGraph example](./python/langgraph_demo.py)
- [TypeScript Vercel AI example](./typescript/vercel-chat-bot.ts)

## Demo scripts

Profile workflow helper demos (5-minute runnable scripts with CLI verify): [examples/demo/](./demo/)

## Composite demo bundle

Support-escalation narrative bundle for dashboard demos: [examples/bundles/demo-workflow/](./bundles/demo-workflow/)

## Incident capture demo

Capture-shaped agent-incident bundle (privileged tool call with no approval, plus a failed tool execution) that verifies clean yet raises the `tool_without_approval` oversight anomaly: [examples/bundles/incident-capture/](./bundles/incident-capture/)

## Tenon pilot fixture

Credential-free pilot fixture with one approved privileged action, one anomalous privileged tool call, local verification, incident reporting, viewer inspection, and optional Mortise handoff: [examples/bundles/tenon-pilot/](./bundles/tenon-pilot/)

## Quickstart variants

- [README quickstart](../README.md#quickstart) — `atb.profile.policy_decision` (support triage deny)
- [examples/quickstart/run.sh](./quickstart/run.sh) — `atb.profile.privileged_tool_action` (six-event privileged action)

## Bundle examples

- [Project bootstrap bundle](./bundles/project-bootstrap.atb) — valid fixture for `atb verify`
- [Tampered bootstrap bundle](./bundles/project-bootstrap-tampered.atb) — hash-chain failure demo; see [tamper demo guide](../docs/guides/tamper-demo.md)
- [Profile fixtures](./bundles/profiles/) — pass/fail bundle pairs for each of the six built-in obligation profiles

The bundle examples are small verified ATB bundles that demonstrate the on-disk format without exposing internal project logs or operational notes.

### Regenerating the bundles

The `.atb` files under `examples/bundles/` are **generated artefacts and are gitignored** — a fresh clone does not contain them. Build the CLI once, then regenerate every example and demo bundle with a single command:

```bash
make build     # builds ./atb with the embedded viewer
make goldens   # regenerates profiles/, project-bootstrap, demo-workflow, incident-capture, and tenon-pilot
```

Each generator asserts its bundle's expected verify outcome (pass or fail), so `make goldens` doubles as a smoke test of the demo path.
