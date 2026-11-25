"""Strict ``atb-jcs-rfc8785-1`` JSON canonicalisation for the Python SDK.

This module implements RFC 8785 JSON Canonicalization Scheme followed by ATB's
safe-integer interoperability restriction. It is a new, non-production entry
point; the legacy canonicaliser in :mod:`atb.canonicalize` and the hashing path
do not use it.

The strict profile rejects duplicate object member names, invalid UTF-8,
unpaired UTF-16 surrogates, trailing data, non-finite numbers, plain integer
literals whose magnitude exceeds ``2**53 - 1``, and numbers that cannot be
represented as a finite binary64 value.
"""

from __future__ import annotations

import json
import math
import re
from typing import Any

__all__ = [
    "StrictCanonicalizationError",
    "canonicalize_strict",
    "canonicalize_strict_raw",
]

_CLASS_DUPLICATE_KEY = "duplicate_key"
_CLASS_INVALID_UTF8 = "invalid_utf8"
_CLASS_LONE_SURROGATE = "lone_surrogate"
_CLASS_TRAILING_DATA = "trailing_data"
_CLASS_NON_FINITE_NUMBER = "non_finite_number"
_CLASS_NUMBER_OUT_OF_POLICY = "number_out_of_policy"
_CLASS_NUMBER_UNREPRESENTABLE = "number_unrepresentable"
_CLASS_INVALID_JSON = "invalid_json"
_CLASS_DEPTH_LIMIT = "depth_limit"

_MAX_SAFE_INTEGER = 2**53 - 1
_MAX_SAFE_INTEGER_DIGITS = len(str(_MAX_SAFE_INTEGER))
_MAX_NESTING_DEPTH = 512

_WHITESPACE = " \t\r\n"

# The non-finite literals Python's decoder recognises via ``parse_constant``.
# The longer ``-Infinity`` form is checked first so it is not matched as
# ``Infinity`` when the scan starts on the leading minus.
_NON_FINITE_LITERALS = ("-Infinity", "Infinity", "NaN")

# Characters that may legally follow a complete non-finite literal. Anything
# else continues the token (for example ``NaNx``) and is malformed JSON.
_NON_FINITE_FOLLOWERS = _WHITESPACE + ",]}"

# A well-formed JSON number token, used to decide whether the characters after
# a parsed top-level number continue that token (malformed JSON) or start a
# separate value (trailing data).
_NUMBER_TOKEN_RE = re.compile(
    r"-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$"
)

_ESCAPE_MAP = {
    '"': '\\"',
    "\\": "\\\\",
    "\b": "\\b",
    "\f": "\\f",
    "\n": "\\n",
    "\r": "\\r",
    "\t": "\\t",
}


class StrictCanonicalizationError(ValueError):
    """Raised when the strict profile rejects a value.

    Attributes:
        error_class: Stable machine-readable rejection class. One of
            ``duplicate_key``, ``invalid_utf8``, ``lone_surrogate``,
            ``trailing_data``, ``non_finite_number``, ``number_out_of_policy``,
            ``number_unrepresentable``, ``invalid_json``, or ``depth_limit``.
    """

    def __init__(self, error_class: str, message: str) -> None:
        """Create an error carrying a stable rejection class.

        Args:
            error_class: Stable machine-readable rejection class.
            message: Human-readable detail.

        Returns:
            None.

        Raises:
            None.
        """
        super().__init__(f"{error_class}: {message}")
        self.error_class = error_class


def canonicalize_strict(value: Any) -> bytes:
    """Return the strict canonical UTF-8 encoding of a Python value.

    Args:
        value: JSON-compatible Python value. Supported types are ``None``,
            booleans, safe integers, finite floats, strings without unpaired
            surrogates, lists/tuples, and dictionaries with string keys.

    Returns:
        The strict canonical UTF-8 JSON bytes.

    Raises:
        StrictCanonicalizationError: If a number violates the strict numeric
            policy or a string contains an unpaired surrogate.
        TypeError: If ``value`` contains an unsupported type or a non-string
            object key.
    """
    return _serialize(value).encode("utf-8")


