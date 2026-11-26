"""Regression tests for the unsupported manifest version guard."""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from atb.bundle import (
    MANIFEST_EVENT_TYPE,
    MANIFEST_VERSION,
    MAX_MANIFEST_VERSION,
    SIGNATURE_EVENT_TYPE,
    Bundle,
    Record,
    UnsupportedManifestVersionError,
)
from atb.canonicalize import canonicalize
from atb.encrypt import encrypt_raw
from atb.exceptions import ATBError, ATBVerificationError
from atb.hash import GENESIS_HASH, compute_hash

CREATED_AT = "2026-03-27T12:00:00Z"
BUNDLE_ID = "a" * 32


def _chain(events: list[dict]) -> list[Record]:
    has_manifest = bool(events) and events[0]["type"] == MANIFEST_EVENT_TYPE
    prev = GENESIS_HASH
    records: list[Record] = []
    for index, raw_event in enumerate(events):
        event = dict(raw_event)
        if has_manifest and index == 0:
            event["seq"] = 0
        elif has_manifest:
            event["seq"] = index
        else:
            event["seq"] = index + 1
        event["prev_hash"] = prev
        event.setdefault("hash_algo", "sha256")
        record_hash = compute_hash(event, prev)
        records.append(Record(event=event, hash=record_hash))
        prev = record_hash
    return records


def _manifest(data: object) -> dict:
    return {"type": MANIFEST_EVENT_TYPE, "data": data}


def _event() -> dict:
    return {"type": "ai.tool.exec", "data": {"ok": True}}


def _signature() -> dict:
    return {
        "type": SIGNATURE_EVENT_TYPE,
        "data": {
            "bundle_hash": "0" * 64,
            "signature": "not-a-signature",
            "pubkey": "not-a-pubkey",
            "backend": "test",
        },
    }


def _path(tmp_path: Path, records: list[Record], name: str = "bundle.atb") -> Path:
    path = tmp_path / name
    Bundle(records=records).save(path)
    return path


@pytest.mark.parametrize(
    "manifest_data",
    [
        pytest.param(
            json.dumps(
                {"version": "1", "created_at": CREATED_AT, "bundle_id": BUNDLE_ID}
            ),
            id="v1-string",
        ),
        pytest.param(
            json.dumps({"created_at": CREATED_AT, "bundle_id": BUNDLE_ID}),
            id="v1-absent-version",
        ),
        pytest.param(
            {"version": 2, "created_at": CREATED_AT, "bundle_id": BUNDLE_ID},
            id="v2-structured",
        ),
        pytest.param(
            {"version": 3, "created_at": CREATED_AT, "bundle_id": BUNDLE_ID},
            id="v3-structured",
        ),
        pytest.param(
            {
                "version": 3,
                "created_at": CREATED_AT,
                "bundle_id": BUNDLE_ID,
                "canonical_profile": "atb.profile.agent",
            },
            id="v3-canonical-profile",
        ),
    ],
)
def test_legacy_manifests_are_accepted(
    tmp_path: Path, manifest_data: object
) -> None:
    path = _path(tmp_path, _chain([_manifest(manifest_data), _event()]))

    loaded = Bundle.load(path)

    assert loaded.verify()["signatures"] == []


def test_absent_manifest_is_accepted(tmp_path: Path) -> None:
    path = _path(tmp_path, _chain([_event()]))

    loaded = Bundle.load(path)

    assert loaded.verify()["signatures"] == []


@pytest.mark.parametrize(
    "version",
    [
        4,
        4.0,
        4.5,
        3.5,
        "4",
        "4.0",
        " 4",
        "4abc",
        "0004",
        "9" * 30,
    ],
    ids=[
        "int",
        "integral-float",
        "fractional-float",
        "fraction-just-above-max",
        "string",
        "fraction-string",
        "leading-whitespace",
        "trailing-junk",
        "zero-padded",
        "oversized-digits",
    ],
)
def test_v4_load_is_rejected_with_distinct_error(
    tmp_path: Path, version: object
) -> None:
    # The record hash is valid for the legacy chain, so rejection must happen
    # before verification rather than as a hash mismatch.
    records = _chain(
        [
            _manifest(
                {"version": version, "created_at": CREATED_AT, "bundle_id": BUNDLE_ID}
            ),
            _event(),
        ]
    )
    path = _path(tmp_path, records)

    with pytest.raises(UnsupportedManifestVersionError) as excinfo:
        Bundle.load(path)

    assert excinfo.value.version == 4
    assert str(excinfo.value).startswith(
        "atb: unsupported manifest version 4: "
        "this bundle requires a newer version of the SDK"
    )
    assert not isinstance(excinfo.value, ATBVerificationError)


