# Instrumentation checklist

Use this checklist to close **L5 (capture boundary)** gaps from the
[provability ladder](../provability-ladder.md). Tick each item before treating a
bundle as review-ready for the selected profile.

## Before capture

- [ ] Obligation profile selected (`atb.profile.*`) and documented for the workflow
- [ ] SDK or CLI instrumentation covers success, failure, retry, and fallback paths
- [ ] `atb capture run` or in-process `Bundle` writer configured for the session
- [ ] TypeScript: optional `AutomationSession` for chained model/tool hops ([automation harness](automation-harness.md))
- [ ] Capture environment variables set (`ATB_BUNDLE_PATH`, `ATB_CAPTURE_RUN_ID`)
- [ ] Optional: `atb agent run` for a local Agent process ([Agent guide](agent.md)); not required for CLI or SDK capture

## Per profile minimum events

### `atb.profile.policy_decision`

- [ ] `ai.request.received` with `request_id`, `actor_id_hash`, `purpose_tag`
- [ ] `ai.policy.decision` with policy fields and `action_id` when applicable
- [ ] Optional: `policy_signature` when L3 source binding is required

### `atb.profile.privileged_tool_action`

- [ ] Full gate chain: precommit → policy → executed → committed
- [ ] ACP gate enforced at runtime for high-impact tools
- [ ] Human approval when required by action type

### `atb.profile.rag_answer`

- [ ] `ai.model.invoked`, `ai.model.output`, and recommended retrieval/response events
- [ ] Prompt and output digests populated

## After capture

- [ ] Named snapshot at review boundary (`atb snapshot <name>`)
- [ ] `atb verify --profile <id> --format json` — inspect `pass` and `provability_gaps`
- [ ] Optional: `atb anchor`, `atb corroborate`, bundle signature for L3/L4
- [ ] Export with `--with-verify` for handoff packages

## Retrospective import

Chatlog import (`atb import chatlog`) is valid for reconstruction but marks bundles
as retrospective. Prefer live capture for primary evidence.

Supported import formats: `generic-jsonl`, `openai-jsonl`.
