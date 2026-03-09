from __future__ import annotations

import hashlib
import warnings

from atb import Bundle
from atb.langchain_callback import ATBCallbackHandler
from atb.integrations.langchain import ATBCallbackHandler as ShimATBCallbackHandler


class _Generation:
    def __init__(self, text: str, finish_reason: str = "stop") -> None:
        self.text = text
        self.generation_info = {"finish_reason": finish_reason}


class _Response:
    def __init__(self, text: str) -> None:
        self.generations = [[_Generation(text)]]
        self.llm_output = {
            "token_usage": {
                "prompt_tokens": 3,
                "completion_tokens": 2,
                "total_tokens": 5,
            },
            "finish_reason": "stop",
        }


def test_langchain_callback_emits_chain_and_llm_events() -> None:
    bundle = Bundle()
    handler = ATBCallbackHandler(bundle=bundle, privacy_mode="off")

    handler.on_chain_start({"name": "qa_chain"}, {"query": "what is atb"}, run_id="chain-1")
    handler.on_llm_start(
        {"name": "gpt-4o-mini", "id": ["openai", "chat"]},
        ["hello"],
        run_id="llm-1",
        parent_run_id="chain-1",
        invocation_params={"model": "gpt-4o-mini", "provider": "openai"},
    )
    handler.on_llm_new_token("A", run_id="llm-1")
    handler.on_llm_new_token("B", run_id="llm-1")
    handler.on_llm_end(_Response("ignored because stream exists"), run_id="llm-1")
    handler.on_chain_end({"answer": "AB"}, run_id="chain-1")

    assert [record.event["type"] for record in bundle.records] == [
        "ai.chain.run",
        "ai.llm.call",
        "ai.llm.call",
        "ai.llm.call",
        "ai.llm.call",
        "ai.chain.run",
    ]

    chain_start = bundle.records[0].event["data"]
    llm_start = bundle.records[1].event["data"]
    llm_end = bundle.records[4].event["data"]

    assert chain_start["phase"] == "start"
    assert llm_start["phase"] == "start"
    assert llm_start["trace_id"] == chain_start["trace_id"]
    assert llm_start["parent_span_id"] == chain_start["span_id"]

    assert llm_end["phase"] == "end"
    assert llm_end["context"]["completion"]["text"] == "AB"
    assert llm_end["context"]["token_usage"]["total_tokens"] == 5


def test_langchain_callback_privacy_hash_mode() -> None:
    bundle = Bundle()
    handler = ATBCallbackHandler(bundle=bundle, privacy_mode="hash")

    handler.on_llm_start({"name": "gpt-4o-mini"}, ["private prompt"], run_id="llm-hash")

    payload = bundle.records[0].event["data"]
    prompt = payload["context"]["prompt"]

    assert prompt["text"].startswith("sha256:")
    expected = "sha256:" + hashlib.sha256(prompt["text"].encode("utf-8")).hexdigest()
    assert prompt["sha256"] == expected


def test_langchain_callback_privacy_redact_mode_hashes_emitted_text() -> None:
    bundle = Bundle()
    handler = ATBCallbackHandler(bundle=bundle, privacy_mode="redact")

    handler.on_llm_start({"name": "gpt-4o-mini"}, ["private prompt"], run_id="llm-redact")

    payload = bundle.records[0].event["data"]
    prompt = payload["context"]["prompt"]

    assert prompt["text"] == "[REDACTED]"
    expected = "sha256:" + hashlib.sha256("[REDACTED]".encode("utf-8")).hexdigest()
    assert prompt["sha256"] == expected


def test_langchain_callback_tool_mapping() -> None:
    bundle = Bundle()
    handler = ATBCallbackHandler(bundle=bundle, privacy_mode="off")

    handler.on_tool_start({"name": "weather.lookup"}, '{"city":"Melbourne"}', run_id="tool-1")
    handler.on_tool_end('{"temp_c":24}', run_id="tool-1")

    assert [record.event["type"] for record in bundle.records] == ["ai.tool.exec", "ai.tool.exec"]
    assert bundle.records[0].event["data"]["phase"] == "start"
    assert bundle.records[1].event["data"]["phase"] == "end"


def test_langchain_callback_disabled_mode_noops() -> None:
    bundle = Bundle()
    handler = ATBCallbackHandler(bundle=bundle, enabled=False)

    handler.on_chain_start({"name": "qa_chain"}, {"query": "x"}, run_id="chain-1")
    handler.on_llm_start({"name": "gpt-4o-mini"}, ["hello"], run_id="llm-1")

    assert len(bundle.records) == 0


def test_langchain_shim_emits_deprecation_warning() -> None:
    with warnings.catch_warnings(record=True) as caught:
        warnings.simplefilter("always")
        ShimATBCallbackHandler(bundle=Bundle())

    assert any(isinstance(item.message, DeprecationWarning) for item in caught)
