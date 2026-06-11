from __future__ import annotations

import pytest

from atb import Bundle
from atb.sdk_capture import wrap_anthropic, wrap_openai


def _user_records(bundle: Bundle) -> list:
    return [
        record
        for record in bundle.records
        if record.event["type"] != "atb.bundle.manifest"
    ]


def _types(bundle: Bundle) -> list[str]:
    return [record.event["type"] for record in _user_records(bundle)]


# ---------------------------------------------------------------------------
# OpenAI
# ---------------------------------------------------------------------------


def test_wrap_openai_records_request_model_and_output() -> None:
    bundle = Bundle()

    def fake_create(**_: object) -> dict:
        return {
            "model": "gpt-4o",
            "choices": [
                {"message": {"content": "hello back"}, "finish_reason": "stop"}
            ],
            "usage": {"prompt_tokens": 5, "completion_tokens": 2, "total_tokens": 7},
        }

    create = wrap_openai(fake_create, bundle=bundle, privacy_mode="off")
    res = create(model="gpt-4o", messages=[{"role": "user", "content": "hi"}])
    assert res["choices"][0]["message"]["content"] == "hello back"

    assert _types(bundle) == [
        "atb.capture.scope",
        "ai.llm.call",
        "ai.request.received",
        "ai.model.invoked",
        "ai.llm.call",
        "ai.model.output",
    ]

    scope = _user_records(bundle)[0].event["data"]
    assert scope["targets"] == ["openai"]
    assert scope["capture_mode"] == "raw"

    invoked = _user_records(bundle)[3].event["data"]
    assert invoked["model_provider"] == "openai"
    assert invoked["model_id"] == "gpt-4o"

    end = _user_records(bundle)[4].event["data"]
    assert end["context"]["completion"]["text"] == "hello back"
    assert end["context"]["token_usage"]["total_tokens"] == 7


def test_wrap_openai_records_tool_calls() -> None:
    bundle = Bundle()

    def fake_create(**_: object) -> dict:
        return {
            "choices": [
                {
                    "message": {
                        "content": "",
                        "tool_calls": [
                            {
                                "function": {
                                    "name": "weather.lookup",
                                    "arguments": '{"city":"Melbourne"}',
                                }
                            }
                        ],
                    },
                    "finish_reason": "tool_calls",
                }
            ]
        }

    create = wrap_openai(fake_create, bundle=bundle)
    create(model="gpt-4o", messages=[{"role": "user", "content": "weather?"}])

    assert "ai.tool.exec" in _types(bundle)
    tool = next(
        r for r in _user_records(bundle) if r.event["type"] == "ai.tool.exec"
    ).event["data"]
    assert tool["context"]["tool_name"] == "weather.lookup"


def test_wrap_openai_records_error_and_reraises() -> None:
    bundle = Bundle()

    def fake_create(**_: object) -> dict:
        raise RuntimeError("rate limited")

    create = wrap_openai(fake_create, bundle=bundle)
    with pytest.raises(RuntimeError, match="rate limited"):
        create(model="gpt-4o", messages=[])

    error = next(
        r
        for r in _user_records(bundle)
        if r.event["data"].get("status", {}).get("ok") is False
    )
    assert error.event["data"]["status"]["error"]["type"] == "RuntimeError"


def test_wrap_openai_hash_privacy_mode() -> None:
    bundle = Bundle()

    def fake_create(**_: object) -> dict:
        return {"choices": [{"message": {"content": "x"}}]}

    create = wrap_openai(fake_create, bundle=bundle, privacy_mode="hash")
    create(model="gpt-4o", messages=[{"role": "user", "content": "secret"}])

    scope = _user_records(bundle)[0].event["data"]
    assert scope["capture_mode"] == "digest"

    start = _user_records(bundle)[1].event["data"]
    assert start["context"]["prompt"]["text"].startswith("sha256:")
    assert "secret" not in start["context"]["prompt"]["text"]


def test_wrap_openai_disabled_passes_through_without_recording() -> None:
    bundle = Bundle()
    called = {"v": False}

    def fake_create(**_: object) -> dict:
        called["v"] = True
        return {"choices": [{"message": {"content": "y"}}]}

    create = wrap_openai(fake_create, bundle=bundle, enabled=False)
    res = create(model="gpt-4o", messages=[])
    assert called["v"] is True
    assert res["choices"][0]["message"]["content"] == "y"
    assert _user_records(bundle) == []


def test_wrap_openai_streaming_raises() -> None:
    create = wrap_openai(lambda **_: {}, bundle=Bundle())
    with pytest.raises(ValueError, match="streaming"):
        create(model="gpt-4o", messages=[], stream=True)


# ---------------------------------------------------------------------------
# Anthropic (object-shaped response, exercising the attribute accessor)
# ---------------------------------------------------------------------------


class _Block:
    def __init__(self, **kw: object) -> None:
        for key, value in kw.items():
            setattr(self, key, value)


class _Usage:
    def __init__(self, input_tokens: int, output_tokens: int) -> None:
        self.input_tokens = input_tokens
        self.output_tokens = output_tokens


class _Message:
    def __init__(self) -> None:
        self.model = "claude-sonnet-4-6"
        self.content = [
            _Block(type="text", text="the answer is "),
            _Block(type="text", text="42"),
            _Block(type="tool_use", name="calc", input={"expr": "6*7"}),
        ]
        self.usage = _Usage(10, 4)
        self.stop_reason = "end_turn"


def test_wrap_anthropic_records_text_and_tool_use() -> None:
    bundle = Bundle()
    create = wrap_anthropic(lambda **_: _Message(), bundle=bundle)
    create(
        model="claude-sonnet-4-6",
        max_tokens=256,
        system="be terse",
        messages=[{"role": "user", "content": "what is 6*7"}],
    )

    records = _user_records(bundle)
    invoked = next(r for r in records if r.event["type"] == "ai.model.invoked").event["data"]
    assert invoked["model_provider"] == "anthropic"
    assert invoked["model_id"] == "claude-sonnet-4-6"

    end = [r for r in records if r.event["type"] == "ai.llm.call"][-1].event["data"]
    assert end["context"]["completion"]["text"] == "the answer is 42"
    usage = end["context"]["token_usage"]
    assert usage["prompt_tokens"] == 10
    assert usage["completion_tokens"] == 4
    assert usage["total_tokens"] == 14

    tool = next(r for r in records if r.event["type"] == "ai.tool.exec").event["data"]
    assert tool["context"]["tool_name"] == "calc"


def test_wrap_anthropic_records_error_and_reraises() -> None:
    bundle = Bundle()

    def fake_create(**_: object) -> object:
        raise RuntimeError("overloaded")

    create = wrap_anthropic(fake_create, bundle=bundle)
    with pytest.raises(RuntimeError, match="overloaded"):
        create(model="claude-sonnet-4-6", messages=[])

    error = next(
        r
        for r in _user_records(bundle)
        if r.event["data"].get("status", {}).get("ok") is False
    )
    assert error.event["data"]["status"]["error"]["type"] == "RuntimeError"


def test_wrap_anthropic_streaming_raises() -> None:
    create = wrap_anthropic(lambda **_: object(), bundle=Bundle())
    with pytest.raises(ValueError, match="streaming"):
        create(model="claude-sonnet-4-6", messages=[], stream=True)
