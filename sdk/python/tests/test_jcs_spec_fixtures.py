"""Validate the specification-only JCS canonicalisation-profile fixtures.

These fixtures describe two profiles: the frozen legacy profile
``atb-jcs-legacy-1`` and the not-yet-implemented strict profile
``atb-jcs-rfc8785-1``. This module only validates the normative fixtures and
pins the Python legacy canonicaliser against the frozen expectations; it does
not attempt to run the strict negative cases because no strict parser exists.
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from atb.canonicalize import canonicalize as canonicalize_value

JCS_DIR = Path(__file__).resolve().parents[3] / "test" / "jcs"

POSITIVE_PATH = JCS_DIR / "strict_positive_vectors.json"
NEGATIVE_PATH = JCS_DIR / "strict_negative_vectors.json"
LEGACY_PATH = JCS_DIR / "legacy_freeze_vectors.json"
PROFILE_SELECTION_PATH = JCS_DIR / "profile_selection_cases.json"

POSITIVE_SCHEMA = "atb.jcs.strict-positive-vectors.v1"
NEGATIVE_SCHEMA = "atb.jcs.strict-negative-vectors.v1"
LEGACY_SCHEMA = "atb.jcs.legacy-freeze-vectors.v1"
PROFILE_SELECTION_SCHEMA = "atb.jcs.profile-selection-cases.v1"

STRICT_PROFILE = "atb-jcs-rfc8785-1"
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
LEGACY = _load(LEGACY_PATH)
PROFILE_SELECTION = _load(PROFILE_SELECTION_PATH)


def test_positive_fixture_metadata() -> None:
    assert POSITIVE["schema"] == POSITIVE_SCHEMA
    assert POSITIVE["profile"] == STRICT_PROFILE
    assert POSITIVE["status"] == "specification-only"

    ids = [case["id"] for case in POSITIVE["cases"]]
    assert len(ids) == len(set(ids)), "positive case ids must be unique"

    categories = {case["category"] for case in POSITIVE["cases"]}
    assert REQUIRED_POSITIVE_CATEGORIES <= categories


def test_negative_fixture_metadata() -> None:
    assert NEGATIVE["schema"] == NEGATIVE_SCHEMA
    assert NEGATIVE["profile"] == STRICT_PROFILE
    assert NEGATIVE["status"] == "specification-only"

    ids = [case["id"] for case in NEGATIVE["cases"]]
    assert len(ids) == len(set(ids)), "negative case ids must be unique"

    categories = {case["category"] for case in NEGATIVE["cases"]}
    assert REQUIRED_NEGATIVE_CATEGORIES <= categories


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
    # Validate the payload decodes as hex, but do not attempt strict parsing:
    # no strict implementation exists yet.
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
