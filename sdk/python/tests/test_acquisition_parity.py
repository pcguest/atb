"""Python acquisition envelope parity: construction, append, and verification.

The Python SDK previously preserved acquisition only passively (raw dicts on
load/verify) and dropped it on any typed construction/append path. These tests
pin the repaired append/reconstruction contract and the shared Go/Python/TS
golden vector.
"""

from __future__ import annotations

import base64
import json
from copy import deepcopy
from pathlib import Path
from tempfile import TemporaryDirectory

import pytest

from atb import (
    Acquisition,
    AcquisitionCheckpoint,
    Bundle,
    Event,
    normalize_acquisition,
)
from atb.canonicalize import canonicalize
from atb.hash import GENESIS_HASH, compute_hash

REPO_ROOT = Path(__file__).resolve().parents[3]
VECTORS_PATH = REPO_ROOT / "internal" / "hash" / "testdata" / "canonical_vectors.json"

FULL_ACQUISITION = {
    "mode": "retrospective",
    "source_system": "chatlog",
    "source_record_id": "rec-1",
    "source_timestamp": "2026-01-01T10:00:00Z",
    "acquired_at": "2026-01-02T09:00:00Z",
    "source_digest": "sha256:" + "9" * 64,
    "adapter": "atb.chatlog.generic-jsonl",
    "adapter_version": "1.0.0",
    "checkpoint": {
        "source_system": "chatlog",
        "acquisition_stream": "chatlog.jsonl",
        "position": "42",
        "observed_at": "2026-01-02T09:00:00Z",
        "adapter": "atb.chatlog.generic-jsonl",
        "adapter_version": "1.0.0",
    },
}


def _acquisition_vector() -> dict:
    vectors = json.loads(VECTORS_PATH.read_text(encoding="utf-8"))
    for vector in vectors:
        if "acquisition provenance" in vector["description"]:
            return vector
    raise AssertionError("shared acquisition vector missing from canonical_vectors.json")


# P1 — no-acquisition events retain previous canonical bytes.
def test_no_acquisition_retains_previous_canonical_bytes() -> None:
    legacy = {
        "seq": 1,
        "prev_hash": GENESIS_HASH,
        "type": "dev.session",
        "data": {"x": 1},
    }
    event = Event(seq=1, prev_hash=GENESIS_HASH, type="dev.session", data={"x": 1})
    assert canonicalize(event.to_dict()) == canonicalize(legacy)
    assert "acquisition" not in event.to_dict()


# P2 — valid acquisition survives Event -> append -> serialisation.
def test_acquisition_survives_append() -> None:
    bundle = Bundle(records=[])
    record = bundle.append("dev.session", {"x": 1}, acquisition=FULL_ACQUISITION)
    assert record.event["acquisition"] == FULL_ACQUISITION
    bundle.verify()


# P3 — acquisition survives save -> load -> verify.
def test_acquisition_survives_save_and_load() -> None:
    bundle = Bundle(records=[])
    bundle.append("dev.session", {"x": 1}, acquisition=FULL_ACQUISITION)
    with TemporaryDirectory() as tmp:
        path = Path(tmp) / "bundle.atb"
        bundle.save(path)
        loaded = Bundle.load(path)
        loaded.verify()
    assert loaded.records[-1].event["acquisition"] == FULL_ACQUISITION


# P4 — removing acquisition changes the canonical bytes and hash.
def test_removing_acquisition_changes_hash() -> None:
    with_acq = Bundle(records=[])
    rec_with = with_acq.append("dev.session", {"x": 1}, acquisition=FULL_ACQUISITION)
    without_acq = Bundle(records=[])
    rec_without = without_acq.append("dev.session", {"x": 1})
    assert rec_with.hash != rec_without.hash
    assert canonicalize(rec_with.event) != canonicalize(rec_without.event)


# P5 — acquisition key insertion order does not change canonical output.
def test_acquisition_key_order_is_insensitive() -> None:
    ordered = {k: FULL_ACQUISITION[k] for k in FULL_ACQUISITION}
    reordered = {k: FULL_ACQUISITION[k] for k in reversed(list(FULL_ACQUISITION))}
    assert canonicalize(normalize_acquisition(ordered)) == canonicalize(
        normalize_acquisition(reordered)
    )


# P6 — malformed acquisition fails deterministically.
@pytest.mark.parametrize(
    "bad",
    [
        "chatlog",
        [1, 2],
        {"mode": 5},
        {"checkpoint": "x"},
        {"checkpoint": {"source_system": "chatlog"}},
    ],
)
def test_malformed_acquisition_fails_closed(bad: object) -> None:
    with pytest.raises(TypeError):
        normalize_acquisition(bad)
    bundle = Bundle(records=[])
    with pytest.raises(TypeError):
        bundle.append("dev.session", {}, acquisition=bad)


