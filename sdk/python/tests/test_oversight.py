"""Tests for canonical atb.* oversight emitters."""

from __future__ import annotations

import pytest

from atb import event_types
from atb.oversight import (
    ActionErrorEmitter,
    DataExportEmitter,
    HumanApprovalEmitter,
    HumanOverrideEmitter,
    ToolCallEmitter,
)


class _StubSink:
    def __init__(self) -> None:
        self.events: list[tuple[str, dict]] = []
        self.session_id = "test-session-1"

    def emit(self, event_type: str, payload: dict) -> None:
        self.events.append((event_type, payload))


def test_tool_call_emitter_missing_required() -> None:
    sink = _StubSink()
    emitter = ToolCallEmitter(sink)
    with pytest.raises(ValueError, match="tool_name"):
        emitter.emit("")
    assert len(sink.events) == 0


def test_tool_call_emitter_valid_emits_correct_type() -> None:
    sink = _StubSink()
    emitter = ToolCallEmitter(sink)
    emitter.emit("web_search", actor_id="alice")
    assert len(sink.events) == 1
    event_type, payload = sink.events[0]
    assert event_type == event_types.TOOL_CALL
    assert payload["tool_name"] == "web_search"
    assert payload["actor_id"] == "alice"


def test_data_export_emitter_valid() -> None:
    sink = _StubSink()
    emitter = DataExportEmitter(sink)
    emitter.emit("csv", record_count=42)
    _, payload = sink.events[0]
    assert payload["export_target"] == "csv"
    assert payload["record_count"] == 42


def test_human_override_emitter_valid() -> None:
    sink = _StubSink()
    emitter = HumanOverrideEmitter(sink)
    emitter.emit("emergency stop", overridden_action_id="act-99")
    _, payload = sink.events[0]
    assert payload["override_reason"] == "emergency stop"
    assert payload["overridden_action_id"] == "act-99"


def test_human_approval_emitter_missing_required() -> None:
    sink = _StubSink()
    emitter = HumanApprovalEmitter(sink)
    with pytest.raises(ValueError, match="approved_action_id"):
        emitter.emit("")
    assert len(sink.events) == 0


def test_human_approval_emitter_valid_emits_correct_type() -> None:
    sink = _StubSink()
    emitter = HumanApprovalEmitter(sink)
    emitter.emit("action-42", approver_id="bob", note="LGTM")
    event_type, payload = sink.events[0]
    assert event_type == event_types.HUMAN_APPROVAL
    assert payload["approved_action_id"] == "action-42"
    assert payload["approver_id"] == "bob"


def test_empty_session_id_propagated() -> None:
    sink = _StubSink()
    emitter = ToolCallEmitter(sink)
    emitter.emit("search")
    _, payload = sink.events[0]
    assert payload.get("session_id") == "test-session-1"


def test_action_error_emitter_missing_required() -> None:
    sink = _StubSink()
    emitter = ActionErrorEmitter(sink)
    with pytest.raises(ValueError, match="action_id"):
        emitter.emit("", "failed")
    with pytest.raises(ValueError, match="error_class"):
        emitter.emit("act-1", "")
    assert len(sink.events) == 0


def test_action_error_emitter_valid_emits_correct_type() -> None:
    sink = _StubSink()
    emitter = ActionErrorEmitter(sink)
    emitter.emit("act-1", "exception", error_detail_digest="sha256:abc")
    assert len(sink.events) == 1
    event_type, payload = sink.events[0]
    assert event_type == event_types.AI_ACTION_ERROR_EVENT_TYPE
    assert payload["action_id"] == "act-1"
    assert payload["error_class"] == "exception"
    assert payload["error_detail_digest"] == "sha256:abc"
    assert payload["session_id"] == "test-session-1"
