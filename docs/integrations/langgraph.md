# LangGraph integration

ATB can record canonical events from a minimal [LangGraph](https://langchain-ai.github.io/langgraph/) workflow using the Python SDK. Each graph node appends events to a local `.atb` bundle; `atb verify --profile atb.profile.rag_answer` checks the result.

## Install

```bash
pip install -e 'sdk/python[langgraph]'
```

The base `atb-sdk` package is required. LangGraph is optional via the `langgraph` extra.

## Quick start

```bash
python examples/python/langgraph_demo.py
```

The demo compiles a three-node graph (`retrieve` → `generate` → `tool_exec`) with stubbed logic — no network or API keys. Events emitted:

| Node | Events |
|------|--------|
| retrieve | `ai.request.received`, `ai.retrieval.executed` |
| generate | `ai.model.invoked`, `ai.model.output` |
| tool_exec | `ai.tool.exec`, `ai.response.sent` |

Verification uses `atb.profile.rag_answer`.

## Wiring your own graph

```python
from atb import Bundle
from langgraph.graph import END, START, StateGraph

bundle = Bundle()

def my_node(state):
    bundle.append("ai.tool.exec", {"tool_name": "search", "context": {"tool_name": "search"}})
    return state

graph = StateGraph(dict)
graph.add_node("search", my_node)
graph.add_edge(START, "search")
graph.add_edge("search", END)
graph.compile().invoke({})
bundle.save("run.atb/bundle.atb")
```

Use `Bundle.append` (or profile workflow helpers) inside nodes so every step is hash-chained. Call `atb verify --profile <id> --format json` after the graph completes.

## Privacy

LangGraph does not redact payloads automatically. Hash or redact sensitive fields before appending, or use the LangChain callback (`docs/integrations/langchain.md`) when wrapping LLM providers.

## Related

- [LangChain integration](./langchain.md) — callback handler for LangChain runtimes
- [Profiles](../profiles.md) — obligation templates including `atb.profile.rag_answer`