def canonicalize_strict_raw(
    data: bytes | bytearray | memoryview | str,
) -> bytes:
    """Return the strict canonical form of one raw JSON document.

    Args:
        data: UTF-8 bytes-like data, or an already-decoded string.

    Returns:
        The strict canonical UTF-8 JSON bytes.

    Raises:
        StrictCanonicalizationError: If the input is invalid UTF-8, is not
            exactly one JSON value, contains duplicate member names or
            unpaired surrogates, or contains a number rejected by the strict
            numeric policy.
        TypeError: If ``data`` is neither bytes-like nor a string.
    """
    text = _decode_input(data)
    _check_nesting_depth(text)
    _check_non_finite_token_boundaries(text)
    decoder = json.JSONDecoder(
        object_pairs_hook=_object_pairs_hook,
        parse_constant=_reject_constant,
        parse_int=_parse_int,
        parse_float=_parse_float,
    )
    start = _skip_whitespace(text, 0)
    try:
        value, end = decoder.raw_decode(text, start)
    except StrictCanonicalizationError:
        raise
    except json.JSONDecodeError as exc:
        raise StrictCanonicalizationError(
            _CLASS_INVALID_JSON,
            f"invalid JSON at offset {exc.pos}: {exc.msg}",
        ) from exc
    except (ValueError, OverflowError) as exc:
        raise StrictCanonicalizationError(
            _CLASS_INVALID_JSON, f"invalid JSON: {exc}"
        ) from exc
    next_index = _skip_whitespace(text, end)
    if next_index != len(text):
        if (
            next_index == end
            and _is_number(value)
            and _malformed_number_continuation(text, end)
        ):
            raise StrictCanonicalizationError(
                _CLASS_INVALID_JSON,
                "malformed number token at the top level",
            )
        raise StrictCanonicalizationError(
            _CLASS_TRAILING_DATA,
            "non-whitespace data follows the top-level JSON value",
        )
    _ensure_no_lone_surrogates(value)
    return _serialize(value).encode("utf-8")


def _decode_input(data: bytes | bytearray | memoryview | str) -> str:
    if isinstance(data, str):
        return data
    if not isinstance(data, (bytes, bytearray, memoryview)):
        raise TypeError(
            f"canonicalize_strict_raw: expected bytes-like or str, got {type(data)!r}"
        )
    raw = bytes(data)
    try:
        return raw.decode("utf-8")
    except UnicodeDecodeError as exc:
        raise StrictCanonicalizationError(
            _CLASS_INVALID_UTF8,
            f"input is not valid UTF-8 at byte offset {exc.start}",
        ) from exc


def _skip_whitespace(text: str, index: int) -> int:
    while index < len(text) and text[index] in _WHITESPACE:
        index += 1
    return index


def _check_nesting_depth(text: str) -> None:
    """Reject raw JSON nested beyond the strict profile limit.

    The scan is intentionally lexical: brackets inside strings, including
    escaped quotes, are ignored. It runs before the JSON decoder so that very
    deep documents cannot exhaust the interpreter stack.
    """
    depth = 0
    in_string = False
    escaped = False
    for char in text:
        if in_string:
            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == '"':
                in_string = False
            continue
        if char == '"':
            in_string = True
        elif char in "[{":
            depth += 1
            if depth > _MAX_NESTING_DEPTH:
                raise StrictCanonicalizationError(
                    _CLASS_DEPTH_LIMIT,
                    "maximum nesting depth exceeded",
                )
        elif char in "]}" and depth > 0:
            depth -= 1


def _check_non_finite_token_boundaries(text: str) -> None:
    """Reject non-finite literals that are not complete tokens.

    Python's decoder consumes the ``NaN``/``Infinity``/``-Infinity`` prefix
    and calls ``parse_constant`` before it notices a trailing character, so
    ``NaNx`` would be reported as a non-finite number rather than malformed
    JSON. This lexical scan runs before decoding and ignores strings, matching
    the token-boundary rule enforced by the Go and TypeScript implementations.
    Exact literals are left for ``parse_constant`` so they still produce
    ``non_finite_number``.
    """
    in_string = False
    escaped = False
    index = 0
    length = len(text)
    while index < length:
        char = text[index]
        if in_string:
            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == '"':
                in_string = False
            index += 1
            continue
        if char == '"':
            in_string = True
            index += 1
            continue
        for literal in _NON_FINITE_LITERALS:
            if text.startswith(literal, index):
                next_index = index + len(literal)
                if (
                    next_index < length
                    and text[next_index] not in _NON_FINITE_FOLLOWERS
                ):
                    raise StrictCanonicalizationError(
                        _CLASS_INVALID_JSON,
                        f"non-finite literal {literal} is not a complete token",
                    )
                index = next_index
                break
        else:
            index += 1


def _is_number(value: Any) -> bool:
    # ``bool`` is an ``int`` subclass, so exclude it explicitly.
    return isinstance(value, (int, float)) and not isinstance(value, bool)


