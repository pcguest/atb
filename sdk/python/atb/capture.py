"""Chatlog capture helpers for the Python SDK.

Mirrors the surface of ``internal/capture`` in the Go reference
implementation: a ``parse_chatlog`` entry point and named exception
classes that callers can catch to classify input errors.

Only the ``generic-jsonl`` format is implemented today; other format
identifiers raise :class:`UnsupportedProviderError` for parity with the
Go CLI's ``ErrUnsupportedProvider``.

Quick start::

    from atb.capture import parse_chatlog
    messages = parse_chatlog("generic-jsonl", b'{"role":"user"}\\n')
    for message in messages: print(message.role)
"""

from __future__ import annotations

import io
import json
from dataclasses import dataclass
from typing import IO, Any, Iterable

#: Default cap on chatlog input size, mirroring the Go CLI's
#: ``defaultMaxImportBytes`` (256 MiB). Callers can override per-call.
DEFAULT_MAX_INPUT_BYTES = 256 * 1024 * 1024

FORMAT_GENERIC_JSONL = "generic-jsonl"
FORMAT_CLAUDE_DESKTOP = "claude-desktop"
FORMAT_OPENAI_JSONL = "openai-jsonl"


class UnsupportedProviderError(ValueError):
    """Raised when the requested chatlog format is not implemented.

    Mirrors ``capture.ErrUnsupportedProvider`` in the Go reference
    implementation. Inherits from :class:`ValueError` so existing
    ``except ValueError`` callers keep working.

    Args:
        *args: Positional error message arguments passed to ``ValueError``.

    Returns:
        None.

    Raises:
        None.
    """


class MalformedChatlogError(ValueError):
    """Raised when chatlog content is structurally invalid.

    Mirrors ``capture.ErrMalformedChatlog`` in the Go reference
    implementation. Inherits from :class:`ValueError` so existing
    ``except ValueError`` callers keep working.

    Args:
        *args: Positional error message arguments passed to ``ValueError``.

    Returns:
        None.

    Raises:
        None.
    """


class InputTooLargeError(ValueError):
    """Raised when the chatlog input exceeds the configured byte cap.

    Args:
        *args: Positional error message arguments passed to ``ValueError``.

    Returns:
        None.

    Raises:
        None.
    """


@dataclass
class ChatMessage:
    """Normalised representation of one parsed chatlog line.

    Args:
        line: One-based source line number.
        role: Normalised turn role.
        content: Turn content.
        timestamp: Source timestamp string.
        model: Optional model identifier.
        tool_name: Optional tool name.
        conversation_id: Optional conversation identifier.
        session_id: Optional session identifier.
        request_id: Optional request identifier.

    Returns:
        A dataclass instance.

    Raises:
        None.
    """

    line: int
    role: str
    content: str
    timestamp: str
    model: str = ""
    tool_name: str = ""
    conversation_id: str = ""
    session_id: str = ""
    request_id: str = ""


def parse_chatlog(
    format: str,
    source: IO[bytes] | IO[str] | bytes | str,
    *,
    max_bytes: int | None = DEFAULT_MAX_INPUT_BYTES,
) -> list[ChatMessage]:
    """Parse a chatlog *source* using *format*.

    Args:
        format: Provider identifier (e.g. ``"generic-jsonl"``).
        source: A bytes/str buffer, or a file-like object yielding either.
        max_bytes: Optional cap on total bytes consumed. Pass ``None`` to
            disable the cap. Defaults to :data:`DEFAULT_MAX_INPUT_BYTES`.

    Raises:
        UnsupportedProviderError: when *format* is not implemented.
        MalformedChatlogError: when the content cannot be parsed.
        InputTooLargeError: when *max_bytes* is exceeded.

    Returns:
        Parsed chat messages in source order.
    """
    fmt = (format or "").strip().lower()
    if fmt == FORMAT_OPENAI_JSONL:
        fmt = FORMAT_GENERIC_JSONL
    if fmt != FORMAT_GENERIC_JSONL:
        if fmt == FORMAT_CLAUDE_DESKTOP:
            raise UnsupportedProviderError(
                f"unsupported provider {format!r}: Capture v1 only implements "
                f"{FORMAT_GENERIC_JSONL!r} and {FORMAT_OPENAI_JSONL!r}"
            )
        raise UnsupportedProviderError(f"unsupported provider {format!r}")

    text = _read_all(source, max_bytes)
    return _parse_generic_jsonl(text)


def _read_all(
    source: IO[bytes] | IO[str] | bytes | str,
    max_bytes: int | None,
) -> str:
    if isinstance(source, (bytes, bytearray)):
        data = bytes(source)
        if max_bytes is not None and len(data) > max_bytes:
            raise InputTooLargeError(
                f"chatlog input exceeds max_bytes={max_bytes}"
            )
        return data.decode("utf-8")
    if isinstance(source, str):
        if max_bytes is not None and len(source.encode("utf-8")) > max_bytes:
            raise InputTooLargeError(
                f"chatlog input exceeds max_bytes={max_bytes}"
            )
        return source
    # File-like.
    if max_bytes is not None:
        chunk = source.read(max_bytes + 1)
        if isinstance(chunk, bytes):
            if len(chunk) > max_bytes:
                raise InputTooLargeError(
                    f"chatlog input exceeds max_bytes={max_bytes}"
                )
            return chunk.decode("utf-8")
        if len(chunk.encode("utf-8")) > max_bytes:
            raise InputTooLargeError(
                f"chatlog input exceeds max_bytes={max_bytes}"
            )
        return chunk
    raw = source.read()
    return raw.decode("utf-8") if isinstance(raw, bytes) else raw


def _parse_generic_jsonl(text: str) -> list[ChatMessage]:
    messages: list[ChatMessage] = []
    for line_no, raw in enumerate(io.StringIO(text), start=1):
        stripped = raw.strip()
        if not stripped:
            continue
        try:
            obj = json.loads(stripped)
        except json.JSONDecodeError as exc:
            raise MalformedChatlogError(
                f"parse generic JSONL line {line_no}: {exc.msg}"
            ) from exc
        if not isinstance(obj, dict):
            raise MalformedChatlogError(
                f"parse generic JSONL line {line_no}: expected object, got "
                f"{type(obj).__name__}"
            )
        try:
            messages.append(
                ChatMessage(
                    line=line_no,
                    role=str(obj.get("role", "")).strip().lower(),
                    content=str(obj.get("content", "")).strip(),
                    timestamp=str(obj.get("timestamp", "")).strip(),
                    model=str(obj.get("model", "")).strip(),
                    tool_name=str(obj.get("tool_name", "")).strip(),
                    conversation_id=str(obj.get("conversation_id", "")).strip(),
                    session_id=str(obj.get("session_id", "")).strip(),
                    request_id=str(obj.get("request_id", "")).strip(),
                )
            )
        except (TypeError, ValueError) as exc:
            raise MalformedChatlogError(
                f"parse generic JSONL line {line_no}: {exc}"
            ) from exc
    if not messages:
        raise MalformedChatlogError("parse generic JSONL: no records found")
    return messages


__all__ = [
    "ChatMessage",
    "DEFAULT_MAX_INPUT_BYTES",
    "FORMAT_CLAUDE_DESKTOP",
    "FORMAT_GENERIC_JSONL",
    "FORMAT_OPENAI_JSONL",
    "InputTooLargeError",
    "MalformedChatlogError",
    "UnsupportedProviderError",
    "parse_chatlog",
]
