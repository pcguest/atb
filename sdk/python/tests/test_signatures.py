from __future__ import annotations

from datetime import datetime
from pathlib import Path
import os
import subprocess

from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

from atb import Bundle


def test_local_signing_records_provenance_fields(tmp_path: Path) -> None:
    bundle = Bundle()
    bundle.append("ai.tool.exec", {"ok": True})
    bundle_path = tmp_path / "bundle.atb"
    bundle.save(bundle_path)

    record = bundle.sign_local(_private_key_pem(), bundle_path)
    data = record.event["data"]

    assert data["backend"] == "local"
    assert data["key_id"] == ""
    assert _is_rfc3339_utc(data["signed_at"])
    report = bundle.verify()
    assert report["signatures"][0]["backend"] == "local"
    assert report["signatures"][0]["valid"] is True


def test_verify_output_includes_go_signed_bundle_signatures(tmp_path: Path) -> None:
    repo_root = Path(__file__).resolve().parents[3]
    bundle = Bundle()
    bundle.append("ai.tool.exec", {"source": "python-fixture"})
    bundle_path = tmp_path / "bundle.atb"
    key_path = tmp_path / "atb-key.pem"
    go_cache = tmp_path / "gocache"
    go_cache.mkdir()
    bundle.save(bundle_path)
    key_path.write_bytes(_private_key_pem())

    subprocess.run(
        [
            "go",
            "run",
            "./cmd/atb",
            "sign",
            "--bundle",
            str(bundle_path),
            "--key",
            str(key_path),
        ],
        cwd=repo_root,
        env={**os.environ, "GOCACHE": str(go_cache)},
        check=True,
        capture_output=True,
        text=True,
    )

    loaded = Bundle.load(bundle_path)
    report = loaded.verify()

    assert len(report["signatures"]) == 1
    signature = report["signatures"][0]
    assert signature["backend"] == "local"
    assert signature["key_id"] == ""
    assert _is_rfc3339_utc(signature["signed_at"])
    assert signature["pubkey"]
    assert signature["bundle_hash"]
    assert signature["valid"] is True


def _private_key_pem() -> bytes:
    private_key = Ed25519PrivateKey.generate()
    return private_key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.PKCS8,
        encryption_algorithm=serialization.NoEncryption(),
    )


def _is_rfc3339_utc(value: str) -> bool:
    parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
    return value.endswith("Z") and parsed.tzinfo is not None
