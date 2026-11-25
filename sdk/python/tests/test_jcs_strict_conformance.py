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


def _value_error_class(value: Any) -> str:
    with pytest.raises(StrictCanonicalizationError) as excinfo:
        canonicalize_strict(value)
    return excinfo.value.error_class


def _nested_arrays(depth: int) -> Any:
    value: Any = []
    for _ in range(1, depth):
        value = [value]
    return value


def _nested_objects(depth: int) -> Any:
    value: Any = {}
    for _ in range(1, depth):
        value = {"a": value}
    return value


def _raw_nested_objects(depth: int) -> bytes:
    return b'{"a":' * (depth - 1) + b"{}" + b"}" * (depth - 1)


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
    "raw",
    [
        b"NaNx",
        b"Infinityx",
        b"-Infinityx",
        b"NaN1",
        b'{"n":NaNx}',
        b"[Infinityx]",
        b'{"n":-Infinityx}',
        b"[NaN1]",
    ],
    ids=[
        "nan-prefix",
        "infinity-prefix",
        "negative-infinity-prefix",
        "nan-digit",
        "object-nan-prefix",
        "array-infinity-prefix",
        "object-negative-infinity-prefix",
        "array-nan-digit",
    ],
)
def test_non_finite_literal_bad_boundary_is_invalid_json(raw: bytes) -> None:
    assert _error_class(raw) == "invalid_json"


@pytest.mark.parametrize(
    "raw",
    [
        b"NaN",
        b"Infinity",
        b"-Infinity",
        b'{"n":NaN}',
        b"[Infinity]",
        b'{"n":-Infinity}',
    ],
    ids=[
        "nan",
        "infinity",
        "negative-infinity",
        "object-nan",
        "array-infinity",
        "object-negative-infinity",
    ],
)
def test_exact_non_finite_literal_is_non_finite_number(raw: bytes) -> None:
    assert _error_class(raw) == "non_finite_number"


@pytest.mark.parametrize(
    ("raw", "expected"),
    [
        (b'"NaNx"', b'"NaNx"'),
        (b'{"s":"NaNx"}', b'{"s":"NaNx"}'),
        (b'["Infinityx","-Infinityx"]', b'["Infinityx","-Infinityx"]'),
    ],
    ids=["top-level-string", "object-value", "array-values"],
)
def test_non_finite_literal_text_inside_string_is_accepted(
    raw: bytes, expected: bytes
) -> None:
    assert canonicalize_strict_raw(raw) == expected


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


def test_raw_depth_limit_arrays() -> None:
    accepted = b"[" * 512 + b"]" * 512
    assert canonicalize_strict_raw(accepted) == accepted
    assert _error_class(b"[" * 513 + b"]" * 513) == "depth_limit"
    assert _error_class(b"[" * 5120 + b"]" * 5120) == "depth_limit"


def test_raw_depth_limit_objects() -> None:
    accepted = _raw_nested_objects(512)
    assert canonicalize_strict_raw(accepted) == accepted
    assert _error_class(_raw_nested_objects(513)) == "depth_limit"


def test_depth_scan_ignores_brackets_inside_strings() -> None:
    raw = b'{"s":"[[[{{{]]]}}}"}'
    assert canonicalize_strict_raw(raw) == raw


def test_value_mode_depth_limit_arrays() -> None:
    assert canonicalize_strict(_nested_arrays(512)) == b"[" * 512 + b"]" * 512
    assert _value_error_class(_nested_arrays(513)) == "depth_limit"


def test_value_mode_depth_limit_objects() -> None:
    expected = b'{"a":' * 511 + b"{}" + b"}" * 511
    assert canonicalize_strict(_nested_objects(512)) == expected
    assert _value_error_class(_nested_objects(513)) == "depth_limit"


def test_value_mode_rejects_list_cycle() -> None:
    cyclic: list[Any] = []
    cyclic.append(cyclic)
    with pytest.raises(StrictCanonicalizationError) as excinfo:
        canonicalize_strict(cyclic)
    assert excinfo.value.error_class == "invalid_json"
    assert "cyclic value" in str(excinfo.value)


def test_value_mode_rejects_dict_cycle() -> None:
    cyclic: dict[str, Any] = {}
    cyclic["self"] = cyclic
    with pytest.raises(StrictCanonicalizationError) as excinfo:
        canonicalize_strict(cyclic)
    assert excinfo.value.error_class == "invalid_json"
    assert "cyclic value" in str(excinfo.value)


def test_value_mode_rejects_indirect_cycle() -> None:
    first: list[Any] = []
    second: list[Any] = [first]
    first.append(second)
    assert _value_error_class(first) == "invalid_json"


def test_value_mode_accepts_shared_non_cyclic_references() -> None:
    shared_object = {"a": 1}
    assert canonicalize_strict([shared_object, shared_object]) == (
        b'[{"a":1},{"a":1}]'
    )
    shared_list = [1]
    assert canonicalize_strict([shared_list, shared_list]) == b"[[1],[1]]"


@pytest.mark.parametrize(
    ("raw", "expected"),
    [
        (b'{"a":1,"a":1e999}', "number_unrepresentable"),
        (b'{"a":1,"a":}', "invalid_json"),
        (b'{"a":1,"a":"\\ud800"}', "lone_surrogate"),
        (b'{"a":1,"\\ud800":2,"\\ud800":3}', "lone_surrogate"),
        (b'{"a":1,"a":2}', "duplicate_key"),
    ],
    ids=[
        "unrepresentable-value",
        "invalid-value",
        "lone-surrogate-value",
        "lone-surrogate-duplicate-key",
        "plain-duplicate",
    ],
)
def test_duplicate_precedence(raw: bytes, expected: str) -> None:
    assert _error_class(raw) == expected


@pytest.mark.parametrize(
    ("raw", "expected"),
    [
        (b'{"n":1e-999}', b'{"n":0}'),
        (b'{"n":-1e-999}', b'{"n":0}'),
        (b"1e-999", b"0"),
    ],
    ids=["positive", "negative", "top-level"],
)
def test_underflow_is_canonical_zero(raw: bytes, expected: bytes) -> None:
    assert canonicalize_strict_raw(raw) == expected


def test_value_mode_underflow_is_canonical_zero() -> None:
    assert canonicalize_strict(1e-999) == b"0"
    assert canonicalize_strict(float("1e-999")) == b"0"
