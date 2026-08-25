# Capture guide

Record AI workflow events into a local `.atb` bundle and verify them offline.
This guide covers the main capture paths; it does not claim complete capture of
everything an agent might do.

## Chatlog import

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb import chatlog --from generic-jsonl --input testdata/chatlog.jsonl --snapshot imported_chatlog
atb verify --bundle run.atb/bundle.atb --profile atb.profile.rag_answer --format json
atb view --bundle run.atb/bundle.atb
```

Example input: [`testdata/chatlog.jsonl`](../../testdata/chatlog.jsonl). Schema:
[Chatlog import](../integrations/chatlog-import.md).

## Capture wrapper

`atb capture run` prepares a bundle path, exports capture context to a child
process via environment variables, and runs your command:

```bash
atb capture run --env-prefix MYAPP -- ./agent-runner --config ./agent.yaml
```

The child sees `ATB_BUNDLE_PATH`, `ATB_CAPTURE_RUN_ID`, and related variables.
Instrument the child with the SDK or append events explicitly.

## Intercept proxy

For provider API traffic, use `atb intercept` as an HTTPS forward proxy.
See [Incident forensics](../investigate/incidents.md) for setup, CA trust, and
session review.

## Framework integrations

| Integration | Doc |
| --- | --- |
| LangChain | [langchain.md](../integrations/langchain.md) |
| Vercel AI SDK | [vercel-ai.md](../integrations/vercel-ai.md) |
| MCP | [mcp.md](../integrations/mcp.md) |

OTLP/JSON import preserves additional producer attributes for later evidence
interpretation. Prompt-, argument-, result-, credential-, and body-shaped
values are stored as canonical SHA-256 digests unless a supported semantic
field already defines a stricter representation.

## Automation contract

For CI and scripted integrations (`--format json`, exit codes, dry-run), see
[architecture guide](../concepts/architecture.md).

## Honest limits

Each capture surface sees only what flows through it. ATB proves integrity of
what was recorded, not universal capture. See the
[trust model](../concepts/trust-model.md).