# P7 — unknown acquisition fields cannot silently enter canonical evidence.
def test_unknown_acquisition_fields_are_dropped() -> None:
    bundle = Bundle(records=[])
    record = bundle.append(
        "dev.session",
        {},
        acquisition={"mode": "retrospective", "unknown_field": "secret"},
    )
    assert record.event["acquisition"] == {"mode": "retrospective"}
    assert "unknown_field" not in canonicalize(record.event).decode("utf-8")


def test_unknown_checkpoint_fields_are_dropped() -> None:
    normalized = normalize_acquisition(
        {
            "checkpoint": {
                "source_system": "chatlog",
                "acquisition_stream": "chatlog.jsonl",
                "position": "1",
                "observed_at": "2026-01-02T09:00:00Z",
                "adapter": "atb.chatlog.generic-jsonl",
                "extra": "nope",
            }
        }
    )
    assert normalized is not None
    assert "extra" not in normalized["checkpoint"]


# P8 — nested checkpoint survives construction and append.
def test_nested_checkpoint_survives() -> None:
    bundle = Bundle(records=[])
    record = bundle.append(
        "dev.session",
        {},
        acquisition=Acquisition(
            mode="retrospective",
            checkpoint=AcquisitionCheckpoint(
                source_system="chatlog",
                acquisition_stream="chatlog.jsonl",
                position="42",
                observed_at="2026-01-02T09:00:00Z",
                adapter="atb.chatlog.generic-jsonl",
            ),
        ),
    )
    assert record.event["acquisition"]["checkpoint"]["position"] == "42"
    assert record.event["acquisition"]["checkpoint"]["source_system"] == "chatlog"


# P9 — Python Event model matches the shared Go/Python/TS acquisition vector.
def test_python_event_matches_shared_vector() -> None:
    vector = _acquisition_vector()
    raw = vector["event"]
    event = Event(
        seq=raw["seq"],
        prev_hash=raw["prev_hash"],
        type=raw["type"],
        data=raw["data"],
        hash_algo=raw.get("hash_algo"),
        timestamp=raw.get("timestamp"),
        acquisition=raw["acquisition"],
    )
    expected_canonical = base64.b64decode(vector["expected_canonical_bytes"])
    assert canonicalize(event.to_dict()) == expected_canonical
    assert compute_hash(event.to_dict(), raw["prev_hash"]) == vector["expected_hash"]


# P10 — historical no-acquisition bundles continue to verify.
def test_historical_bundle_without_acquisition_verifies() -> None:
    legacy_event = {
        "seq": 1,
        "prev_hash": GENESIS_HASH,
        "type": "legacy.test",
        "data": {"x": 1},
    }
    legacy_hash = compute_hash(legacy_event, GENESIS_HASH)
    line = json.dumps({"event": legacy_event, "hash": legacy_hash}) + "\n"
    with TemporaryDirectory() as tmp:
        path = Path(tmp) / "legacy.atb"
        path.write_text(line, encoding="utf-8")
        loaded = Bundle.load(path)
    loaded.verify()
    assert "acquisition" not in loaded.records[0].event


def test_append_without_acquisition_is_backward_compatible() -> None:
    baseline = Bundle(records=[])
    baseline.append("dev.session", {"x": 1})
    explicit_none = Bundle(records=[])
    explicit_none.append("dev.session", {"x": 1}, acquisition=None)
    assert baseline.records[-1].event == explicit_none.records[-1].event
    assert baseline.records[-1].hash == explicit_none.records[-1].hash
    assert "acquisition" not in baseline.records[-1].event


def test_append_events_in_memory_preserves_acquisition() -> None:
    from atb.bundle import append_events_in_memory

    bundle = Bundle(records=[])
    appended = append_events_in_memory(
        bundle,
        [{"type": "dev.session", "data": {"x": 1}, "acquisition": FULL_ACQUISITION}],
    )
    assert appended == 1
    assert bundle.records[-1].event["acquisition"] == FULL_ACQUISITION
    bundle.verify()


def test_append_events_in_memory_accepts_event_instances() -> None:
    from atb.bundle import append_events_in_memory

    bundle = Bundle(records=[])
    appended = append_events_in_memory(
        bundle,
        [
            Event(
                seq=0,
                prev_hash=GENESIS_HASH,
                type="dev.session",
                data={},
                acquisition=deepcopy(FULL_ACQUISITION),
            )
        ],
    )
    assert appended == 1
    assert bundle.records[-1].event["acquisition"] == FULL_ACQUISITION
