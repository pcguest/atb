"""Emitters for canonical atb.* oversight event types."""

from __future__ import annotations

from collections.abc import Mapping
from typing import Any

from atb import event_types

__all__ = [
    "ToolCallEmitter",
    "DataExportEmitter",
    "HumanOverrideEmitter",
    "HumanApprovalEmitter",
    "ActionErrorEmitter",
]


def _require_non_empty(value: str, field: str) -> str:
    trimmed = (value or "").strip()
    if not trimmed:
        raise ValueError(f"atb: {field} is required")
    return trimmed


def _dispatch(ctx: Any, event_type: str, payload: Mapping[str, Any]) -> None:
    emit = getattr(ctx, "emit", None)
    if callable(emit):
        emit(event_type, dict(payload))
        return
    append = getattr(ctx, "append", None)
    if callable(append):
        append(event_type, dict(payload))
        return
    raise ValueError("atb: context must provide emit or append")


def _session_id(ctx: Any, kwargs: Mapping[str, Any]) -> str | None:
    override = kwargs.get("session_id")
    if override is not None and str(override).strip():
        return str(override).strip()
    sid = getattr(ctx, "session_id", None)
    if sid is not None and str(sid).strip():
        return str(sid).strip()
    return None


def _base_payload(ctx: Any, kwargs: Mapping[str, Any]) -> dict[str, Any]:
    payload: dict[str, Any] = {}
    sid = _session_id(ctx, kwargs)
    if sid:
        payload["session_id"] = sid
    return payload


def _optional_str(payload: dict[str, Any], kwargs: Mapping[str, Any], key: str) -> None:
    value = kwargs.get(key)
    if value is not None and str(value).strip():
        payload[key] = str(value).strip()


class ToolCallEmitter:
    """Emits atb.tool.call events for direct tool invocations.

    Attach to a WorkflowContext (or compatible event sink) and call emit() once
    per tool invocation that should be recorded in the audit trail.
    """

    def __init__(self, ctx: Any) -> None:
        if not (callable(getattr(ctx, "emit", None)) or callable(getattr(ctx, "append", None))):
            raise ValueError("atb: context must provide emit or append")
        self._ctx = ctx

    def emit(self, tool_name: str, **kwargs: Any) -> None:
        """Emit an atb.tool.call event.

        Args:
            tool_name: The name of the tool invoked. Required.
            **kwargs: Optional fields — actor_id, tool_input_digest,
                tool_output_digest, session_id (overrides context session_id
                if provided).

        Raises:
            ValueError: If tool_name is empty or not provided.
        """
        payload = _base_payload(self._ctx, kwargs)
        payload["tool_name"] = _require_non_empty(tool_name, "tool_name")
        _optional_str(payload, kwargs, "actor_id")
        _optional_str(payload, kwargs, "tool_input_digest")
        _optional_str(payload, kwargs, "tool_output_digest")
        _dispatch(self._ctx, event_types.TOOL_CALL, payload)


class DataExportEmitter:
    """Emits atb.data.export events for outbound data movements."""

    def __init__(self, ctx: Any) -> None:
        if not (callable(getattr(ctx, "emit", None)) or callable(getattr(ctx, "append", None))):
            raise ValueError("atb: context must provide emit or append")
        self._ctx = ctx

    def emit(self, export_target: str, **kwargs: Any) -> None:
        """Emit an atb.data.export event.

        Args:
            export_target: Identifier of the export destination. Required.
            **kwargs: Optional actor_id, record_count (int), classification.

        Raises:
            ValueError: If export_target is empty or not provided.
        """
        payload = _base_payload(self._ctx, kwargs)
        payload["export_target"] = _require_non_empty(export_target, "export_target")
        _optional_str(payload, kwargs, "actor_id")
        record_count = kwargs.get("record_count")
        if isinstance(record_count, int) and record_count > 0:
            payload["record_count"] = record_count
        _optional_str(payload, kwargs, "classification")
        _dispatch(self._ctx, event_types.DATA_EXPORT, payload)


class HumanOverrideEmitter:
    """Emits atb.human.override events when a human overrides an AI recommendation."""

    def __init__(self, ctx: Any) -> None:
        if not (callable(getattr(ctx, "emit", None)) or callable(getattr(ctx, "append", None))):
            raise ValueError("atb: context must provide emit or append")
        self._ctx = ctx

    def emit(self, override_reason: str, **kwargs: Any) -> None:
        """Emit an atb.human.override event.

        Args:
            override_reason: Short rationale for the override. Required.
            **kwargs: Optional actor_id, overridden_action_id.

        Raises:
            ValueError: If override_reason is empty or not provided.
        """
        payload = _base_payload(self._ctx, kwargs)
        payload["override_reason"] = _require_non_empty(override_reason, "override_reason")
        _optional_str(payload, kwargs, "actor_id")
        _optional_str(payload, kwargs, "overridden_action_id")
        _dispatch(self._ctx, event_types.HUMAN_OVERRIDE, payload)


class HumanApprovalEmitter:
    """Emits atb.human.approval events when a human explicitly approves a pending action."""

    def __init__(self, ctx: Any) -> None:
        if not (callable(getattr(ctx, "emit", None)) or callable(getattr(ctx, "append", None))):
            raise ValueError("atb: context must provide emit or append")
        self._ctx = ctx

    def emit(self, approved_action_id: str, **kwargs: Any) -> None:
        """Emit an atb.human.approval event.

        Args:
            approved_action_id: Identifier of the action being approved. Required.
            **kwargs: Optional actor_id, approver_id, note.

        Raises:
            ValueError: If approved_action_id is empty or not provided.
        """
        payload = _base_payload(self._ctx, kwargs)
        payload["approved_action_id"] = _require_non_empty(
            approved_action_id,
            "approved_action_id",
        )
        _optional_str(payload, kwargs, "actor_id")
        _optional_str(payload, kwargs, "approver_id")
        _optional_str(payload, kwargs, "note")
        _dispatch(self._ctx, event_types.HUMAN_APPROVAL, payload)


class ActionErrorEmitter:
    """Emits ai.action.error events for privileged actions that did not succeed."""

    def __init__(self, ctx: Any) -> None:
        if not (callable(getattr(ctx, "emit", None)) or callable(getattr(ctx, "append", None))):
            raise ValueError("atb: context must provide emit or append")
        self._ctx = ctx

    def emit(self, action_id: str, error_class: str, **kwargs: Any) -> None:
        """Emit an ai.action.error event.

        Args:
            action_id: Identifier of the action that failed. Required.
            error_class: One of failed, blocked, timeout, exception,
                denied_at_sink. Required.
            **kwargs: Optional actor_id, error_detail_digest, action_type,
                session_id (overrides context session_id if provided).

        Raises:
            ValueError: If action_id or error_class is empty or not provided.
        """
        payload = _base_payload(self._ctx, kwargs)
        payload["action_id"] = _require_non_empty(action_id, "action_id")
        payload["error_class"] = _require_non_empty(error_class, "error_class")
        _optional_str(payload, kwargs, "actor_id")
        _optional_str(payload, kwargs, "error_detail_digest")
        _optional_str(payload, kwargs, "action_type")
        _dispatch(self._ctx, event_types.AI_ACTION_ERROR_EVENT_TYPE, payload)