def test_non_finite_manifest_version_is_rejected(tmp_path: Path) -> None:
    # Python's json module decodes Infinity/NaN unless configured otherwise;
    # the guard must fail closed rather than treat them as legacy.
    event = {
        "seq": 0,
        "prev_hash": GENESIS_HASH,
        "type": MANIFEST_EVENT_TYPE,
        "hash_algo": "sha256",
        "data": {
            "version": float("inf"),
            "created_at": CREATED_AT,
            "bundle_id": BUNDLE_ID,
        },
    }
    path = _path(tmp_path, [Record(event=event, hash="0" * 64)])

    with pytest.raises(UnsupportedManifestVersionError) as excinfo:
        Bundle.load(path)

    assert excinfo.value.version == MAX_MANIFEST_VERSION + 1


def test_unsupported_version_is_catchable_as_atb_error(tmp_path: Path) -> None:
    path = _path(
        tmp_path,
        _chain(
            [
                _manifest(
                    {
                        "version": 4,
                        "created_at": CREATED_AT,
                        "bundle_id": BUNDLE_ID,
                    }
                ),
                _event(),
            ]
        ),
    )

    with pytest.raises(ATBError) as excinfo:
        Bundle.load(path)

    assert isinstance(excinfo.value, UnsupportedManifestVersionError)
    assert isinstance(excinfo.value, ValueError)
    assert not isinstance(excinfo.value, ATBVerificationError)


def test_v4_decrypt_is_rejected_before_legacy_verification() -> None:
    records = _chain(
        [
            _manifest(
                {"version": 4, "created_at": CREATED_AT, "bundle_id": BUNDLE_ID}
            ),
            _event(),
        ]
    )
    payload = {
        "head_hash": records[-1].hash,
        "records": [{"event": record.event, "hash": record.hash} for record in records],
    }
    encrypted = encrypt_raw(canonicalize(payload), "password")

    with pytest.raises(UnsupportedManifestVersionError) as excinfo:
        Bundle.decrypt("password", encrypted)

    assert excinfo.value.version == 4
    assert not isinstance(excinfo.value, ATBVerificationError)


def test_max_manifest_version_matches_go() -> None:
    assert MAX_MANIFEST_VERSION == 3


def test_v4_with_signature_rejects_load_and_signatures(tmp_path: Path) -> None:
    records = _chain(
        [
            _manifest(
                {"version": 4, "created_at": CREATED_AT, "bundle_id": BUNDLE_ID}
            ),
            _event(),
            _signature(),
        ]
    )

    with pytest.raises(UnsupportedManifestVersionError):
        Bundle.load(_path(tmp_path, records))

    in_memory = Bundle(records=records)
    with pytest.raises(UnsupportedManifestVersionError):
        in_memory.signatures()
    with pytest.raises(UnsupportedManifestVersionError):
        in_memory.verify()


def test_in_memory_v4_verify_and_encrypt_raise() -> None:
    bundle = Bundle(
        records=_chain(
            [
                _manifest(
                    {"version": 4, "created_at": CREATED_AT, "bundle_id": BUNDLE_ID}
                ),
                _event(),
            ]
        )
    )

    with pytest.raises(UnsupportedManifestVersionError):
        bundle.verify()
    with pytest.raises(UnsupportedManifestVersionError):
        bundle.encrypt("password")


def test_writer_still_emits_manifest_version_one(tmp_path: Path) -> None:
    path = tmp_path / "writer.atb"
    Bundle().save(path)

    first = json.loads(path.read_text(encoding="utf-8").splitlines()[0])

    assert MANIFEST_VERSION == "1"
    assert json.loads(first["event"]["data"])["version"] == "1"