def _malformed_number_continuation(text: str, end: int) -> bool:
    if end >= len(text):
        return False
    nxt = text[end]
    match = _NUMBER_TOKEN_RE.search(text[:end])
    if match is None:
        return False
    token = match.group(0)
    if token in ("0", "-0") and nxt.isdigit():
        return True
    if "e" not in token and "E" not in token:
        if nxt in ("e", "E"):
            # A fraction followed by an exponent marker without exponent
            # digits is a continued malformed token, not trailing data.
            return True
        if "." not in token and nxt == ".":
            return True
    return False


def _object_pairs_hook(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    obj: dict[str, Any] = {}
    for key, value in pairs:
        # JSON parsing rejects unpaired surrogates as it reads them, which is
        # before duplicate detection runs on the completed member. Mirror that
        # precedence here because Python's decoder accepts such code points.
        if _is_lone_surrogate(key):
            raise StrictCanonicalizationError(
                _CLASS_LONE_SURROGATE,
                "unpaired UTF-16 surrogate code unit in member name",
            )
        if _contains_lone_surrogate(value):
            raise StrictCanonicalizationError(
                _CLASS_LONE_SURROGATE,
                "unpaired UTF-16 surrogate code unit",
            )
        if key in obj:
            raise StrictCanonicalizationError(
                _CLASS_DUPLICATE_KEY,
                f"duplicate object member name {key!r}",
            )
        obj[key] = value
    return obj


def _reject_constant(token: str) -> Any:
    raise StrictCanonicalizationError(
        _CLASS_NON_FINITE_NUMBER,
        f"non-finite number literal {token}",
    )


def _parse_int(token: str) -> int:
    digits = token[1:] if token.startswith("-") else token
    significant = digits.lstrip("0") or "0"
    if len(significant) > _MAX_SAFE_INTEGER_DIGITS:
        raise StrictCanonicalizationError(
            _CLASS_NUMBER_OUT_OF_POLICY,
            "plain integer literal exceeds 2^53 - 1",
        )
    try:
        value = int(token)
    except ValueError as exc:
        raise StrictCanonicalizationError(
            _CLASS_INVALID_JSON, f"invalid integer literal {token!r}"
        ) from exc
    if abs(value) > _MAX_SAFE_INTEGER:
        raise StrictCanonicalizationError(
            _CLASS_NUMBER_OUT_OF_POLICY,
            "plain integer literal exceeds 2^53 - 1",
        )
    return value


def _parse_float(token: str) -> float:
    try:
        value = float(token)
    except (ValueError, OverflowError) as exc:
        raise StrictCanonicalizationError(
            _CLASS_NUMBER_UNREPRESENTABLE,
            f"number {token!r} cannot be represented as binary64",
        ) from exc
    if not math.isfinite(value):
        raise StrictCanonicalizationError(
            _CLASS_NUMBER_UNREPRESENTABLE,
            f"number {token!r} cannot be represented as a finite binary64",
        )
    return value


def _is_lone_surrogate(text: str) -> bool:
    return any(0xD800 <= ord(char) <= 0xDFFF for char in text)


def _contains_lone_surrogate(value: Any) -> bool:
    if isinstance(value, str):
        return _is_lone_surrogate(value)
    if isinstance(value, (list, tuple)):
        for item in value:
            if _contains_lone_surrogate(item):
                return True
    elif isinstance(value, dict):
        for key, item in value.items():
            if _is_lone_surrogate(key) or _contains_lone_surrogate(item):
                return True
    return False


def _ensure_no_lone_surrogates(value: Any) -> None:
    if isinstance(value, str):
        if _is_lone_surrogate(value):
            raise StrictCanonicalizationError(
                _CLASS_LONE_SURROGATE,
                "unpaired UTF-16 surrogate code unit",
            )
    elif isinstance(value, list):
        for item in value:
            _ensure_no_lone_surrogates(item)
    elif isinstance(value, dict):
        for key, item in value.items():
            if _is_lone_surrogate(key):
                raise StrictCanonicalizationError(
                    _CLASS_LONE_SURROGATE,
                    "unpaired UTF-16 surrogate code unit in member name",
                )
            _ensure_no_lone_surrogates(item)


def _serialize(
    value: Any,
    depth: int = 0,
    seen: set[int] | None = None,
) -> str:
    if seen is None:
        seen = set()
    if value is None:
        return "null"
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, int):
        if abs(value) > _MAX_SAFE_INTEGER:
            raise StrictCanonicalizationError(
                _CLASS_NUMBER_OUT_OF_POLICY,
                "integer value exceeds 2^53 - 1",
            )
        return str(value)
    if isinstance(value, float):
        return _format_float(value)
    if isinstance(value, str):
        return _encode_string(value)
    if isinstance(value, (list, tuple)):
        next_depth = _enter_container(value, depth, seen)
        try:
            parts: list[str] = []
            for item in value:
                parts.append(_serialize(item, next_depth, seen))
            return "[" + ",".join(parts) + "]"
        finally:
            seen.discard(id(value))
    if isinstance(value, dict):
        return _serialize_object(value, depth, seen)
    raise TypeError(f"canonicalize_strict: unsupported type {type(value)!r}")


