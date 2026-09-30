"""Canonical envelope drift guard for the Python SDK.

The Python Event reconstruction is a fixed-field representation. This test
fails if the normative envelope schema gains a recognised canonical field that
the Python representation does not preserve. It is a test-time drift detector,
not runtime schema-driven passthrough.
"""

from __future__ import annotations

import json
from pathlib import Path

from atb import Acquisition, AcquisitionCheckpoint, Event, normalize_acquisition

REPO_ROOT = Path(__file__).resolve().parents[3]
SCHEMA_PATH = REPO_ROOT / "schemas" / "event.v1.json"

FULLY_POPULATED_ACQUISITION = {
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


def _schema() -> dict:
    return json.loads(SCHEMA_PATH.read_text(encoding="utf-8"))


def test_event_represents_every_schema_envelope_property() -> None:
    schema = _schema()
    event = Event(
        seq=1,
        prev_hash="0" * 64,
        type="dev.session",
        data={},
        hash_algo="sha256",
        actor_id="actor-1",
        org_id="org-1",
        workspace_id="ws-1",
        timestamp="2026-01-01T10:00:00Z",
        trace_id="0" * 32,
        span_id="0" * 16,
        parent_span_id="0" * 16,
        acquisition=Acquisition(
            **{
                **FULLY_POPULATED_ACQUISITION,
                "checkpoint": AcquisitionCheckpoint(
                    **FULLY_POPULATED_ACQUISITION["checkpoint"]
                ),
            }
        ),
    )
    represented = sorted(event.to_dict().keys())
    assert represented == sorted(schema["properties"].keys())


def test_acquisition_represents_every_schema_def_property() -> None:
    schema = _schema()
    normalized = normalize_acquisition(FULLY_POPULATED_ACQUISITION)
    assert normalized is not None
    assert sorted(normalized.keys()) == sorted(
        schema["$defs"]["acquisition"]["properties"].keys()
    )


def test_checkpoint_represents_every_schema_def_property() -> None:
    schema = _schema()
    normalized = normalize_acquisition(FULLY_POPULATED_ACQUISITION)
    assert normalized is not None
    assert "checkpoint" in normalized
    assert sorted(normalized["checkpoint"].keys()) == sorted(
        schema["$defs"]["acquisition_checkpoint"]["properties"].keys()
    )
