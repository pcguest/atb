# ATB Agent (local service)

The ATB Agent is an optional local background service. It coordinates bundle
workspaces, capture sessions, and (future) viewer integration on a single
machine. **You do not need the Agent** to create, append, verify, or review
bundles: the CLI, SDKs, `atb capture run`, and `atb view` continue to work
independently.

For local desktop usage with multiple session bundles, start the Agent and
point SDK capture at it — see [First-run behaviour](#first-run-behaviour) below.

## Current release

Start the Agent:

```bash
atb agent run
```

The process binds to the loopback interface only.

### Configuration layering

Settings are resolved in this order (highest priority first):

1. **Environment variables** — `ATB_AGENT_DATA_DIR`, `ATB_AGENT_LISTEN_ADDR`
2. **Optional config file** — `agent` object in `~/.atb/config.json` or
   `./.atb/config.json` (same JSON file shape as other ATB CLI settings; only
   the `agent` block is read by the Agent today)
3. **Defaults** — listen `127.0.0.1:6180`; data directory `~/.atb/agent`
   (or `./data/agent` when `HOME` is unset)

| Variable / key | Default | Purpose |
| --- | --- | --- |
| `ATB_AGENT_LISTEN_ADDR` / `agent.listen_addr` | `127.0.0.1:6180` | Loopback listen address |
| `ATB_AGENT_DATA_DIR` / `agent.data_dir` | `~/.atb/agent` | Workspace root for session bundles |

Example config file fragment:

```json
{
  "version": 1,
  "agent": {
    "listen_addr": "127.0.0.1:6180",
    "data_dir": "/home/you/.atb/agent"
  }
}
```

`atb view` does not read the Agent config today; it remains an ephemeral
single-bundle server. Both commands can coexist on one machine without shared
process state — only optional path/env conventions overlap.

### HTTP endpoints

| Method | Path | Response |
| --- | --- | --- |
| `GET` | `/healthz` | `200` and `{"status":"ok"}` |
| `GET` | `/v1/info` | ATB version, optional build metadata, and config summary |
| `POST` | `/v1/session/open` | Open a capture session |
| `POST` | `/v1/session/{id}/event` | Append a capture event |
| `POST` | `/v1/session/{id}/close` | Close session and persist bundle metadata |
| `GET` | `/v1/workspace/bundles` | Read-only list of closed session bundles |

Example:

```bash
curl -s http://127.0.0.1:6180/healthz
curl -s http://127.0.0.1:6180/v1/info
curl -s http://127.0.0.1:6180/v1/workspace/bundles
```

The Agent logs startup, listen address, and clean shutdown to stderr. Send
`SIGINT` or `SIGTERM` to stop.

## First-run behaviour

On the first start (or whenever the data directory is empty), the Agent:

- Creates the workspace directory (default `~/.atb/agent`).
- Writes a short `README` in that directory describing local storage and
  capture options (not tracked in git).
- Logs a **first run** message with the data path and suggested next steps.

Session bundles are stored under:

```text
${ATB_AGENT_DATA_DIR}/sessions/<session_id>/bundle.atb
${ATB_AGENT_DATA_DIR}/sessions/<session_id>/meta.json
```

To get your first bundle into the workspace:

1. Start the Agent: `atb agent run`
2. Capture through one of:
   - TypeScript `AutomationSession` with `ATB_AGENT_URL` or `ATB_AGENT_AUTO`
   - `atb capture run` (when wired to the Agent in your environment)
3. Open the viewer workspace page (`/workspace/`) when served in Agent mode to
   see closed sessions listed (read-only index).

Change the data directory with `ATB_AGENT_DATA_DIR` or `agent.data_dir` in
`~/.atb/config.json`.

## Relationship to other surfaces

| Surface | Today | With Agent |
| --- | --- | --- |
| CLI (`atb append`, `atb verify`, …) | Direct bundle access | Unchanged; remains usable without the Agent |
| `atb mcp serve` | Standalone stdio MCP bridge | Future: hosted by the Agent with shared workspace |
| `atb view` | Ephemeral HTTP server for one bundle | Single-bundle mode unchanged; workspace picker when Agent-hosted |
| SDKs | In-process or file-based bundle writers | Optional capture via Agent API when env indicates availability |

## Planned direction (not yet available)

- Multi-bundle viewer routes backed by Agent read APIs.
- Integrate MCP and viewer on one long-lived local process.

The Agent remains **local-only and opt-in**. It does not replace explicit CLI
commands or push bundles to cloud storage automatically.

## See also

- [Instrumentation checklist](instrumentation-checklist.md)
- [Automation harness (TypeScript)](automation-harness.md)
- [MCP integration](../integrations/mcp.md)
- [Viewer specification](../spec-dashboard.md)
