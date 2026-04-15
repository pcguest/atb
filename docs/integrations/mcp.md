# MCP bridge

## Overview

Model Context Protocol, or MCP, gives agent hosts a stable tool-calling
interface. ATB ships a local stdio MCP bridge via `atb mcp serve` so an
MCP host can initialise bundles, query bundle state, verify recorded
evidence, and record PageIndex RAG events.

The bridge is implemented and tested, but it is still a narrow tool
surface. It wraps the existing CLI and bundle model; it does not
auto-instrument third-party MCP servers or expose the full CLI command
set.

## How it works

An MCP host invokes a tool through the MCP boundary. The ATB bridge
appends structured events into the local bundle as NDJSON and links each
record to the previous record by hash. Reviewers can then run
`atb verify` or `atb trust-report` on the resulting bundle.

```text
MCP host
  -> tool call
    -> ATB MCP bridge
      -> local bundle (.atb, hash chained)
        -> atb verify
        -> atb trust-report
```

## Start the bridge

```bash
atb mcp serve
```

## Configuration

### Claude Desktop

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "atb": {
      "command": "atb",
      "args": ["mcp", "serve"]
    }
  }
}
```

### Claude Code

Register the server for your project:

```bash
claude mcp add atb -- atb mcp serve
```

Or add a `.mcp.json` file at your project root:

```json
{
  "mcpServers": {
    "atb": {
      "command": "atb",
      "args": ["mcp", "serve"]
    }
  }
}
```

## MCP handshake

The server uses protocol version `2024-11-05` and responds to
`initialize` with:

```json
{
  "protocolVersion": "2024-11-05",
  "capabilities": { "tools": {} },
  "serverInfo": { "name": "atb", "version": "<release>" }
}
```

## Available tools

| Tool | Description |
| --- | --- |
| `status` | Return the ATB version, whether a local bundle is present, the chain length, and the current head hash when available. |
| `atb_init` | Initialise the default local bundle (idempotent). |
| `verify` | Verify the current bundle, or a supplied bundle path, and return the JSON report. |
| `rag_index_record` | Record a PageIndex index build as `atb.event.rag_index`. |
| `rag_retrieval_record` | Record a PageIndex retrieval as `atb.event.rag_retrieval`. |

The current bridge does not expose a general-purpose append tool over
MCP. Use the CLI or SDKs directly when you need arbitrary event writes.

### Tool input schemas

#### `status`

No input fields.

#### `atb_init`

No input fields.

#### `verify`

All fields are optional.

| Field | Type | Description |
| --- | --- | --- |
| `path` | string | Path to the `.atb` bundle file. Defaults to `./run.atb/`. |
| `profile` | string | Profile ID or path to a YAML profile file. |
| `trace` | boolean | Emit trace diagnostics during verification. |
| `with_anchor` | boolean | Verify RFC 3161 timestamp token material when present. |
| `quiet` | boolean | Suppress non-JSON terminal chatter. |

#### `rag_index_record`

Required fields: `index_id`, `source_uri`, `page_count`, `node_count`,
`model_id`, `index_hash`. Optional: `indexed_at` (defaults to server
time).

| Field | Type | Description |
| --- | --- | --- |
| `index_id` | string | Unique identifier for this index build. |
| `source_uri` | string | Absolute path or URI of the indexed document. |
| `page_count` | integer | Total pages in the document. |
| `node_count` | integer | Total nodes in the PageIndex tree (recursive count). |
| `model_id` | string | LLM model used to build the index. |
| `index_hash` | string | SHA-256 hex digest of `json.dumps(tree, sort_keys=True)`. |
| `indexed_at` | string | RFC 3339 timestamp. Defaults to server time if omitted. |

#### `rag_retrieval_record`

Required fields: `query`, `retrieval_id`, `index_id`, `node_id`,
`node_title`, `source_uri`, `page_start`, `page_end`, `model_id`,
`latency_ms`. Optional: `node_summary`.

| Field | Type | Description |
| --- | --- | --- |
| `query` | string | The retrieval query string. |
| `retrieval_id` | string | Unique identifier for this retrieval result. |
| `index_id` | string | Foreign key -- must match a prior `rag_index_record.index_id`. |
| `node_id` | string | PageIndex `node_id` from the matched tree node. |
| `node_title` | string | PageIndex `title` field from the matched tree node. |
| `source_uri` | string | Absolute path or URI of the source document. |
| `page_start` | integer | PageIndex `start_index` of the matched node. |
| `page_end` | integer | PageIndex `end_index` of the matched node. |
| `model_id` | string | LLM model used for the retrieval call. |
| `latency_ms` | integer | Wall-clock milliseconds for the retrieval call. |
| `node_summary` | string | PageIndex `summary` field. Omit if absent. |

## CLI quickstart

```bash
atb bundle new

