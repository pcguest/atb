"""Contract tests for atb.bundle.append_events_in_memory."""

from __future__ import annotations

import base64
import hashlib
import json
from pathlib import Path

import pytest
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec, utils
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

from atb.bundle import (
    SIGNATURE_EVENT_TYPE,
    Bundle,
    BundleResourceLimitError,
    UnsupportedAlgorithmError,
    _save_atomic,
    append_events_in_memory,
)


def _make_event(seq: int) -> dict:
    return {
        "type": "ai.request.received",
        "data": {
            "request_id": f"req-test-{seq:03d}",
            "actor_id_hash": "sha256-test",
            "purpose_tag": "contract_test",
        },
        "timestamp": f"2026-04-26T12:00:{seq:02d}Z",
    }


def test_append_events_in_memory_happy():
    bundle = Bundle()
    start = len(bundle.records)  # manifest record
    count = append_events_in_memory(
        bundle, [_make_event(1), _make_event(2), _make_event(3)]
    )
    assert count == 3
    # 1 manifest + 3 appended
    assert len(bundle.records) == start + 3 == 4


def test_append_events_in_memory_empty():
    bundle = Bundle()
    start = len(bundle.records)
    count = append_events_in_memory(bundle, [])
    assert count == 0
    assert len(bundle.records) == start


def test_append_events_in_memory_invalid_type():
    bundle = Bundle()
    start = len(bundle.records)
    bad = {"type": "INVALID TYPE", "data": {"x": 1}}
    with pytest.raises(ValueError):
        append_events_in_memory(bundle, [bad])
    assert len(bundle.records) == start


def test_append_events_in_memory_nil_bundle_rejected():
    with pytest.raises(ValueError):
        append_events_in_memory(None, [_make_event(1)])


def test_append_events_in_memory_partial_failure_pins_behaviour():
    bundle = Bundle()
    start = len(bundle.records)
    good = _make_event(1)
    bad = {"type": "INVALID TYPE", "data": {}}
    with pytest.raises(ValueError):
        append_events_in_memory(bundle, [good, bad])
    # Stops at first error; the good event remains in memory.
    assert len(bundle.records) == start + 1


def test_save_atomic_writes_payload(tmp_path):
    path = tmp_path / "bundle.atb"
    data = b"first\nsecond\n"

    _save_atomic(path, data)

    assert path.read_bytes() == data


def test_load_rejects_byte_and_record_limits(tmp_path):
    path = tmp_path / "bundle.atb"
    bundle = Bundle()
    bundle.append("ai.tool.exec", {"ok": True})
    bundle.save(path)

    with pytest.raises(BundleResourceLimitError, match="maximum size"):
        Bundle.load(path, max_bytes=path.stat().st_size - 1)

    with pytest.raises(BundleResourceLimitError, match="record count"):
        Bundle.load(path, max_records=1)


def test_save_atomic_failure_keeps_original_and_removes_temp(tmp_path, monkeypatch):
    path = tmp_path / "bundle.atb"
    original = b"original\n"
    path.write_bytes(original)

    def fail_fsync(_fd: int) -> None:
        raise OSError("simulated fsync failure")

    monkeypatch.setattr("os.fsync", fail_fsync)

    with pytest.raises(OSError):
        _save_atomic(path, b"replacement\n")

    assert path.read_bytes() == original
    assert list(tmp_path.glob("*.atb.tmp")) == []


def test_ed25519_signature_verification_happy_path(tmp_path):
    private_key = Ed25519PrivateKey.generate()
    bundle_path = _signed_bundle_path(tmp_path, "ed25519", private_key)

    loaded = Bundle.load(bundle_path)

    assert loaded.verify()["signatures"][0]["valid"] is True


def test_ecdsa_p256_signature_verification_happy_path(tmp_path):
    private_key = ec.generate_private_key(ec.SECP256R1())
    bundle_path = _signed_bundle_path(tmp_path, "ecdsa-p256", private_key)

    loaded = Bundle.load(bundle_path)

    assert loaded.verify()["signatures"][0]["valid"] is True


@pytest.mark.parametrize("algorithm", ["ed25519", "ecdsa-p256"])
def test_signature_verification_returns_false_for_tampered_payload(tmp_path, algorithm):
    private_key = (
        Ed25519PrivateKey.generate()
        if algorithm == "ed25519"
        else ec.generate_private_key(ec.SECP256R1())
    )
    bundle_path = _signed_bundle_path(tmp_path, algorithm, private_key)
    raw = bundle_path.read_text(encoding="utf-8")
    bundle_path.write_text(
        raw.replace('"ok": true', '"ok": false', 1), encoding="utf-8"
    )

    loaded = Bundle.load(bundle_path)

    assert loaded.signatures()[0]["valid"] is False


def test_unknown_signature_algorithm_raises(tmp_path):
    private_key = Ed25519PrivateKey.generate()
    bundle_path = _signed_bundle_path(tmp_path, "rsa-4096", private_key)

    loaded = Bundle.load(bundle_path)

    with pytest.raises(UnsupportedAlgorithmError):
        loaded.signatures()


def _signed_bundle_path(tmp_path: Path, algorithm: str, private_key) -> Path:
    bundle = Bundle()
    bundle.append("ai.tool.exec", {"ok": True})
    bundle_path = tmp_path / f"{algorithm}.atb"
    bundle.save(bundle_path)

    raw = bundle_path.read_bytes()
    digest = hashlib.sha256(raw).digest()
    signature, public_key = _sign_digest(algorithm, private_key, digest)
    record = bundle.append(
        SIGNATURE_EVENT_TYPE,
        {
            "bundle_hash": digest.hex(),
            "signature": base64.b64encode(signature).decode("ascii"),
            "pubkey": base64.b64encode(public_key).decode("ascii"),
            "algorithm": algorithm,
            "backend": "test",
        },
    )
    encoded_record = json.dumps({"event": record.event, "hash": record.hash}).encode(
        "utf-8"
    )
    payload = raw
    if payload and not payload.endswith(b"\n"):
        payload += b"\n"
    bundle_path.write_bytes(payload + encoded_record + b"\n")
    return bundle_path


def _sign_digest(algorithm: str, private_key, digest: bytes) -> tuple[bytes, bytes]:
    if algorithm in {"ed25519", "rsa-4096"}:
        signature = private_key.sign(digest)
        public_key = private_key.public_key().public_bytes(
            encoding=serialization.Encoding.Raw,
            format=serialization.PublicFormat.Raw,
        )
        return signature, public_key

    if algorithm == "ecdsa-p256":
        signature = private_key.sign(
            digest,
            ec.ECDSA(utils.Prehashed(hashes.SHA256())),
        )
        public_key = private_key.public_key().public_bytes(
            encoding=serialization.Encoding.X962,
            format=serialization.PublicFormat.UncompressedPoint,
        )
        return signature, public_key

    raise AssertionError(f"unsupported test algorithm {algorithm}")
