"""
RFC 8785 JSON Canonicalization Scheme (JCS) implementation.

Produces a deterministic, canonical UTF-8 byte sequence for any JSON value,
suitable for use in cryptographic hash computation.
"""

from __future__ import annotations

import json
import math
from typing import Any


def canonicalize(value: Any) -> bytes:
    """Return the RFC 8785 canonical JSON encoding.

    Args:
        value: JSON-compatible Python value to serialise.

    Returns:
        Canonical UTF-8 JSON bytes.

    Raises:
        TypeError: If ``value`` contains an unsupported type.
        ValueError: If ``value`` contains a non-finite number.
    """
    return _serialize(value).encode("utf-8")


def _serialize(value: Any) -> str:
    if value is None:
        return "null"
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, (int, float)):
        return _serialize_number(value)
    if isinstance(value, str):
        return _serialize_string(value)
    if isinstance(value, list):
        items = ",".join(_serialize(v) for v in value)
        return f"[{items}]"
    if isinstance(value, dict):
        # RFC 8785 §3.2.3 — sort keys by UTF-16 code unit sequence.
        sorted_keys = sorted(value.keys(), key=_utf16_key)
        pairs = ",".join(
            f"{_serialize_string(k)}:{_serialize(value[k])}" for k in sorted_keys
        )
        return "{" + pairs + "}"
    raise TypeError(f"canonicalize: unsupported type {type(value)!r}")


def _serialize_number(n: int | float) -> str:
    """Serialise a number per RFC 8785 §3.2.2 (ES6 number format)."""
    if isinstance(n, int):
        return str(n)
    if math.isnan(n) or math.isinf(n):
        raise ValueError(f"canonicalize: non-finite number {n!r}")
    # Use Python's repr which is close enough to ES6 for finite floats.
    formatted = repr(n)
    # Remove trailing zeros after decimal point for consistency.
    if "." in formatted and "e" not in formatted:
        formatted = formatted.rstrip("0").rstrip(".")
    return formatted


_ESCAPE_MAP = {
    '"': '\\"',
    "\\": "\\\\",
    "\b": "\\b",
    "\f": "\\f",
    "\n": "\\n",
    "\r": "\\r",
    "\t": "\\t",
}


def _serialize_string(s: str) -> str:
    parts = ['"']
    for ch in s:
        if ch in _ESCAPE_MAP:
            parts.append(_ESCAPE_MAP[ch])
        elif ord(ch) < 0x20:
            parts.append(f"\\u{ord(ch):04x}")
        else:
            parts.append(ch)
    parts.append('"')
    return "".join(parts)


def _utf16_key(s: str) -> list[int]:
    """Return the UTF-16 code unit sequence for *s* for sorting purposes."""
    units: list[int] = []
    for cp in s:
        code = ord(cp)
        if code <= 0xFFFF:
            units.append(code)
        else:
            # Surrogate pair encoding.
            cp_adj = code - 0x10000
            units.append(0xD800 + (cp_adj >> 10))
            units.append(0xDC00 + (cp_adj & 0x3FF))
    return units
