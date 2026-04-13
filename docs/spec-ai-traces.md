# ATB AI trace specification (current AI trace specification)

This specification defines how framework runtime callbacks map into ATB events without changing the core ATB bundle record format in `docs/spec-v1.0.md`.

This document supersedes the legacy `langchain.*` taxonomy in `docs/spec-v1.0.md` for new integrations. Legacy types remain valid only for backward-compatible verification of older bundles.

Integrations MUST also populate the top-level canonical event fields `timestamp`, `trace_id`, `span_id`, and optional `parent_span_id`. The same trace identifiers are repeated inside `event.data` so the envelope remains self-contained when readers only inspect payloads.

## Goals

- Keep integrations low-friction and opt-in.
- Keep traces machine-parseable and human-auditable.
- Keep privacy handling explicit and deterministic.
- Keep output compatible with existing ATB verification and hashing.

## Event mapping

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

## Canonical `event.data` envelope

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

## Context payloads by type

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

## Privacy modes

Integrations MUST support privacy mode as an explicit option:

- `off`: raw text values are included.
- `hash`: sensitive text is replaced with deterministic `sha256:<hash>`.
- `redact`: sensitive text is replaced with `[REDACTED]`.

Notes:

- `prompt.text`, `completion.text`, tool input/output text are privacy-filtered.
- `*.sha256` values are computed from emitted text (post-privacy transform) for deterministic verification.
- PII category guidance follows the prior specification GDPR policy (`email`, `ip`, `user_id`, `payment`, `health`, `bio`, third-party IDs).

## Streaming and async behaviour

For streaming LLM output:

1. Emit `ai.llm.call` with `phase=start`.
2. Emit `ai.llm.call` with `phase=delta` for each token or chunk.
3. Emit `ai.llm.call` with `phase=end` after completion, including aggregate text and token usage if available.
4. Emit `phase=error` if callback flow errors.

Ordering guarantees:

- Preserve callback emission order in the bundle.
- A single LLM run must follow `start -> delta* -> end|error` for the same `trace_id`/`span_id`.

## JSON examples

### LLM start

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

### Tool end

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

### Chain start

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

## Complete event type registry

The table below lists every canonical ATB event type. The three integration events (`ai.llm.call`, `ai.tool.exec`, `ai.chain.run`) are documented in detail above. The remaining types are used directly via the CLI or SDKs without a framework callback mapping.

Developer-only types (`dev.session`, `snapshot.build`) are used internally by tooling and tests and are not intended for operator use.

| Event type | Category | Criticality | Primary profiles |
| --- | --- | --- | --- |
| `atb.bundle.manifest` | Bundle lifecycle | critical | All profiles |
| `atb.bundle.anchor` | Bundle lifecycle | required | All profiles |
| `atb.bundle.signature` | Bundle lifecycle | required | All profiles |
| `atb.snapshot` | Bundle lifecycle | informational | — |
| `ai.request.received` | AI request/response | critical | `rag_answer`, `privileged_tool_action`, `data_export`, `policy_decision`, `human_override` |
| `ai.response.sent` | AI request/response | required | `rag_answer` |
| `ai.llm.call` | AI integration (see above) | informational | — |
| `ai.tool.exec` | AI integration (see above) | informational | — |
| `ai.chain.run` | AI integration (see above) | informational | — |
| `ai.policy.decision` | Policy | critical | `privileged_tool_action`, `rag_answer`, `data_export`, `policy_decision` |
| `ai.retrieval.executed` | RAG | required | `rag_answer` |
| `ai.model.invoked` | RAG | critical | `rag_answer` |
| `ai.model.output` | RAG | critical | `rag_answer` |
| `atb.event.rag_index` | RAG (PageIndex) | required | `rag_answer` |
| `atb.event.rag_retrieval` | RAG (PageIndex) | required | `rag_answer` |
| `ai.action.precommit` | Privileged action | critical | `privileged_tool_action`, `data_export`, `policy_decision`, `human_override` |
| `ai.action.executed` | Privileged action | critical | `privileged_tool_action`, `data_export`, `human_override` |
| `ai.action.committed` | Privileged action | critical | `privileged_tool_action`, `data_export`, `human_override` |
| `ai.human.approval` | Human oversight | required | `privileged_tool_action`, `data_export`, `human_override` |
| `ai.override.requested` | Human oversight | critical | — |
| `ai.job.scheduled` | Background automation | critical | `background_automation` |
| `ai.job.started` | Background automation | critical | `background_automation` |
| `ai.job.step` | Background automation | critical | `background_automation` |
| `ai.job.completed` | Background automation | critical | `background_automation` |
| `data.export.precommit` | Data export | critical | `data_export` |
| `data.export.executed` | Data export | critical | `data_export` |
| `dev.session` | Developer tooling | informational | — |
| `snapshot.build` | Developer tooling | informational | — |
