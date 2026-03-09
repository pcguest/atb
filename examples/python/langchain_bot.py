"""Minimal LangChain + ATB callback example.

Run:
    export OPENAI_API_KEY=...
    python examples/python/langchain_bot.py
"""

from __future__ import annotations

import os

from atb import Bundle
from atb.langchain_callback import ATBCallbackHandler


def main() -> None:
    api_key = os.getenv("OPENAI_API_KEY")
    if not api_key:
        raise RuntimeError("OPENAI_API_KEY is required")

    from langchain_openai import ChatOpenAI

    bundle = Bundle()

    # Toggle privacy_mode: "off" | "hash" | "redact"
    callback = ATBCallbackHandler(
        bundle=bundle,
        privacy_mode="redact",
        auto_save=True,
        save_path="run.atb/bundle.atb",
    )

    llm = ChatOpenAI(model="gpt-4o-mini", api_key=api_key, callbacks=[callback])
    response = llm.invoke("Summarize this text in one sentence: ATB provides verifiable AI workflow audit trails.")

    print(response.content)
    path = bundle.save("run.atb/bundle.atb")
    print(f"ATB bundle written to: {path}")


if __name__ == "__main__":
    main()
