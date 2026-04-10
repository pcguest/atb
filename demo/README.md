# ATB x PageIndex - Demo

End-to-end demonstration of ATB audit trail recording with PageIndex reasoning-based retrieval.

## Requirements

- Python 3.11+
- `atb` CLI on PATH (`go install github.com/pcguest/atb/cmd/atb@latest`)
- OpenAI API key (`export OPENAI_API_KEY=sk-...`)
- PageIndex installed (`pip install pageindex`)

## Run

From the repository root:

```bash
python demo/pageindex_demo.py --doc demo/sample.md --query "What is ATB?"
```

## What It Does

1. Initialises an ATB bundle in `./run.atb/`
2. Builds a PageIndex tree over `sample.md`
3. Records `atb.event.rag_index` in the bundle
4. Retrieves the best-matching node for your query
5. Records `atb.event.rag_retrieval` in the bundle
6. Runs `atb verify` to prove the chain is intact

The full provenance chain - document ingest, retrieval decision, and verification - is cryptographically linked in a single portable bundle.

## Inspect The Bundle

```bash
atb inspect
atb verify --json
```
