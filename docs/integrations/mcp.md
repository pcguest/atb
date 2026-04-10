# MCP Integration

## Overview

Model Context Protocol, or MCP, is a protocol for agent hosts to invoke tools through a stable interface. ATB integrates with that execution path so MCP tool activity can be recorded as structured bundle events rather than left to mutable application logs. A tamper-evident bundle gives an auditor a hash-chained record that can be verified locally and exported deterministically. A plain activity log can show what was written, but it does not show whether the sequence was altered afterwards.

ATB includes a native MCP stdio server via `atb mcp serve`. It exposes
bundle initialisation, verification, status, and PageIndex RAG
recording tools.

## How it works

An MCP host invokes a tool through the MCP boundary. The ATB CLI or an SDK wrapper appends a structured event for the request, model call, tool call, or response. Each appended record is written into the bundle as NDJSON and linked to the previous record by hash. The resulting bundle can then be checked with `atb verify` and summarised with `atb trust-report`.

```text
     MCP Host
       └─► Tool Call
             └─► ATB SDK (atb append / SDK client)
                   └─► Bundle (.ndjson, hash-chained)
                             └─► atb verify  →  VerifierReport
                                 atb trust-report → markdown / JSON
```

## Quickstart (Go CLI)

```bash
atb init

atb append ai.request.received --data \
  '{"request_id":"req-001","actor_id_hash":"sha256-actor-abc",
    "purpose_tag":"rag_answer"}'

atb append ai.model.invoked --data \
  '{"model_provider":"openai","model_id":"gpt-4o",
    "model_parameters_digest":"sha256-params-def",
    "prompt_digest":"sha256-prompt-ghi"}'

atb verify --format json
```

example output:

```json
{
  "bundle_path": "run.atb/bundle.atb",
  "profile_id": "atb.profile.rag_answer",
  "pass": false,
  "cas_score": 0.535,
  "cas_grade": "Low",
  "sub_scores": {
    "AC": 0,
    "EC": 0.75,
    "FC": 1,
    "GC": 0.3,
    "RC": 0,
    "SC": 0.45,
    "TC": 1,
    "XC": 0
  },
  "critical_failures": [
    {
      "kind": "missing_event",
      "detail": "ai.model.output missing required fields"
    }
  ],
  "required_warnings": [
    "ai.policy.decision recommended for recorded authorisation context",
    "ai.response.sent recommended for response delivery evidence",
    "ai.retrieval.executed recommended for RAG answer provenance"
  ],
  "informational_notes": [
    "Does not prove retrieval completeness beyond recorded corpus/version.",
    "Does not prove the model produced output exactly as provider executed internally; only that recorded invocation/output digests match the bundle."
  ],
  "residual_risk": "High"
}
```

## SDK usage

The current Python and TypeScript SDKs do not provide MCP-specific helpers. Use the generic `Bundle.append(...)` path and emit the canonical event types directly.

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

## What the output looks like

When a reviewer runs `atb verify --format json` on a bundle captured from an MCP workflow, they see the matched profile, whether it passed, any critical failures or required warnings, the CAS grade as `High`, `Medium`, `Low`, or `Insufficient`, and the ResidualRisk level as `Low`, `Medium`, or `High`. When they run `atb trust-report --format markdown`, they see the same evidence summarised as audit sections with bundle integrity, profile checks, and supporting context rendered for review.

## Compliance note

Bundles produced through MCP wrapping are designed to assist with evidence collection and support documentation for EU AI Act Article 12 logging obligations for high-risk AI systems, NIST AI RMF GOVERN and MANAGE function documentation, and the UK DSIT AI Code of Practice transparency and explainability duty; this approach supports review and is designed to assist with those obligations, but formal compliance assessment still depends on system design, deployment controls, and operating evidence outside the bundle.

## Limitations and roadmap

- Recording arbitrary external tool activity still requires explicit ATB
  instrumentation or an MCP host that calls ATB's tools. The server
  does not auto-instrument third-party MCP servers.
- Auto-detection matches on `ai.request.received` or
  `ai.model.invoked`. Bundles that emit only custom event types may not
  match a built-in profile automatically and will require `--profile`.
- `rag_index_record` and `rag_retrieval_record` record PageIndex
  indexing and retrieval evidence. They do not record downstream model
  answer provenance unless the surrounding workflow emits those events
  separately.
