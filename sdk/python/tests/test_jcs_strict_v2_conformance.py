"""Conformance tests for the additive ``atb-jcs-rfc8785-2`` Python profile.

The v1 profile remains frozen. These tests execute the authored v2 vectors,
re-run the v1 negative vectors through v2, check raw round-trip closure, and pin
the canonical-output policy boundaries.
"""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import pytest

from atb import (
    StrictCanonicalizationError,
    canonicalize_strict,
    canonicalize_strict_raw,
    canonicalize_strict_raw_v2,
    canonicalize_strict_v2,
)
from atb.canonicalize import canonicalize as canonicalize_legacy_value

JCS_DIR = Path(__file__).resolve().parents[3] / "test" / "jcs"

V2_POSITIVE = json.loads(
    (JCS_DIR / "strict_v2_positive_vectors.json").read_text(encoding="utf-8")
)
V2_NEGATIVE = json.loads(
    (JCS_DIR / "strict_v2_negative_vectors.json").read_text(encoding="utf-8")
)
V1_NEGATIVE = json.loads(
    (JCS_DIR / "strict_negative_vectors.json").read_text(encoding="utf-8")
)
NUMERIC_PARITY = json.loads(
    (JCS_DIR / "numeric_parity_cases.json").read_text(encoding="utf-8")
)


def _raw_v2_error(raw: bytes | str) -> str:
    with pytest.raises(StrictCanonicalizationError) as excinfo:
        canonicalize_strict_raw_v2(raw)
    return excinfo.value.error_class


def _value_v2_error(value: Any) -> str:
    with pytest.raises(StrictCanonicalizationError) as excinfo:
        canonicalize_strict_v2(value)
    return excinfo.value.error_class


@pytest.mark.parametrize("case", V2_POSITIVE["cases"], ids=lambda case: case["id"])
def test_v2_positive_vectors(case: dict[str, Any]) -> None:
    got = canonicalize_strict_raw_v2(bytes.fromhex(case["input_utf8_hex"]))
    assert got == bytes.fromhex(case["canonical_utf8_hex"])

    round_trip = canonicalize_strict_raw_v2(got)
    assert round_trip == got


@pytest.mark.parametrize("case", V2_NEGATIVE["cases"], ids=lambda case: case["id"])
def test_v2_negative_vectors(case: dict[str, Any]) -> None:
    assert (
        _raw_v2_error(bytes.fromhex(case["input_utf8_hex"]))
        == case["expected_error"]
    )


@pytest.mark.parametrize("case", V1_NEGATIVE["cases"], ids=lambda case: case["id"])
def test_v1_negative_vectors_remain_rejected_by_v2(case: dict[str, Any]) -> None:
    assert (
        _raw_v2_error(bytes.fromhex(case["input_utf8_hex"]))
        == case["expected_error"]
    )


def test_v2_value_mode_policy() -> None:
    assert _value_v2_error(1e20) == "number_out_of_policy"
    assert canonicalize_strict_v2(1e21) == b"1e+21"
    assert canonicalize_strict_v2(1e-6) == b"0.000001"
    assert canonicalize_strict_v2(9007199254740991) == b"9007199254740991"
    assert _value_v2_error(10**21) == "number_out_of_policy"


def test_v2_fraction_interval_both_signs_reject() -> None:
    assert _raw_v2_error('{"n":36028797018963968.0}') == "number_out_of_policy"
    assert _raw_v2_error('{"n":-9007199254740992.0}') == "number_out_of_policy"


def test_v2_python_duplicate_key_precedence_caveat() -> None:
    # Python detects the duplicate during decoding before its serialisation
    # output-policy pass. Go and TypeScript report number_out_of_policy on the
    # first 1e20 member value. Both outcomes fail closed.
    assert _raw_v2_error(b'{"a":1e20,"a":2}') == "duplicate_key"


def test_v2_raw_plain_10e21_asymmetry_is_pinned() -> None:
    assert _raw_v2_error("1000000000000000000000") == "number_out_of_policy"
    assert canonicalize_strict_raw_v2("1e21") == b"1e+21"


def test_v1_profile_remains_frozen() -> None:
    assert canonicalize_strict_raw(b'{"n":1e20}') == b'{"n":100000000000000000000}'
    assert canonicalize_strict(1e20) == b"100000000000000000000"
    assert canonicalize_strict(1e-6) == b"0.000001"


def test_numeric_parity_fixture() -> None:
    case = NUMERIC_PARITY["cases"][0]
    input_bytes = bytes.fromhex(case["input_utf8_hex"])

    legacy_value = json.loads(input_bytes.decode("utf-8"))
    assert (
        canonicalize_legacy_value(legacy_value).hex()
        == case["legacy_expected"]["python"]
    )
    strict = canonicalize_strict_raw_v2(input_bytes).hex()
    assert strict == case["strict_expected"]["python"]
    assert strict == case["strict_expected"]["go"]
    assert strict == case["strict_expected"]["typescript"]
