# ATB automation inventory and harness design

Internal design note for SDK/CLI automation work. Not a user-facing guide; see
[automation harness](../guides/automation-harness.md) for usage.

## Existing automation capabilities

### CLI

| Entry point | Role | Chaining support |
| --- | --- | --- |
| `atb capture run` | Prepares bundle path, injects `ATB_BUNDLE_PATH` / `ATB_CAPTURE_RUN_ID`, runs child process, optional snapshot + verify | Process-scoped; child must write events via SDK or export JSONL for import |
| `atb import chatlog` | Retrospective reconstruction from saved chatlogs (`generic-jsonl`, `openai-jsonl`) | One-shot import; marks bundles as retrospective |
| `atb mcp serve` | Stdio MCP bridge: init bundle, append events, verify, PageIndex RAG tools | Host-driven tool calls; narrow surface, no auto-instrumentation of third-party MCP servers |

### SDK profile workflow helpers (Go parity in TS + Python)

Both TypeScript and Python ship `WorkflowContext` plus profile-specific recorders:

- `PolicyDecisionRecorder` — `atb.profile.policy_decision`
- `ActionGate` — privileged action precommit / policy / executed (TS + Python)
- `DataExportGate`, `HumanOverrideGate`, `BackgroundJobTracker` — remaining built-in profiles

These emit correct event sequences but require callers to wire each hop manually.

### Framework integrations

| Integration | Language | Events emitted |
| --- | --- | --- |
| `atbMiddleware` (Vercel AI SDK) | TypeScript | `ai.llm.call`, `ai.tool.exec`, `ai.chain.run` (+ optional RAG events) |
| `gateVercelTool` | TypeScript | Privileged tool gate around Vercel AI tools |
| `ATBCallbackHandler` | Python | LangChain lifecycle → `ai.model.invoked`, `ai.model.output`, etc. |
| `gate_langchain_tool` | Python | LangChain tool gate |

### Capture helpers

- `isCaptureEnvironment()` / `ATB_BUNDLE_PATH` detection (TS)
- `parse_chatlog()` (Python) mirroring Go `internal/capture`

## Friction points

1. **Repeated bootstrap** — every profile helper expects callers to create a `Bundle`, pass identity fields, and often call `bootstrapRequest()` before the first event.
2. **No session lifetime** — no single object owns open → emit → flush/snapshot for multi-hop AI workflows within one process.
3. **Profile vs integration event shapes** — Vercel/LangChain middleware emit informational `ai.llm.call` events; obligation profiles (`rag_answer`, `privileged_tool_action`) need `ai.model.invoked` / gate chains. Developers must choose and compose manually.
4. **Capture env is detect-only** — SDKs can read `ATB_BUNDLE_PATH` but do not load/append to the prepared bundle automatically.
5. **Committed gap** — `ActionGate` stops at `ai.action.executed`; `privileged_tool_action` verification expects `ai.action.committed` as well.

## Chosen approach: Option A — TypeScript `AutomationSession`

**Why:** Smallest high-impact change. TypeScript already has the richest integration surface (Vercel AI middleware + profile gates). A session wrapper composes existing helpers without new event types or CLI changes.

**Scope (this session):**

- `AutomationSession` in `sdk/typescript/src/automation-session.ts`
- Convenience methods: request bootstrap, RAG model/retrieval/response, tool action (with commit), policy decision, save/snapshot/close
- `fromCaptureEnvironment()` to append into `atb capture run` bundles
- Unit tests + short guide under `docs/guides/automation-harness.md`

**Non-goals:**

- Python parity (future)
- YAML/DSL config layer (Option B)
- CLI capture harness changes (Option C)
- New event types or verify.report.v1 / profile schema changes
- Full orchestration framework or host-level capture

**Future expansion:**

- Python `AutomationSession` mirroring TS API
- Optional bridge from `AutomationSession` to Vercel middleware for mixed profile + integration traces
- Snapshot name validation shared with CLI
