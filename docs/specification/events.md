# ATB AI trace specification (current AI trace specification)

This specification defines how framework runtime callbacks map into ATB events without changing the core ATB bundle record format in `docs/specification/bundle-v1.md`.

This document supersedes the legacy `langchain.*` taxonomy in `docs/specification/bundle-v1.md` for new integrations. Legacy types remain valid only for backward-compatible verification of older bundles.

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

## Corroboration events

The `atb.corroboration.*` namespace holds events that record evidence retrieved from external
systems. These events do not change the hash-chain structure and are subject to the same
append-only integrity guarantees as any other event type.

### `atb.corroboration.external`

Records evidence retrieved by a corroboration adapter from an external source — a queue
dequeue receipt, a gateway execution log, a storage confirmation, or any JSON-returning
endpoint that can corroborate a locally recorded event.

A bundle that contains one or more valid `atb.corroboration.external` events earns XC
sub-score credit in CAS evaluation. XC is a diagnostic signal: it records that corroboration
was attempted, not that the external system's record is authoritative or complete.

**Required fields (schema-locked at v1):**

| Field | Type | Description |
| --- | --- | --- |
| `source` | string | Identifies the external system. Examples: `"sqs"`, `"s3"`, `"http-gateway"`, `"manual"`. |
| `reference_id` | string | The hash of the bundle record being corroborated (the record's hash string from `atb append` output). |
| `digest` | string | SHA-256 hex of the external evidence payload retrieved by the adapter. |
| `retrieved_at` | string | RFC 3339 timestamp of when the external evidence was retrieved. |

**Optional fields:**

| Field | Type | Description |
| --- | --- | --- |
| `adapter` | string | The adapter type that produced this record (e.g. `"http-gateway"`). |
| `raw_evidence` | string | Base64-encoded raw payload from the external source. Capped at 4 KB; if the payload exceeds 4 KB the field is omitted and `truncated` is set to `true`. |
| `truncated` | bool | `true` when `raw_evidence` was omitted because the payload exceeded 4 KB. |

**Trust note:** `atb.corroboration.external` events are as trustworthy as the adapter and
the external system that produced them. A compromised gateway can produce valid-looking
corroboration records. ATB records that the adapter retrieved something with the recorded
digest; it does not verify the external system's own integrity.

## Session, oversight, and tool events

The following event types support session lifecycle, human oversight, tool
invocation, and data export workflows. ATB records and verifies these events;
it does not certify legal or regulatory compliance.

Each subsection lists the required `event.data` fields. Implementations MAY
emit additional payload fields; readers MUST ignore unknown fields.

Classification:

- **proxy-internal** — emitted by `atb intercept` only. SDKs SHOULD NOT emit
  these types directly; they describe capture-side lifecycle that the proxy
  owns.
- **canonical (SDK-emittable)** — declared canonical types that SDKs and
  framework integrations MAY emit when the corresponding workflow occurs.

### Reviewer identity evidence

The oversight/action events `ai.policy.decision`, `ai.action.precommit`,
`ai.action.executed`, `ai.action.committed`, `ai.human.approval`,
`atb.human.approval`, and `atb.human.override` MAY carry an
`identity_evidence` object.

| Field | Required | Description |
| --- | --- | --- |
| `identity_provider` | yes | IdP, issuer, CA, or deployment identity-system identifier. |
| `subject` | yes | Subject identifier asserted by the external identity layer. |
| `assertion_type` | yes | `jwt`, `saml`, `x509`, or `opaque`. |
| `assertion_digest` | yes | Digest of the separately retained identity assertion. |
| `auth_context` | no | Authentication method or assurance context, such as MFA. |
| `raw_evidence_digest` | no | Digest of separately retained raw evidence. |

Writers SHOULD NOT place bearer assertions or private credentials in this
object. ATB preserves the supplied evidence in the hash chain but does not
validate the assertion.

### Retention operation events

The following additive, unprofiled event types are normally written to the
separate `.atb/operations.atb` bundle:

- `data.retention.policy_set` records initial local retention configuration.
- `data.retention.policy_changed` links replacement and previous policy
  digests.
- `data.retention.enforced` records a completed local archive operation or an
  accepted remote retention request.

`data.retention.enforced.independently_verified` is required. S3 Object Lock
requests emitted by `atb push` use `false`: a successful PUT proves API
acceptance of the request, not continuing bucket-side enforcement.

### Accountability events from capture

Beyond the capture lifecycle types, `atb intercept` derives accountability
events from the traffic it observes:

- **`atb.tool.call`** — emitted for each tool/function call a model requests in
  a response (Anthropic `tool_use`, OpenAI Chat `tool_calls`, OpenAI Responses
  `function_call`). Records `tool_name` and a `tool_input_digest` (SHA-256 of
  the arguments); raw arguments are never stored. Feeds the session-index
  `tool_without_approval` oversight signal.
- **`ai.action.error`** — emitted for each failed tool result a client reports
  back (Anthropic `tool_result` with `is_error: true`), with `action_id` set to
  the originating `tool_use_id`, `error_class` `failed`, and an
  `error_detail_digest`. OpenAI tool messages carry no standard error flag, so
  they are not classified as failures.

### `atb.session.close`

Classification: proxy-internal. Emitted by `atb intercept` on session
teardown; triggers Mortise auto-push when `MortiseEndpoint` is configured (see
`internal/proxy/recorder.go`).

**Required fields:**

| Field | Type | Description |
| --- | --- | --- |
| `session_id` | string | Stable identifier of the closed capture session. |
| `actor_id` | string | Canonical actor reference for the session; empty string when unresolved. |

#### Always-emitted fields

The proxy always writes these on close, even when a value is zero or empty, so
consumers can rely on their presence:

| Field | Type | Notes |
| --- | --- | --- |
| `model` | string | Last model observed on the session; empty string when none seen. |
| `exchange_count` | int | Number of request/response exchanges captured in the session. |
| `total_tokens` | int | Summed prompt and output tokens across the session, when reported by the model. |
| `closed_at` | string | RFC 3339 timestamp the session was closed. |

### `atb.exchange.complete`

Classification: proxy-internal. Emitted by `atb intercept` at the close of
one request/response exchange within a session. Consumed by the session
index to count exchanges per session.

**Required fields:**

| Field | Type | Description |
| --- | --- | --- |
| `session_id` | string | Session identifier the exchange belongs to. |
| `exchange_id` | string | Stable identifier of the completed exchange. |
| `request_event_id` | string | Hash of the originating proxy request record. |

#### Always-emitted fields

The proxy always writes these, even when a value is zero or empty, so consumers
can rely on their presence:

| Field | Type | Notes |
| --- | --- | --- |
| `actor_id` | string | Resolved actor identity; empty string when unresolvable. |
| `completed_at` | string | RFC 3339 timestamp of response completion. |
| `tool_calls_count` | int | Tool/function invocations counted in the response body (`tool_use` for Anthropic, `tool_calls`/`function_call` for OpenAI); `0` when none or when the body is not recognised. |

#### Optional fields (emitted when known)

| Field | Type | Notes |
| --- | --- | --- |
| `model` | string | Model identifier from the proxied response. |
| `input_tokens` | int | Token count from the request, when reported by the model. |
| `output_tokens` | int | Token count from the response, when reported by the model. |
| `latency_ms` | int | Wall-clock milliseconds from request arrival to response flush. |

> All fields beyond the required set are additive under the manifest v1
> contract. Consumers MUST NOT require them; producers SHOULD emit them when
> the information is available without additional I/O. `tool_calls_count` is
> best-effort: it reflects only what is parseable from the response body and
> never blocks recording.

### `atb.tool.call`

Classification: canonical (SDK-emittable). Records that an AI agent invoked
a tool. Triggers the `tool_without_approval` anomaly flag in the session
index when no preceding `atb.human.approval` event exists in the same
session.

**Required fields:**

| Field | Type | Description |
| --- | --- | --- |
| `session_id` | string | Session identifier the tool call belongs to. |
| `tool_name` | string | Identifier of the invoked tool. |

**Optional fields (emitted when known):**

| Field | Type | Description |
| --- | --- | --- |
| `actor_id` | string | Actor reference for the invocation. |
| `tool_input_digest` | string | SHA-256 hex of the canonicalised tool input. |
| `tool_output_digest` | string | SHA-256 hex of the canonicalised tool output. |

### `atb.data.export`

Classification: canonical (SDK-emittable). Records that an AI agent exported
data outside the session boundary.

**Required fields:**

| Field | Type | Description |
| --- | --- | --- |
| `session_id` | string | Session identifier the export belongs to. |
| `export_target` | string | Identifier of the export destination (URI, system, or sink name). |

**Optional fields (emitted when known):**

| Field | Type | Description |
| --- | --- | --- |
| `actor_id` | string | Actor reference for the export. |
| `record_count` | integer | Number of records exported; omitted when zero or unknown. |
| `classification` | string | Data classification label. |

### `atb.human.override`

Classification: canonical (SDK-emittable). Records that a human operator
overrode an AI-recommended action.

**Required fields:**

| Field | Type | Description |
| --- | --- | --- |
| `session_id` | string | Session identifier the override belongs to. |
| `override_reason` | string | Short rationale for the override. |

**Optional fields (emitted when known):**

| Field | Type | Description |
| --- | --- | --- |
| `actor_id` | string | Canonical identifier of the human operator. |
| `overridden_action_id` | string | Identifier of the overridden action. |

### `atb.human.approval`

Classification: canonical (SDK-emittable). Records that a human operator
explicitly approved a pending action. When recorded in sequence before a
corresponding `atb.tool.call` in the same session, it closes the
`tool_without_approval` anomaly flag.

**Required fields:**

| Field | Type | Description |
| --- | --- | --- |
| `session_id` | string | Session identifier the approval belongs to. |
| `approved_action_id` | string | Identifier of the action being approved. |

**Optional fields (emitted when known):**

| Field | Type | Description |
| --- | --- | --- |
| `actor_id` | string | Canonical identifier of the approving human operator. |
| `approver_id` | string | Approver reference. |
| `note` | string | Free-text approval note. |

## Complete event type registry

The table below lists every canonical ATB event type. The three integration events (`ai.llm.call`, `ai.tool.exec`, `ai.chain.run`) are documented in detail above. The remaining types are used directly via the CLI or SDKs without a framework callback mapping.

Developer-only types (`dev.session`) are used internally by tooling and tests and are not intended for operator use.

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
| `ai.action.error` | Privileged action | required | — |
| `ai.human.approval` | Human oversight | required | `privileged_tool_action`, `data_export`, `human_override` |
| `ai.job.scheduled` | Background automation | critical | `background_automation` |
| `ai.job.started` | Background automation | critical | `background_automation` |
| `ai.job.step` | Background automation | required | `background_automation` |
| `ai.job.completed` | Background automation | critical | `background_automation` |
| `data.export.precommit` | Data export | critical | `data_export` |
| `data.export.executed` | Data export | critical | `data_export` |
| `data.export.error` | Data export | required | — |
| `atb.capture.scope` | Capture (proxy-internal) | required | — |
| `atb.capture.rejected` | Capture (proxy-internal) | required | — |
| `atb.llm.request` | Capture (proxy-internal) | informational | — |
| `atb.llm.response` | Capture (proxy-internal) | informational | — |
| `atb.session.close` | Session lifecycle | informational | — |
| `atb.exchange.complete` | Session lifecycle | informational | — |
| `atb.tool.call` | Tool invocation | required | — |
| `atb.data.export` | Data export | required | — |
| `atb.human.override` | Human oversight | required | — |
| `atb.human.approval` | Human oversight | required | — |
| `dev.session` | Developer tooling | informational | — |
| `atb.corroboration.external` | Corroboration | informational | All (contributes to XC) |
