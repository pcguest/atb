"""Validate the executable JCS canonicalisation-profile fixtures.

These fixtures cover the frozen legacy profile ``atb-jcs-legacy-1``, the
frozen strict profile ``atb-jcs-rfc8785-1``, the additive
``atb-jcs-rfc8785-2`` profile, and the numeric-parity case. This module
validates the normative fixtures and pins the Python legacy canonicaliser
against the frozen expectations. The strict fixtures are executed against the
strict parser by ``test_jcs_strict_conformance.py`` and
``test_jcs_strict_v2_conformance.py``; this module checks their status,
metadata, declared error classes, and the frozen numeric-parity case.
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from atb.canonicalize import canonicalize as canonicalize_value
from atb.canonicalize_strict import canonicalize_strict_raw_v2

JCS_DIR = Path(__file__).resolve().parents[3] / "test" / "jcs"

POSITIVE_PATH = JCS_DIR / "strict_positive_vectors.json"
NEGATIVE_PATH = JCS_DIR / "strict_negative_vectors.json"
V2_POSITIVE_PATH = JCS_DIR / "strict_v2_positive_vectors.json"
V2_NEGATIVE_PATH = JCS_DIR / "strict_v2_negative_vectors.json"
NUMERIC_PARITY_PATH = JCS_DIR / "numeric_parity_cases.json"
LEGACY_PATH = JCS_DIR / "legacy_freeze_vectors.json"
PROFILE_SELECTION_PATH = JCS_DIR / "profile_selection_cases.json"

POSITIVE_SCHEMA = "atb.jcs.strict-positive-vectors.v1"
NEGATIVE_SCHEMA = "atb.jcs.strict-negative-vectors.v1"
NUMERIC_PARITY_SCHEMA = "atb.jcs.numeric-parity-cases.v1"
LEGACY_SCHEMA = "atb.jcs.legacy-freeze-vectors.v1"
PROFILE_SELECTION_SCHEMA = "atb.jcs.profile-selection-cases.v1"

STRICT_PROFILE = "atb-jcs-rfc8785-1"
V2_PROFILE = "atb-jcs-rfc8785-2"
NUMERIC_PARITY_PROFILE = "atb-jcs-rfc8785-1-and-rfc8785-2"
LEGACY_PROFILE = "atb-jcs-legacy-1"

ALLOWED_ERROR_CLASSES = {
    "duplicate_key",
    "invalid_utf8",
    "lone_surrogate",
    "trailing_data",
    "non_finite_number",
    "number_out_of_policy",
    "number_unrepresentable",
    "invalid_json",
    "depth_limit",
    "missing_profile",
    "unsupported_profile",
}

REQUIRED_POSITIVE_CATEGORIES = {
    "key_ordering",
    "string_escaping",
    "numbers",
    "nesting",
}

REQUIRED_NEGATIVE_CATEGORIES = {
    "duplicate_key",
    "invalid_utf8",
    "lone_surrogate",
    "trailing_data",
    "non_finite_number",
    "number_out_of_policy",
    "number_unrepresentable",
    "invalid_json",
}


def _load(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


POSITIVE = _load(POSITIVE_PATH)
NEGATIVE = _load(NEGATIVE_PATH)
V2_POSITIVE = _load(V2_POSITIVE_PATH)
V2_NEGATIVE = _load(V2_NEGATIVE_PATH)
NUMERIC_PARITY = _load(NUMERIC_PARITY_PATH)
LEGACY = _load(LEGACY_PATH)
PROFILE_SELECTION = _load(PROFILE_SELECTION_PATH)


def test_positive_fixture_metadata() -> None:
    assert POSITIVE["schema"] == POSITIVE_SCHEMA
    assert POSITIVE["profile"] == STRICT_PROFILE
    assert POSITIVE["status"] == "executable"

    ids = [case["id"] for case in POSITIVE["cases"]]
    assert len(ids) == len(set(ids)), "positive case ids must be unique"

    categories = {case["category"] for case in POSITIVE["cases"]}
    assert REQUIRED_POSITIVE_CATEGORIES <= categories


def test_negative_fixture_metadata() -> None:
    assert NEGATIVE["schema"] == NEGATIVE_SCHEMA
    assert NEGATIVE["profile"] == STRICT_PROFILE
    assert NEGATIVE["status"] == "executable"

    ids = [case["id"] for case in NEGATIVE["cases"]]
    assert len(ids) == len(set(ids)), "negative case ids must be unique"

    categories = {case["category"] for case in NEGATIVE["cases"]}
    assert REQUIRED_NEGATIVE_CATEGORIES <= categories


def test_v2_positive_fixture_metadata() -> None:
    assert V2_POSITIVE["schema"] == POSITIVE_SCHEMA
    assert V2_POSITIVE["profile"] == V2_PROFILE
    assert V2_POSITIVE["status"] == "executable"
    assert len(V2_POSITIVE["cases"]) == 24

    ids = [case["id"] for case in V2_POSITIVE["cases"]]
    assert len(ids) == len(set(ids)), "v2 positive case ids must be unique"

    categories = {case["category"] for case in V2_POSITIVE["cases"]}
    assert REQUIRED_POSITIVE_CATEGORIES <= categories


def test_v2_negative_fixture_metadata() -> None:
    assert V2_NEGATIVE["schema"] == NEGATIVE_SCHEMA
    assert V2_NEGATIVE["profile"] == V2_PROFILE
    assert V2_NEGATIVE["status"] == "executable"
    assert len(V2_NEGATIVE["cases"]) == 9

    ids = [case["id"] for case in V2_NEGATIVE["cases"]]
    assert len(ids) == len(set(ids)), "v2 negative case ids must be unique"

    for case in V2_NEGATIVE["cases"]:
        assert case["expected_error"] in ALLOWED_ERROR_CLASSES
        assert case["category"] == case["expected_error"]


def test_numeric_parity_fixture_metadata() -> None:
    assert NUMERIC_PARITY["schema"] == NUMERIC_PARITY_SCHEMA
    assert NUMERIC_PARITY["profile"] == NUMERIC_PARITY_PROFILE
    assert NUMERIC_PARITY["status"] == "frozen"
    assert len(NUMERIC_PARITY["cases"]) == 1

    case = NUMERIC_PARITY["cases"][0]
    assert case["id"] == "parity-1e-6-legacy-vs-strict"
    assert case["input_utf8_hex"] == "7b226e223a31652d367d"
    assert case["legacy_expected"] == {
        "go": "7b226e223a302e3030303030317d",
        "python": "7b226e223a31652d30367d",
        "typescript": "7b226e223a302e3030303030317d",
    }
    assert case["strict_expected"] == {
        "go": "7b226e223a302e3030303030317d",
        "python": "7b226e223a302e3030303030317d",
        "typescript": "7b226e223a302e3030303030317d",
    }
    assert case["notes"]


def test_numeric_parity_case_values() -> None:
    case = NUMERIC_PARITY["cases"][0]
    value = json.loads(bytes.fromhex(case["input_utf8_hex"]).decode("utf-8"))

    assert canonicalize_value(value).hex() == case["legacy_expected"]["python"]
    assert (
        canonicalize_strict_raw_v2(bytes.fromhex(case["input_utf8_hex"])).hex()
        == case["strict_expected"]["python"]
    )


def test_legacy_fixture_metadata() -> None:
    assert LEGACY["schema"] == LEGACY_SCHEMA
    assert LEGACY["profile"] == LEGACY_PROFILE
    assert LEGACY["status"] == "frozen"

    ids = [case["id"] for case in LEGACY["raw_cases"]]
    ids += [case["id"] for case in LEGACY["value_cases"]]
    assert len(ids) == len(set(ids)), "legacy case ids must be unique"


def test_profile_selection_fixture_metadata() -> None:
    assert PROFILE_SELECTION["schema"] == PROFILE_SELECTION_SCHEMA
    assert PROFILE_SELECTION["profiles"]["legacy"] == LEGACY_PROFILE
    assert PROFILE_SELECTION["profiles"]["strict"] == STRICT_PROFILE
    assert PROFILE_SELECTION["manifest_field"] == "canonical_profile"
    assert PROFILE_SELECTION["manifest_version_strict"] == 4

    ids = [case["id"] for case in PROFILE_SELECTION["cases"]]
    assert len(ids) == len(set(ids)), "profile selection case ids must be unique"
    for case in PROFILE_SELECTION["cases"]:
        assert case["current_reader"]
        assert case["strict_reader"]


def test_case_ids_are_unique_across_fixtures() -> None:
    ids = [case["id"] for case in POSITIVE["cases"]]
    ids += [case["id"] for case in NEGATIVE["cases"]]
    ids += [case["id"] for case in LEGACY["raw_cases"]]
    ids += [case["id"] for case in LEGACY["value_cases"]]
    ids += [case["id"] for case in PROFILE_SELECTION["cases"]]
    ids += [case["id"] for case in NUMERIC_PARITY["cases"]]
    assert len(ids) == len(set(ids))


@pytest.mark.parametrize("case", POSITIVE["cases"], ids=lambda case: case["id"])
def test_strict_positive_vectors(case: dict) -> None:
    input_bytes = bytes.fromhex(case["input_utf8_hex"])
    canonical_bytes = bytes.fromhex(case["canonical_utf8_hex"])

    # Decoding as UTF-8 asserts the fixture bytes are valid UTF-8.
    input_text = input_bytes.decode("utf-8")
    canonical_text = canonical_bytes.decode("utf-8")

    value = json.loads(input_text)
    json.loads(canonical_text)

    got = canonicalize_value(value).hex()
    diverges = got != case["canonical_utf8_hex"]
    assert diverges == case["legacy_divergence"]["python"], (
        f"legacy_divergence.python = {case['legacy_divergence']['python']}, "
        f"got {got}, strict canonical {case['canonical_utf8_hex']}"
    )


@pytest.mark.parametrize("case", NEGATIVE["cases"], ids=lambda case: case["id"])
def test_strict_negative_vectors(case: dict) -> None:
    # Fixture-integrity check: the strict conformance module executes these
    # cases against the strict parser and asserts the reported error class.
    bytes.fromhex(case["input_utf8_hex"])
    assert case["expected_error"] in ALLOWED_ERROR_CLASSES
    assert case["category"] in REQUIRED_NEGATIVE_CATEGORIES


def test_legacy_raw_cases_are_schema_only() -> None:
    # Raw legacy cases are Go-only fixtures. Python validates their schema but
    # does not execute them.
    for case in LEGACY["raw_cases"]:
        assert case["id"]
        bytes.fromhex(case["input_utf8_hex"])
        has_canonical = "go_canonical_utf8_hex" in case
        has_error = "go_error" in case
        assert has_canonical != has_error, (
            f"{case['id']} must declare exactly one Go outcome"
        )


@pytest.mark.parametrize("case", LEGACY["value_cases"], ids=lambda case: case["id"])
def test_legacy_value_cases(case: dict) -> None:
    value = json.loads(case["value_json"])
    got = canonicalize_value(value).hex()
    assert got == case["expected"]["python"]

    if case["equal_across_languages"]:
        expected = case["expected"]
        assert expected["go"] == expected["python"] == expected["typescript"]
