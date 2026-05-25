# Demo workflow bundle

A composite **support escalation** narrative (~20 events) for dashboard demos and Custos pitch rehearsals. It tells a single coherent story rather than padding profile fixtures.

## Story

1. Agent receives a support request (`ai.request.received`).
2. RAG pipeline indexes and retrieves refund policy context (`atb.event.rag_index`, `atb.event.rag_retrieval`, `ai.retrieval.executed`).
3. Model analyses the case and looks up the customer (`ai.model.*`, `ai.tool.exec`).
4. Agent proposes a $250 refund (`ai.action.precommit`); policy **denies** auto-refund (`ai.policy.decision`).
5. Supervisor approves store credit instead (`ai.human.approval` + second precommit/execute/commit cycle).
6. CRM corroboration receipt is recorded (`atb.corroboration.external`) for XC credit.
7. Customer notification and response sent; bundle signed and snapshotted.

## Verification

```bash
atb verify --bundle examples/bundles/demo-workflow/demo-workflow.atb \
  --profile atb.profile.policy_decision --format json

atb verify --bundle examples/bundles/demo-workflow/demo-workflow.atb \
  --profile atb.profile.human_override --format json
```

Both profiles pass. XC sub-score is non-zero thanks to corroboration.

## Viewer

```bash
# Requires embedded UI build (see repo README)
atb view --bundle examples/bundles/demo-workflow/demo-workflow.atb \
  --profile atb.profile.policy_decision
```

## Regenerate

```bash
bash examples/bundles/demo-workflow/generate.sh
```

Requires `atb` on PATH (or set `ATB_BIN`). The generator asserts both profiles pass before signing. Committed `demo-signing-key.pem` is a **demo-only** key — never use in production.
