# ATB Python SDK

The official Python SDK for [ATB (Agent Trace Bundle)](https://github.com/pcguest/atb) - tamper-evident, replayable audit trails for AI agent workflows.

## Installation

```bash
pip install atb-sdk
```

The package does not include a standalone ATB CLI. The installed `atb` command is a compatibility stub that prints Go CLI install guidance and will be removed in a future major release.

With LangChain integration:

```bash
pip install atb-sdk[langchain]
```

## Quick Start

```python
from atb import Bundle

# Create a new bundle
bundle = Bundle()

# Append events
bundle.append("dev.session", {
    "date": "2025-01-15",
    "features_built": ["hash chaining", "CLI init"],
    "blockers": ["RFC 8785 library compatibility"],
})

bundle.append("decision", {
    "choice": "Go over Rust for CLI",
    "reason": "Solo founder velocity",
    "alternatives": ["Rust", "Python-only"],
})

# Save to disk
bundle.save("run.atb/bundle.atb")

# Later - reload and verify integrity
b = Bundle.load("run.atb/bundle.atb")
b.verify()  # Raises ATBVerificationError if tampered
print(f"Verified {len(b)} events - chain intact.")
```

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

## Licence

MIT
