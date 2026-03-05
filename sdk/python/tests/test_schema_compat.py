"""Schema compatibility tests for optional multi-tenant event fields."""

from __future__ import annotations

import json
from pathlib import Path
from tempfile import TemporaryDirectory

from atb import Bundle, Event
from atb.canonicalize import canonicalize
from atb.hash import GENESIS_HASH, compute_hash


def test_canonical_json_backward_compat() -> None:
    legacy_event = {
        "seq": 1,
        "prev_hash": GENESIS_HASH,
        "type": "schema.compat",
        "data": {"x": 1},
    }
    new_event = Event(
        seq=1,
        prev_hash=GENESIS_HASH,
        type="schema.compat",
        data={"x": 1},
        actor_id=None,
        org_id=None,
        workspace_id=None,
    )
    legacy_json = canonicalize(legacy_event)
    new_json = canonicalize(new_event.to_dict())
    assert legacy_json == new_json


def test_append_with_none_or_empty_identity_is_backward_compatible() -> None:
    baseline = Bundle()
    baseline.append("schema.compat", {"x": 1})

    with_none = Bundle()
    with_none.append(
        "schema.compat",
        {"x": 1},
        actor_id=None,
        org_id=None,
        workspace_id=None,
    )

    with_empty = Bundle()
    with_empty.append(
        "schema.compat",
        {"x": 1},
        actor_id="",
        org_id="   ",
        workspace_id="",
    )

    assert baseline.records[0].event == with_none.records[0].event
    assert baseline.records[0].event == with_empty.records[0].event
    assert baseline.records[0].hash == with_none.records[0].hash
    assert baseline.records[0].hash == with_empty.records[0].hash


def test_append_with_identity_fields_changes_hash_and_verifies() -> None:
    baseline = Bundle()
    baseline.append("schema.compat", {"x": 1})

    with_identity = Bundle()
    with_identity.append(
        "schema.compat",
        {"x": 1},
        actor_id="paddy",
        org_id="pcguest",
        workspace_id="local",
    )

    assert with_identity.records[0].hash != baseline.records[0].hash
    with_identity.verify()
    assert with_identity.records[0].event["actor_id"] == "paddy"
    assert with_identity.records[0].event["org_id"] == "pcguest"
    assert with_identity.records[0].event["workspace_id"] == "local"


def test_legacy_bundle_verifies_with_new_sdk() -> None:
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
    assert "actor_id" not in loaded.records[0].event
    assert "org_id" not in loaded.records[0].event
    assert "workspace_id" not in loaded.records[0].event
