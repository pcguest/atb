"""Contract tests for atb.capture.parse_chatlog."""

from __future__ import annotations

import io

import pytest

from atb.capture import (
    DEFAULT_MAX_INPUT_BYTES,
    InputTooLargeError,
    MalformedChatlogError,
    UnsupportedProviderError,
    parse_chatlog,
)


VALID_LINE = (
    '{"role":"user","content":"hello","timestamp":"2026-04-26T12:00:00Z"}\n'
)


def test_parse_chatlog_unsupported_provider_raises():
    with pytest.raises(UnsupportedProviderError):
        parse_chatlog("claude-desktop", VALID_LINE)
    with pytest.raises(UnsupportedProviderError):
        parse_chatlog("not-a-real-provider", VALID_LINE)


def test_parse_chatlog_malformed_raises():
    payload = (
        VALID_LINE
        + '{"role":"assistant","content": this is not valid json\n'
    )
    with pytest.raises(MalformedChatlogError):
        parse_chatlog("generic-jsonl", payload.encode("utf-8"))


def test_parse_chatlog_happy_path_generic_jsonl():
    messages = parse_chatlog("generic-jsonl", VALID_LINE)
    assert len(messages) == 1
    assert messages[0].role == "user"
    assert messages[0].content == "hello"
    assert messages[0].timestamp == "2026-04-26T12:00:00Z"


def test_max_input_size_enforced():
    cap = 64
    oversized = ("x" * (cap + 1)).encode("utf-8")
    with pytest.raises(InputTooLargeError):
        parse_chatlog("generic-jsonl", oversized, max_bytes=cap)


def test_max_input_size_enforced_for_file_like():
    cap = 64
    oversized = io.BytesIO(("x" * (cap + 1)).encode("utf-8"))
    with pytest.raises(InputTooLargeError):
        parse_chatlog("generic-jsonl", oversized, max_bytes=cap)


def test_default_max_input_bytes_matches_go_cli():
    # 256 MiB, per internal/capture and cmd/atb/import.
    assert DEFAULT_MAX_INPUT_BYTES == 256 * 1024 * 1024