def _enter_container(value: Any, depth: int, seen: set[int]) -> int:
    next_depth = depth + 1
    if next_depth > _MAX_NESTING_DEPTH:
        raise StrictCanonicalizationError(
            _CLASS_DEPTH_LIMIT,
            "maximum nesting depth exceeded",
        )
    identity = id(value)
    if identity in seen:
        raise StrictCanonicalizationError(
            _CLASS_INVALID_JSON,
            "cyclic value",
        )
    seen.add(identity)
    return next_depth


def _serialize_object(
    value: dict[Any, Any],
    depth: int = 0,
    seen: set[int] | None = None,
) -> str:
    if seen is None:
        seen = set()
    for key in value:
        if not isinstance(key, str):
            raise TypeError(
                f"canonicalize_strict: object keys must be strings, got {type(key)!r}"
            )
    next_depth = _enter_container(value, depth, seen)
    try:
        ordered_keys = sorted(value, key=_utf16_units)
        parts: list[str] = []
        for key in ordered_keys:
            prefix = f"{_encode_string(key)}:"
            item = value[key]
            if isinstance(item, dict):
                parts.append(prefix + _serialize_object(item, next_depth, seen))
            else:
                parts.append(prefix + _serialize(item, next_depth, seen))
        return "{" + ",".join(parts) + "}"
    finally:
        seen.discard(id(value))


def _encode_string(value: str) -> str:
    if _is_lone_surrogate(value):
        raise StrictCanonicalizationError(
            _CLASS_LONE_SURROGATE,
            "unpaired UTF-16 surrogate code unit",
        )
    parts = ['"']
    for char in value:
        escaped = _ESCAPE_MAP.get(char)
        if escaped is not None:
            parts.append(escaped)
        else:
            code = ord(char)
            if code < 0x20:
                parts.append(f"\\u{code:04x}")
            else:
                parts.append(char)
    parts.append('"')
    return "".join(parts)


def _utf16_units(value: str) -> tuple[int, ...]:
    units: list[int] = []
    for char in value:
        code = ord(char)
        if code <= 0xFFFF:
            units.append(code)
        else:
            offset = code - 0x10000
            units.append(0xD800 + (offset >> 10))
            units.append(0xDC00 + (offset & 0x3FF))
    return tuple(units)


def _format_float(value: float) -> str:
    if not math.isfinite(value):
        raise StrictCanonicalizationError(
            _CLASS_NON_FINITE_NUMBER,
            "value contains a non-finite number",
        )
    if value == 0:
        return "0"

    negative = math.copysign(1.0, value) < 0
    digits, decimal_exponent = _shortest_decimal_parts(abs(value))
    digit_count = len(digits)
    point_position = digit_count + decimal_exponent

    if digit_count <= point_position <= 21:
        out = digits + "0" * (point_position - digit_count)
    elif 0 < point_position <= 21:
        out = digits[:point_position] + "." + digits[point_position:]
    elif -6 < point_position <= 0:
        out = "0." + "0" * (-point_position) + digits
    else:
        if digit_count == 1:
            mantissa = digits
        else:
            mantissa = digits[0] + "." + digits[1:]
        exponent = point_position - 1
        sign = "+" if exponent >= 0 else "-"
        out = f"{mantissa}e{sign}{abs(exponent)}"

    return "-" + out if negative else out


def _shortest_decimal_parts(value: float) -> tuple[str, int]:
    """Return shortest decimal digits and a power-of-ten exponent.

    The result ``(digits, exponent)`` satisfies
    ``float(f"{digits}e{exponent}") == value`` and ``digits`` has no leading or
    trailing zeros.
    """
    text = repr(value)
    if "e" in text or "E" in text:
        mantissa, _, exponent_text = text.lower().partition("e")
        exponent = int(exponent_text)
    else:
        mantissa = text
        exponent = 0

    if "." in mantissa:
        integer_part, fraction_part = mantissa.split(".", 1)
    else:
        integer_part, fraction_part = mantissa, ""

    digits = (integer_part + fraction_part).lstrip("0")
    decimal_exponent = exponent - len(fraction_part)
    trimmed = digits.rstrip("0")
    decimal_exponent += len(digits) - len(trimmed)
    return trimmed, decimal_exponent
