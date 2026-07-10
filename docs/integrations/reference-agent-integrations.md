# Reference agent integrations

This page names the supported reference paths for evaluating ATB with AI-agent
workflows. It separates deterministic local evidence from examples that require
live provider credentials.

## Recommended evaluator path

Use the Tenon pilot fixture for the first end-to-end result: it creates a
representative AI-agent session with an approved action, an anomalous privileged
action, local verification, incident reporting, and optional Mortise custody.

For framework-specific integration work, start with the fake-provider tests
below. They exercise ATB capture semantics without network calls or provider
credentials.

## No external credentials

| Path | Runtime | What it proves | Validation |
| --- | --- | --- | --- |
| `sdk/python/tests/test_sdk_capture.py` | Python SDK | Direct OpenAI and Anthropic wrappers record request, model, output, tool-call, privacy-mode, disabled, and error paths against fake callables. | `python -m pytest sdk/python/tests/test_sdk_capture.py` |
| `sdk/typescript/src/sdk-capture.test.ts` | TypeScript SDK | Direct OpenAI and Anthropic wrappers record the same fake-provider paths in Node. | `cd sdk/typescript && npm test -- --run sdk-capture` |
| `examples/python/agent_incident_demo.py` | Python SDK + local CLI | Creates an offline denied privileged-action incident bundle and runs verify, trust report, incident list, and incident report. | `ATB_BIN=/path/to/atb python examples/python/agent_incident_demo.py` |
| `examples/python/langgraph_demo.py` | Python SDK + optional LangGraph extra | Demonstrates a deterministic LangGraph-style RAG flow with profile verification. | `pip install -e 'sdk/python[langgraph]' && ATB_BIN=/path/to/atb python examples/python/langgraph_demo.py` |

## Live-provider examples

These examples are useful once the local path is understood, but they should not
be required for CI or first-time evaluation.

| Path | Provider requirement | Notes |
| --- | --- | --- |
| `examples/python/langchain_bot.py` | `OPENAI_API_KEY` and LangChain dependencies | Minimal LangChain callback example. Use the fake-provider SDK tests first when validating ATB behavior. |
| `examples/typescript/vercel-chat-bot.ts` | `OPENAI_API_KEY`, Vercel AI SDK dependencies | Minimal Vercel AI middleware example. Use `sdk/typescript/src/sdk-capture.test.ts` for deterministic capture assertions. |

## Capture and honesty limits

Capture completeness is bounded by the integration boundary. ATB records calls
that pass through the wrapper, callback, import, or proxy configured by the
operator. It cannot prove that an agent did not bypass the integration, that an
operator emitted every relevant event, or that omitted upstream context never
existed. Keep those limitations visible in pilot reports and incident
walkthroughs.
