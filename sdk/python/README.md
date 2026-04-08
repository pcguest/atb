# ATB Python SDK

The official Python SDK for [ATB (Agent Trace Bundle)](https://github.com/pcguest/atb) - local-first, tamper-evident audit trails for AI workflows.

Source repository: [github.com/pcguest/atb](https://github.com/pcguest/atb)

## Installation

```bash
pip install atb-sdk
```

Use this package when you need to write or verify bundles from Python code. The Go CLI remains the authoritative CLI path:

```bash
go install github.com/pcguest/atb/cmd/atb@latest
```

The package does not include a standalone ATB CLI. The installed `atb` command is a compatibility stub that prints Go CLI install guidance and will be removed in a future major release.

With LangChain integration:

```bash
pip install atb-sdk[langchain]
```

`atb-sdk[langchain]` is not yet published to PyPI; use `pip install atb-sdk` plus the LangChain packages directly until that extra is available.

## Quick Start

```python
from atb import Bundle
from atb.event_types import (
    AI_MODEL_INVOKED_EVENT_TYPE,
    AI_MODEL_OUTPUT_EVENT_TYPE,
    AI_REQUEST_RECEIVED_EVENT_TYPE,
)

bundle = Bundle()

bundle.append(AI_REQUEST_RECEIVED_EVENT_TYPE, {
    "request_id": "req-001",
    "actor_id_hash": "sha256-actor-abc",
    "purpose_tag": "rag_answer",
})
bundle.append(AI_MODEL_INVOKED_EVENT_TYPE, {
    "model_provider": "openai",
    "model_id": "gpt-4o",
    "model_parameters_digest": "sha256-params-def",
    "prompt_digest": "sha256-prompt-ghi",
})
bundle.append(AI_MODEL_OUTPUT_EVENT_TYPE, {
    "output_digest": "sha256-output-jkl",
    "output_format": "text/plain",
})

bundle.save("run.atb/bundle.atb")

b = Bundle.load("run.atb/bundle.atb")
b.verify()
print(f"Verified {len(b)} records (including manifest).")
```

`Bundle()` starts with an `atb.bundle.manifest` record at `seq = 0`. Appended events start at `seq = 1`.

## LangChain Integration

```python
from atb import Bundle
from atb.langchain_callback import ATBCallbackHandler
from langchain.chat_models import ChatOpenAI

bundle = Bundle()
handler = ATBCallbackHandler(bundle, auto_save=True)

llm = ChatOpenAI(callbacks=[handler])
# All LLM calls are now automatically recorded in the bundle.
```

The callback emits the canonical `ai.chain.run`, `ai.llm.call`, and `ai.tool.exec` event types and also sets the top-level `timestamp`, `trace_id`, `span_id`, and `parent_span_id` event fields used by the Go runtime.

The deprecated shim import path `atb.integrations.langchain.ATBCallbackHandler` still works for compatibility, but it emits a `DeprecationWarning`. Use `atb.langchain_callback.ATBCallbackHandler` for new code.

## Licence

MIT