atb append ai.request.received --data \
  '{"request_id":"req-001","actor_id_hash":"sha256-actor-abc","purpose_tag":"rag_answer"}'

atb append ai.model.invoked --data \
  '{"model_provider":"openai","model_id":"gpt-4o","model_parameters_digest":"sha256-params-def","prompt_digest":"sha256-prompt-ghi"}'

atb verify --format json
```

## SDK usage

The current Python and TypeScript SDKs do not provide MCP-specific
helpers. Use the generic `Bundle.append(...)` path and emit the
canonical event types directly.

Python:

```python
from atb import Bundle

bundle = Bundle()
bundle.append(
    "ai.request.received",
    {
        "request_id": "req-001",
        "actor_id_hash": "sha256-actor-abc",
        "purpose_tag": "rag_answer",
    },
)
bundle.append(
    "ai.model.invoked",
    {
        "model_provider": "openai",
        "model_id": "gpt-4o",
        "model_parameters_digest": "sha256-params-def",
        "prompt_digest": "sha256-prompt-ghi",
    },
)
```

TypeScript:

```ts
import { Bundle } from "@pcguest/atb-sdk";

const bundle = new Bundle();

bundle.append("ai.request.received", {
  request_id: "req-001",
  actor_id_hash: "sha256-actor-abc",
  purpose_tag: "rag_answer",
});

bundle.append("ai.model.invoked", {
  model_provider: "openai",
  model_id: "gpt-4o",
  model_parameters_digest: "sha256-params-def",
  prompt_digest: "sha256-prompt-ghi",
});
```

## Output and review

When a reviewer runs `atb verify --format json`, the report shows the
matched profile, whether it passed, any critical failures or warnings,
and CAS results when the selected profile supports CAS. When they run
`atb trust-report`, the same evidence is rendered as review-oriented
sections.

## Compliance note

Bundles produced through the MCP bridge can support evidence collection
for AI-system logging and review obligations, including EU AI Act
Article 12 and NIST AI RMF governance or management discussions. They do
not provide formal compliance on their own. System design, deployment
controls, identity assurance, and evidence outside the bundle still
matter.

## Limitations

- Recording arbitrary external tool activity still requires explicit ATB instrumentation or an MCP host that calls the ATB bridge tools.
- The bridge does not auto-instrument third-party MCP servers.
- Auto-detection matches on recorded AI event families. Bundles that emit only custom event types may require explicit `--profile` selection.
- `rag_index_record` and `rag_retrieval_record` record PageIndex indexing and retrieval evidence only. They do not record downstream model answer provenance unless the surrounding workflow emits those events separately.
- The PageIndex RAG tools emit `atb.event.rag_index` and `atb.event.rag_retrieval`. The `atb.profile.rag_answer` profile requires `ai.retrieval.executed`. These are different event types; a bundle produced by the MCP RAG tools alone will not satisfy the retrieval sub-score of `atb.profile.rag_answer`. To use that profile, the surrounding workflow must also emit `ai.retrieval.executed`.
