# Automation harness

The TypeScript and Python SDKs ship `AutomationSession` for chained AI workflow
capture. It composes existing profile helpers so you open one bundle
connection, emit events across successive model and tool hops, and flush
without repeating bootstrap boilerplate at every call site.

For inventory and design rationale see [internal automation notes](../internal/automation-notes.md).

## ATB Agent (optional)

`AutomationSession` writes bundles in-process, via paths from `atb capture run`,
or through the optional [ATB Agent](agent.md) (`atb agent run`). Set
`ATB_AGENT_URL` to an explicit loopback Agent URL, or set `ATB_AGENT_AUTO=1` to
use the default `http://127.0.0.1:6180` when the Agent health check passes. CLI
and SDK workflows remain valid without the Agent.

## Compared to manual profile helpers

Without a session, a RAG path requires creating a `Bundle`, a `WorkflowContext`,
calling `bootstrapRequest()`, then emitting each event type in order. With a
session the same flow collapses to a few method calls on one object.

## Example: RAG answer chain

```typescript
import { AutomationSession } from "@pcguest/atb-sdk";

const session = AutomationSession.open({
  savePath: "run.atb/bundle.atb",
  actorId: "agent-1",
  purposeTag: "rag_answer",
  requestId: "req-001",
});

session.logRetrieval({
  query: userQuestion,
  corpusId: "product-docs",
  corpusVersion: "2026-05",
  topK: 5,
  resultSet: chunks,
});

session.logModelInvocation({
  provider: "openai",
  model: "gpt-4o",
  prompt: composedPrompt,
  parameters: { temperature: 0.2 },
});

const answer = await callModel(composedPrompt);
session.logModelOutput({ output: answer });
session.logResponseSent({ output: answer });
session.close({ snapshotName: "rag_turn_complete" });
```

Verify the bundle with:

```bash
atb verify --bundle run.atb/bundle.atb --profile atb.profile.rag_answer --format json
```

## Example: under `atb capture run`

When the agent process is launched via `atb capture run`, resume the prepared
bundle automatically:

```typescript
import { AutomationSession } from "@pcguest/atb-sdk";

const session = AutomationSession.fromCaptureEnvironment();
if (session) {
  await session.runToolAction(
    {
      actionType: "deploy",
      targetResourceId: "svc-prod",
      intendedEffect: "promote release",
      actionParameters: { version: "1.2.0" },
    },
    () => deploy()
  );
  session.close();
}
```

## Python parity

The Python SDK exposes the same session concept via `atb.automation_session`.
Use `AutomationSession.open(...)` for disk-backed capture or
`AutomationSession.from_capture_environment()` inside an `atb capture run`
child process.

## API summary

| Method | Events emitted |
| --- | --- |
| `beginRequest()` | `ai.request.received` (once per session) |
| `logRetrieval()` | `ai.retrieval.executed` |
| `logModelInvocation()` | `ai.model.invoked` |
| `logModelOutput()` | `ai.model.output` |
| `logResponseSent()` | `ai.response.sent` |
| `runToolAction()` | request + precommit → policy → executed → committed |
| `logPolicyDecision()` | request + precommit + policy decision |
| `snapshot()` / `close()` | `atb.snapshot` + persist |
