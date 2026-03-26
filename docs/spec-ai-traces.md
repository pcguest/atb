# ATB AI Trace Specification (Phase 5)

This specification defines how framework runtime callbacks map into ATB events without changing the core ATB bundle record format in `docs/spec-v1.0.md`.

This document supersedes the legacy `langchain.*` taxonomy in `docs/spec-v1.0.md` for new integrations. Legacy types remain valid only for backward-compatible verification of older bundles.

## Goals

- Keep integrations low-friction and opt-in.
- Keep traces machine-parseable and human-auditable.
- Keep privacy handling explicit and deterministic.
- Keep output compatible with existing ATB verification and hashing.

## Event Mapping

### LangChain

| LangChain Callback | ATB `event.type` | Required `event.data.phase` |
| --- | --- | --- |
| `on_llm_start` | `ai.llm.call` | `start` |
| `on_llm_new_token` | `ai.llm.call` | `delta` |
| `on_llm_end` | `ai.llm.call` | `end` |
| `on_llm_error` | `ai.llm.call` | `error` |
| `on_tool_start` | `ai.tool.exec` | `start` |
| `on_tool_end` | `ai.tool.exec` | `end` |
| `on_tool_error` | `ai.tool.exec` | `error` |
| `on_chain_start` | `ai.chain.run` | `start` |
| `on_chain_end` | `ai.chain.run` | `end` |
| `on_chain_error` | `ai.chain.run` | `error` |

### Vercel AI SDK / LangChain.js

Use the same ATB event types and phases:

- LLM lifecycle: `ai.llm.call`
- Tool lifecycle: `ai.tool.exec`
- Chain or step lifecycle: `ai.chain.run`

## Canonical `event.data` Envelope

All AI integration events MUST use this envelope.

```json
{
  "trace_id": "trc_...",
  "span_id": "spn_...",
  "parent_span_id": "spn_parent_...",
  "framework": "langchain",
  "framework_version": "0.2.x",
  "phase": "start",
  "run_id": "framework-run-id",
  "context": {},
  "privacy": {
    "mode": "off",
    "redaction_enabled": false,
    "pii_ruleset": "phase4-gdpr-v1"
  },
  "timing": {
    "started_at": "2026-03-09T09:15:02Z",
    "ended_at": null,
    "latency_ms": null
  },
  "status": {
    "ok": true,
    "error": null
  }
}
```

Required fields:

- `trace_id`, `span_id`, `framework`, `phase`, `context`, `privacy`, `timing`, `status`
- `parent_span_id` is optional
- `run_id` is optional but recommended

## Context Payloads by Type

### `ai.llm.call`

```json
{
  "provider": "openai",
  "model": "gpt-4.1-mini",
  "prompt": {
    "text": "Tell me a joke",
    "sha256": "sha256:..."
  },
  "completion": {
    "text": "Why did the...",
    "sha256": "sha256:..."
  },
  "token_usage": {
    "prompt_tokens": 12,
    "completion_tokens": 28,
    "total_tokens": 40
  },
  "temperature": 0.2,
  "max_tokens": 256,
  "finish_reason": "stop"
}
```

### `ai.tool.exec`

```json
{
  "tool_name": "weather.lookup",
  "tool_version": "1.0.0",
  "input": {
    "text": "{\"city\":\"Melbourne\"}",
    "sha256": "sha256:..."
  },
  "output": {
    "text": "{\"temp_c\":24}",
    "sha256": "sha256:..."
  }
}
```

### `ai.chain.run`

```json
{
  "chain_name": "support_triage_chain",
  "input_keys": ["user_query"],
  "output_keys": ["answer", "confidence"],
  "step_count": 4
}
```

## Privacy Modes

Integrations MUST support privacy mode as an explicit option:

- `off`: raw text values are included.
- `hash`: sensitive text is replaced with deterministic `sha256:<hash>`.
- `redact`: sensitive text is replaced with `[REDACTED]`.

Notes:

- `prompt.text`, `completion.text`, tool input/output text are privacy-filtered.
- `*.sha256` values are computed from emitted text (post-privacy transform) for deterministic verification.
- PII category guidance follows Phase 4 GDPR policy (`email`, `ip`, `user_id`, `payment`, `health`, `bio`, third-party IDs).

## Streaming and Async Behavior

For streaming LLM output:

1. Emit `ai.llm.call` with `phase=start`.
2. Emit `ai.llm.call` with `phase=delta` for each token or chunk.
3. Emit `ai.llm.call` with `phase=end` after completion, including aggregate text and token usage if available.
4. Emit `phase=error` if callback flow errors.

Ordering guarantees:

- Preserve callback emission order in the bundle.
- A single LLM run must follow `start -> delta* -> end|error` for the same `trace_id`/`span_id`.

## JSON Examples

### LLM Start

```json
{
  "type": "ai.llm.call",
  "data": {
    "trace_id": "trc_abc",
    "span_id": "spn_llm_1",
    "framework": "langchain",
    "phase": "start",
    "context": {
      "provider": "openai",
      "model": "gpt-4.1-mini",
      "prompt": {
        "text": "Summarize this ticket",
        "sha256": "sha256:..."
      }
    },
    "privacy": {
      "mode": "hash",
      "redaction_enabled": true,
      "pii_ruleset": "phase4-gdpr-v1"
    },
    "timing": {
      "started_at": "2026-03-09T09:15:02Z",
      "ended_at": null,
      "latency_ms": null
    },
    "status": {
      "ok": true,
      "error": null
    }
  }
}
```

### Tool End

```json
{
  "type": "ai.tool.exec",
  "data": {
    "trace_id": "trc_abc",
    "span_id": "spn_tool_2",
    "parent_span_id": "spn_llm_1",
    "framework": "langchain",
    "phase": "end",
    "context": {
      "tool_name": "crm.lookup",
      "input": {
        "sha256": "sha256:..."
      },
      "output": {
        "sha256": "sha256:..."
      }
    },
    "privacy": {
      "mode": "redact",
      "redaction_enabled": true,
      "pii_ruleset": "phase4-gdpr-v1"
    },
    "timing": {
      "started_at": "2026-03-09T09:15:05Z",
      "ended_at": "2026-03-09T09:15:06Z",
      "latency_ms": 912
    },
    "status": {
      "ok": true,
      "error": null
    }
  }
}
```

### Chain Start

```json
{
  "type": "ai.chain.run",
  "data": {
    "trace_id": "trc_abc",
    "span_id": "spn_chain_0",
    "framework": "langchain",
    "phase": "start",
    "context": {
      "chain_name": "support_triage_chain",
      "input_keys": ["user_query"]
    },
    "privacy": {
      "mode": "off",
      "redaction_enabled": false,
      "pii_ruleset": "phase4-gdpr-v1"
    },
    "timing": {
      "started_at": "2026-03-09T09:15:01Z",
      "ended_at": null,
      "latency_ms": null
    },
    "status": {
      "ok": true,
      "error": null
    }
  }
}
```
