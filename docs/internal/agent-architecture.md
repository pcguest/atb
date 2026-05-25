# ATB Agent — internal architecture note

Internal design note for the local ATB Agent. Not a user-facing guide.

**Current code (v1.12.0):** `atb agent run` starts a loopback HTTP server with
health/info endpoints, capture session open/append/close APIs, and a read-only
workspace bundle index. Configuration is via `ATB_AGENT_LISTEN_ADDR` (default
`127.0.0.1:6180`) and `ATB_AGENT_DATA_DIR` (default `~/.atb/agent`, or
`./data/agent` when `HOME` is unset). Closed session bundles are persisted
under the Agent data directory.

## Three-layer model

| Layer | Role | Current implementation |
| --- | --- | --- |
| **1 — CLI and SDKs** | Developer-facing capture, verify, export, and push | `atb` CLI; Python and TypeScript SDKs; `atb capture run` |
| **2 — ATB Agent** | Optional local background service: workspace, APIs, MCP and viewer coordination | `atb agent run` — health/info, capture sessions, workspace bundle index |
| **3 — Viewer** | Human review of workflow evidence | `atb view` + embedded Next.js UI; read-only bundle API in `pkg/api/v1` |

Layer 1 remains fully functional without Layer 2. The Agent is an opt-in
install-time convenience, not a prerequisite for bundle integrity.

## Implemented Agent capabilities

- Workspace registry under `ATB_AGENT_DATA_DIR` to list and track local bundles.
- Small JSON API to create, open, append to, and close bundle contexts
  (delegating all writes to `internal/bundle`).
- Safe local append path for SDKs: one active bundle per session, advisory
  locking preserved, explicit opt-in through Agent environment variables.
- Graceful shutdown, structured logging, and health/readiness suitable for
  local service supervision.

## Planned Agent capabilities (not yet implemented)

- Read-only query API for the viewer (reuse `pkg/api/v1` patterns; multi-bundle
  navigation instead of one bundle per `atb view` invocation).
- Compose MCP stdio (today: standalone `atb mcp serve`) and viewer HTTP on a
  single long-lived process with shared workspace state.
- OS installer or service manager integration.

Update this list after each Agent prompt so it matches shipped behaviour.

## Out of scope for the Agent

- Direct cloud sync or Custos ingestion (explicit `atb push` and separate
  custodian products remain the export path).
- Credentials management, OAuth, or identity-provider integration.
- Host-level or network-level capture (eBPF, transparent proxy, packet
  inspection).
- Mutating bundle contents through unauthenticated remote interfaces (loopback
  and session tokens only for local use).
- Replacing the CLI one-shot commands; `append`, `verify`, `capture run`, and
  related tools stay available without the Agent.

## Related notes

- [Automation inventory](automation-notes.md) — SDK and CLI capture helpers.
- [MCP integration](../integrations/mcp.md) — stdio bridge (standalone today).
- [Viewer specification](../spec-dashboard.md) — single-bundle review UI.
