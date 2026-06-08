"""Opt-in capture adapters for the direct OpenAI and Anthropic SDK clients.

Unlike LangChain, the first-party ``openai`` and ``anthropic`` Python clients
expose no callback hook. The thin adapter therefore wraps the client's
``create`` method: it records the request, calls through, records the response
(or error), and returns the untouched value. All event emission is delegated to
the existing :class:`atb.langchain_callback.ATBCallbackHandler` recorder, so this
module only maps each SDK's request/response shape onto the shared callbacks — it
adds no second emit path.

Quick start::

    from openai import OpenAI
    from atb import Bundle
    from atb.sdk_capture import wrap_openai

    client = OpenAI()
    create = wrap_openai(
        client.chat.completions.create, bundle=Bundle(), privacy_mode="hash"
    )
    res = create(model="gpt-4o", messages=[{"role": "user", "content": "hi"}])

Blind spots (documented, by design):
    * Streaming responses (``stream=True``) are NOT supported by the thin adapter
      — it cannot aggregate a streamed result without consuming the caller's
      iterator. Such calls raise ``ValueError``; use ``atb intercept`` (proxy
      capture) for token-level streaming capture.
    * Only the chat/messages ``create`` call is instrumented. Embeddings, files,
      and other endpoints are pass-through and unrecorded.
"""

from __future__ import annotations

import uuid
from typing import Any, Callable

from atb.bundle import Bundle
from atb.langchain_callback import ATBCallbackHandler, PrivacyMode


def wrap_openai(
    create: Callable[..., Any],
    *,
    handler: ATBCallbackHandler | None = None,
    bundle: Bundle | None = None,
    enabled: bool = True,
    privacy_mode: PrivacyMode = "off",
    actor_id: str | None = None,
    org_id: str | None = None,
    workspace_id: str | None = None,
) -> Callable[..., Any]:
    """Wrap an OpenAI ``chat.completions.create`` callable with ATB capture.

    Args:
        create: The bound ``client.chat.completions.create`` method.
        handler: Optional existing recorder to reuse; one is built when absent.
        bundle: Optional bundle to record into (used when ``handler`` is absent).
        enabled: Disable all recording when false (the call still passes through).
        privacy_mode: ``"off"``, ``"hash"``, or ``"redact"`` text handling.
        actor_id: Optional actor identity metadata.
        org_id: Optional organisation identity metadata.
        workspace_id: Optional workspace identity metadata.

    Returns:
        A drop-in replacement for ``create`` that records each call.

    Raises:
        ValueError: When a streaming request (``stream=True``) is passed.
    """
    rec = handler or ATBCallbackHandler(
        bundle=bundle,
        enabled=enabled,
        privacy_mode=privacy_mode,
        actor_id=actor_id,
        org_id=org_id,
        workspace_id=workspace_id,
        framework="openai",
    )

    def instrumented(**params: Any) -> Any:
        if params.get("stream"):
            raise ValueError(
                "wrap_openai does not support streaming (stream=True); "
                "use atb intercept for token-level streaming capture"
            )
        run_id = _new_run_id()
        model = params.get("model", "unknown")
        prompt = _serialize_messages(params.get("messages"))
        rec.on_llm_start(
            {"name": model, "id": ["openai"]},
            [prompt],
            run_id=run_id,
            parent_run_id=None,
            invocation_params={"model": model, "provider": "openai"},
        )
        try:
            response = create(**params)
        except Exception as exc:  # noqa: BLE001 - re-raised after recording.
            rec.on_llm_error(exc, run_id=run_id)
            raise

        choices = _get(response, "choices") or []
        choice = choices[0] if choices else None
        message = _get(choice, "message") if choice is not None else None
        text = (_get(message, "content") if message is not None else "") or ""
        finish_reason = _get(choice, "finish_reason") if choice is not None else None
        usage = _get(response, "usage")
        _record_tool_calls(rec, run_id, _openai_tool_calls(message))
        rec.on_llm_end(
            _LLMResult(text, _openai_usage(usage), finish_reason),
            run_id=run_id,
        )
        return response

    return instrumented


