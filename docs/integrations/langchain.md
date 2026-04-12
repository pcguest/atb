# LangChain integration

ATB provides an opt-in callback middleware for LangChain that emits AI trace events into a standard `run.atb/bundle.atb` bundle.

## Install

```bash
pip install atb-sdk
pip install langchain langchain-openai
```

Use `pip install atb-sdk` as the base package install.

## Quick start

```python
from atb import Bundle
from atb.langchain_callback import ATBCallbackHandler
from langchain_openai import ChatOpenAI

bundle = Bundle()

# privacy_mode: "off" | "hash" | "redact"
callback = ATBCallbackHandler(
    bundle=bundle,
    privacy_mode="redact",
    auto_save=True,
    save_path="run.atb/bundle.atb",
)

llm = ChatOpenAI(model="gpt-4o-mini", callbacks=[callback])
_ = llm.invoke("Summarize this text")
```

## Event mapping

- `on_llm_start` -> `ai.llm.call` (`phase=start`)
- `on_llm_new_token` -> `ai.llm.call` (`phase=delta`)
- `on_llm_end` -> `ai.llm.call` (`phase=end`)
- `on_tool_start` / `on_tool_end` -> `ai.tool.exec`
- `on_chain_start` / `on_chain_end` -> `ai.chain.run`

## Privacy modes

- `off`: plain prompt/completion text is recorded.
- `hash`: text fields are replaced with deterministic SHA-256 values.
- `redact`: text fields are replaced with `[REDACTED]`.

All modes also store `sha256` values for emitted text payloads.

## Disable tracing

Set `enabled=False` to make the handler a no-op without changing app flow:

```python
callback = ATBCallbackHandler(enabled=False)
```
