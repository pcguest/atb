"""Invariant/property-style tests for the ATB Python SDK."""

from __future__ import annotations

from copy import deepcopy
from pathlib import Path
from tempfile import TemporaryDirectory

from atb import Bundle
from atb.exceptions import ATBVerificationError
from atb.hash import GENESIS_HASH


def _sample_events() -> list[tuple[str, dict]]:
    return [
        (
            "agent.session",
            {"id": "prop-001", "flags": {"offline": True, "zk": True}},
        ),
        (
            "agent.decision",
            {"choice": "use-go-cli", "reason": "deterministic tooling"},
        ),
        (
            "agent.snapshot",
            {"gate": "pass", "metrics": {"duration_ms": 123}},
        ),
    ]


def _build_bundle() -> Bundle:
    bundle = Bundle(goal="property-tests")
    for event_type, data in _sample_events():
        bundle.append(event_type, deepcopy(data))
    return bundle


def test_hash_chain_is_deterministic_for_identical_events() -> None:
    first = _build_bundle()
    second = _build_bundle()
    assert [record.hash for record in first.records] == [
        record.hash for record in second.records
    ]


def test_verify_is_idempotent_and_non_mutating() -> None:
    bundle = _build_bundle()
    before_events = [deepcopy(record.event) for record in bundle.records]
    before_hashes = [record.hash for record in bundle.records]

    bundle.verify()
    bundle.verify()

    assert [record.event for record in bundle.records] == before_events
    assert [record.hash for record in bundle.records] == before_hashes


def test_save_load_roundtrip_preserves_head_hash() -> None:
    bundle = _build_bundle()
    head_hash = bundle.records[-1].hash

    with TemporaryDirectory() as tmp:
        path = Path(tmp) / "bundle.atb"
        bundle.save(path)
        loaded = Bundle.load(path)

    loaded.verify()
    assert loaded.records[-1].hash == head_hash
    assert len(loaded.records) == len(bundle.records)


def test_tampering_is_detected() -> None:
    bundle = _build_bundle()
    bundle.records[1].event["data"]["choice"] = "tampered"

    try:
        bundle.verify()
        raise AssertionError("expected ATBVerificationError for tampered bundle")
    except ATBVerificationError:
        pass


def test_prev_hash_linkage_is_continuous() -> None:
    bundle = _build_bundle()
    assert bundle.records[0].event["prev_hash"] == GENESIS_HASH
    for i in range(1, len(bundle.records)):
        assert bundle.records[i].event["prev_hash"] == bundle.records[i - 1].hash


def test_hash_is_stable_when_dict_key_order_changes() -> None:
    first = Bundle()
    second = Bundle()
    first.append("canonical.order", {"a": 1, "b": 2, "nested": {"x": 3, "y": 4}})
    second.append("canonical.order", {"nested": {"y": 4, "x": 3}, "b": 2, "a": 1})

    assert first.records[0].hash == second.records[0].hash
