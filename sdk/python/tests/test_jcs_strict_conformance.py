"""Conformance tests for the strict ``atb-jcs-rfc8785-1`` Python profile.

The positive and negative cases are the authored step-1 vectors in ``test/jcs``.
The remaining tests pin the strict numeric policy, duplicate rejection, UTF-8
and surrogate handling, trailing-data detection, and ECMAScript number
formatting.
"""

from __future__ import annotations

import json
import math
from pathlib import Path
from typing import Any

import pytest

from atb import (
    StrictCanonicalizationError,
    canonicalize_strict,
    canonicalize_strict_raw,
)

JCS_DIR = Path(__file__).resolve().parents[3] / "test" / "jcs"

POSITIVE = json.loads(
    (JCS_DIR / "strict_positive_vectors.json").read_text(encoding="utf-8")
)
NEGATIVE = json.loads(
    (JCS_DIR / "strict_negative_vectors.json").read_text(encoding="utf-8")
)

NUMBER_FORMATTING_CASES = [
    (0, "0"),
    (-0.0, "0"),
    (1.5, "1.5"),
    (0.1, "0.1"),
    (1e-7, "1e-7"),
    (1e-6, "0.000001"),
    (1e-5, "0.00001"),
    (1e20, "100000000000000000000"),
    (1e21, "1e+21"),
    (1e22, "1e+22"),
    (1e23, "1e+23"),
    (9007199254740991, "9007199254740991"),
    (1.2345678901234567, "1.2345678901234567"),
    (5e-324, "5e-324"),
    (1.7976931348623157e308, "1.7976931348623157e+308"),
]


def _error_class(raw: bytes | str) -> str:
    with pytest.raises(StrictCanonicalizationError) as excinfo:
        canonicalize_strict_raw(raw)
    return excinfo.value.error_class


@pytest.mark.parametrize("case", POSITIVE["cases"], ids=lambda case: case["id"])
def test_strict_positive_vectors(case: dict[str, Any]) -> None:
    got = canonicalize_strict_raw(bytes.fromhex(case["input_utf8_hex"]))
    assert got == bytes.fromhex(case["canonical_utf8_hex"])


@pytest.mark.parametrize("case", NEGATIVE["cases"], ids=lambda case: case["id"])
def test_strict_negative_vectors(case: dict[str, Any]) -> None:
    with pytest.raises(StrictCanonicalizationError) as excinfo:
        canonicalize_strict_raw(bytes.fromhex(case["input_utf8_hex"]))
    assert excinfo.value.error_class == case["expected_error"]


@pytest.mark.parametrize(
    ("value", "expected"),
    NUMBER_FORMATTING_CASES,
    ids=[repr(value) for value, _ in NUMBER_FORMATTING_CASES],
)
def test_es6_number_formatting(value: Any, expected: str) -> None:
    assert canonicalize_strict(value) == expected.encode("utf-8")


@pytest.mark.parametrize(
    "raw",
    [
        b'{"a":1,"a":2}',
        b'{"a":1,"\\u0061":2}',
        b'{"x":{"k":1,"k":2}}',
    ],
    ids=["plain", "escaped-equivalent", "nested"],
)
def test_duplicate_keys(raw: bytes) -> None:
    assert _error_class(raw) == "duplicate_key"


def test_invalid_utf8_bytes() -> None:
    assert _error_class(b'{"s":"\xc3("}') == "invalid_utf8"


@pytest.mark.parametrize(
    "raw",
    [
        b'{"s":"\\ud800"}',
        b'{"s":"\\udc00"}',
        b'{"s":"\\ud800\\u0041"}',
    ],
    ids=["high", "low", "high-then-non-low"],
)
def test_lone_surrogate_escapes(raw: bytes) -> None:
    assert _error_class(raw) == "lone_surrogate"


def test_trailing_data() -> None:
    assert _error_class(b'{"a":1} {"b":2}') == "trailing_data"


def test_overflowing_exponent_is_unrepresentable() -> None:
    assert _error_class(b'{"n":1e999}') == "number_unrepresentable"


@pytest.mark.parametrize(
    ("raw", "expected"),
    [
        (b"9007199254740991", b"9007199254740991"),
        (b"-9007199254740991", b"-9007199254740991"),
        (b"0", b"0"),
        (b"-0", b"0"),
    ],
)
def test_plain_integer_policy_accepts_boundaries(raw: bytes, expected: bytes) -> None:
    assert canonicalize_strict_raw(raw) == expected


@pytest.mark.parametrize(
    "raw",
    [
        b"9007199254740992",
        b"-9007199254740992",
        b"9007199254740993",
        b"90071992547409910000000000000000000",
    ],
)
def test_plain_integer_policy_rejects_out_of_range(raw: bytes) -> None:
    assert _error_class(raw) == "number_out_of_policy"


def test_value_mode_integer_policy() -> None:
    assert canonicalize_strict(9007199254740991) == b"9007199254740991"

    for value in (9007199254740992, -9007199254740992):
        with pytest.raises(StrictCanonicalizationError) as excinfo:
            canonicalize_strict(value)
        assert excinfo.value.error_class == "number_out_of_policy"


def test_value_mode_float_policy() -> None:
    assert canonicalize_strict(1e20) == b"100000000000000000000"

    for value in (math.nan, math.inf, -math.inf):
        with pytest.raises(StrictCanonicalizationError) as excinfo:
            canonicalize_strict(value)
        assert excinfo.value.error_class == "non_finite_number"


def test_value_mode_rejects_lone_surrogate() -> None:
    with pytest.raises(StrictCanonicalizationError) as excinfo:
        canonicalize_strict({"s": "\ud800"})
    assert excinfo.value.error_class == "lone_surrogate"


def test_value_mode_rejects_unsupported_type() -> None:
    with pytest.raises(TypeError):
        canonicalize_strict(object())
