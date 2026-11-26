"""Regression tests for recorded seq/prev_hash chain-field validation."""

from __future__ import annotations

import json

import pytest

from atb.bundle import Bundle
from atb.exceptions import ATBVerificationError
from atb.hash import GENESIS_HASH, compute_hash


def _make_bundle(with_manifest: bool) -> Bundle:
    bundle = Bundle() if with_manifest else Bundle(records=[])
    bundle.append("ai.tool.exec", {"ok": True})
    bundle.append("ai.tool.exec", {"ok": False})
    return bundle


def _target_index(with_manifest: bool) -> int:
    return 1 if with_manifest else 0


@pytest.mark.parametrize(
    "with_manifest", [True, False], ids=["manifest", "no-manifest"]
)
def test_valid_bundle_still_verifies(with_manifest: bool) -> None:
    bundle = _make_bundle(with_manifest)

    assert bundle.verify()["signatures"] == []


@pytest.mark.parametrize(
    "with_manifest", [True, False], ids=["manifest", "no-manifest"]
)
@pytest.mark.parametrize("field", ["seq", "prev_hash"])
def test_mutated_chain_field_is_rejected(with_manifest: bool, field: str) -> None:
    bundle = _make_bundle(with_manifest)
    event = bundle.records[_target_index(with_manifest)].event
    if field == "seq":
        event["seq"] = 999
    else:
        event["prev_hash"] = "f" * 64

    # A plain mutation also breaks the stored record hash: the pre-fix
    # verifier would report a hash mismatch, not a chain-field mismatch. The
    # duplicate-key test below is the actual acceptance delta.
    expected = "sequence mismatch" if field == "seq" else "prev_hash mismatch"
    with pytest.raises(ATBVerificationError, match=expected):
        bundle.verify()


@pytest.mark.parametrize(
    "with_manifest", [True, False], ids=["manifest", "no-manifest"]
)
@pytest.mark.parametrize("field", ["seq", "prev_hash"])
def test_missing_chain_field_is_rejected(with_manifest: bool, field: str) -> None:
    bundle = _make_bundle(with_manifest)
    event = bundle.records[_target_index(with_manifest)].event
    event.pop(field)

    expected = "sequence mismatch" if field == "seq" else "prev_hash mismatch"
    with pytest.raises(ATBVerificationError, match=expected):
        bundle.verify()


@pytest.mark.parametrize(
    "with_manifest", [True, False], ids=["manifest", "no-manifest"]
)
def test_mutating_both_chain_fields_is_rejected(with_manifest: bool) -> None:
    bundle = _make_bundle(with_manifest)
    event = bundle.records[_target_index(with_manifest)].event
    event["seq"] = 999
    event["prev_hash"] = "f" * 64

    with pytest.raises(ATBVerificationError, match="sequence mismatch"):
        bundle.verify()


def test_genesis_prev_hash_still_required() -> None:
    bundle = _make_bundle(with_manifest=True)
    assert bundle.records[0].event["prev_hash"] == GENESIS_HASH

    bundle.records[0].event["prev_hash"] = "f" * 64

    with pytest.raises(ATBVerificationError, match="prev_hash mismatch"):
        bundle.verify()


@pytest.mark.parametrize(
    ("field", "duplicate_value", "message"),
    [
        ("seq", 999, "sequence mismatch"),
        ("prev_hash", "f" * 64, "prev_hash mismatch"),
    ],
)
def test_duplicate_chain_key_is_the_acceptance_delta(
    tmp_path, field: str, duplicate_value: object, message: str
) -> None:
    """JSON parsers keep the last duplicate key; the hash uses the expected one.

    The pre-fix verifier overwrote the recorded value with the expected value
    and recomputed the stored hash, so it accepted this. The fixed verifier
    compares the parsed (last) value first and rejects it.
    """
    bundle = _make_bundle(with_manifest=True)
    path = tmp_path / "duplicate-key.atb"
    bundle.save(path)

    lines = path.read_text(encoding="utf-8").splitlines()
    record = json.loads(lines[1])
    event = dict(record["event"])
    expected_prev_hash = event["prev_hash"]
    expected_seq = 1
    event["seq"] = expected_seq
    expected_hash = compute_hash(event, expected_prev_hash)
    event_json = json.dumps(event, separators=(",", ":"))

    if field == "seq":
        needle = f'"seq":{expected_seq}'
        replacement = f'{needle},"seq":{duplicate_value}'
    else:
        needle = f'"prev_hash":"{expected_prev_hash}"'
        replacement = f'{needle},"prev_hash":"{duplicate_value}"'
    tampered_event_json = event_json.replace(needle, replacement, 1)
    assert tampered_event_json != event_json
    lines[1] = '{"event":' + tampered_event_json + ',"hash":"' + expected_hash + '"}'
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")

    loaded = Bundle.load(path)
    assert loaded.records[1].event[field] == duplicate_value

    with pytest.raises(ATBVerificationError, match=message):
        loaded.verify()