def wrap_anthropic(
    create: Callable[..., Any],
    *,
    handler: ATBCallbackHandler | None = None,
    bundle: Bundle | None = None,
    enabled: bool = True,
    privacy_mode: PrivacyMode = "off",
    actor_id: str | None = None,
    org_id: str | None = None,
    workspace_id: str | None = None,
) -> Callable[..., Any]:
    """Wrap an Anthropic ``messages.create`` callable with ATB capture.

    Args:
        create: The bound ``client.messages.create`` method.
        handler: Optional existing recorder to reuse; one is built when absent.
        bundle: Optional bundle to record into (used when ``handler`` is absent).
        enabled: Disable all recording when false (the call still passes through).
        privacy_mode: ``"off"``, ``"hash"``, or ``"redact"`` text handling.
        actor_id: Optional actor identity metadata.
        org_id: Optional organisation identity metadata.
        workspace_id: Optional workspace identity metadata.

    Returns:
        A drop-in replacement for ``create`` that records each call.

    Raises:
        ValueError: When a streaming request (``stream=True``) is passed.
    """
    rec = handler or ATBCallbackHandler(
        bundle=bundle,
        enabled=enabled,
        privacy_mode=privacy_mode,
        actor_id=actor_id,
        org_id=org_id,
        workspace_id=workspace_id,
        framework="anthropic",
    )

    def instrumented(**params: Any) -> Any:
        if params.get("stream"):
            raise ValueError(
                "wrap_anthropic does not support streaming (stream=True); "
                "use atb intercept for token-level streaming capture"
            )
        run_id = _new_run_id()
        model = params.get("model", "unknown")
        prompt = _serialize_messages(params.get("messages"), params.get("system"))
        rec.on_llm_start(
            {"name": model, "id": ["anthropic"]},
            [prompt],
            run_id=run_id,
            parent_run_id=None,
            invocation_params={"model": model, "provider": "anthropic"},
        )
        try:
            response = create(**params)
        except Exception as exc:  # noqa: BLE001 - re-raised after recording.
            rec.on_llm_error(exc, run_id=run_id)
            raise

        blocks = _get(response, "content") or []
        text = "".join(
            str(_get(b, "text") or "") for b in blocks if _get(b, "type") == "text"
        )
        tool_calls = [
            (_get(b, "name") or "unknown", _get(b, "input"))
            for b in blocks
            if _get(b, "type") == "tool_use"
        ]
        usage = _get(response, "usage")
        stop_reason = _get(response, "stop_reason")
        _record_tool_calls(rec, run_id, tool_calls)
        rec.on_llm_end(
            _LLMResult(text, _anthropic_usage(usage), stop_reason),
            run_id=run_id,
        )
        return response

    return instrumented


# ---------------------------------------------------------------------------
# Internal helpers
# ---------------------------------------------------------------------------


class _Generation:
    """Duck-typed LangChain generation: exposes ``.text``."""

    def __init__(self, text: str) -> None:
        self.text = text


class _LLMResult:
    """Duck-typed LangChain ``LLMResult`` consumed by ``ATBCallbackHandler``."""

    def __init__(self, text: str, usage: dict[str, int], finish_reason: Any) -> None:
        self.generations = [[_Generation(text)]]
        self.llm_output = {"token_usage": usage, "finish_reason": finish_reason}


def _record_tool_calls(
    rec: ATBCallbackHandler, run_id: str, calls: list[tuple[str, Any]]
) -> None:
    for index, (name, payload) in enumerate(calls):
        tool_run_id = f"{run_id}:tool:{index}"
        rec.on_tool_start(
            {"name": name},
            _to_input_str(payload),
            run_id=tool_run_id,
            parent_run_id=run_id,
        )
        rec.on_tool_end(payload, run_id=tool_run_id)


def _openai_tool_calls(message: Any) -> list[tuple[str, Any]]:
    raw = _get(message, "tool_calls") if message is not None else None
    if not raw:
        return []
    calls: list[tuple[str, Any]] = []
    for call in raw:
        fn = _get(call, "function")
        name = (_get(fn, "name") if fn is not None else None) or "unknown"
        arguments = _get(fn, "arguments") if fn is not None else None
        calls.append((name, arguments))
    return calls


def _openai_usage(usage: Any) -> dict[str, int]:
    prompt = int(_get(usage, "prompt_tokens") or 0)
    completion = int(_get(usage, "completion_tokens") or 0)
    total = int(_get(usage, "total_tokens") or (prompt + completion))
    return {
        "prompt_tokens": prompt,
        "completion_tokens": completion,
        "total_tokens": total,
    }


def _anthropic_usage(usage: Any) -> dict[str, int]:
    prompt = int(_get(usage, "input_tokens") or 0)
    completion = int(_get(usage, "output_tokens") or 0)
    return {
        "prompt_tokens": prompt,
        "completion_tokens": completion,
        "total_tokens": prompt + completion,
    }


def _serialize_messages(messages: Any, system: str | None = None) -> str:
    lines: list[str] = []
    if isinstance(system, str) and system != "":
        lines.append(f"system: {system}")
    for message in messages or []:
        role = _get(message, "role") or "user"
        content = _get(message, "content")
        lines.append(f"{role}: {_to_input_str(content)}")
    return "\n".join(lines)


def _to_input_str(value: Any) -> str:
    if value is None:
        return ""
    if isinstance(value, str):
        return value
    import json

    try:
        return json.dumps(value, sort_keys=True, ensure_ascii=True, default=str)
    except TypeError:
        return str(value)


def _get(obj: Any, key: str) -> Any:
    """Read ``key`` from a dict or an attribute-bearing SDK object."""
    if obj is None:
        return None
    if isinstance(obj, dict):
        return obj.get(key)
    return getattr(obj, key, None)


def _new_run_id() -> str:
    return f"run_{uuid.uuid4().hex}"
