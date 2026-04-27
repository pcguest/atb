from __future__ import annotations

import json
from pathlib import Path

from atb import Bundle


def test_manifest_data_is_double_encoded(tmp_path: Path) -> None:
    """
    The manifest record's data field is a JSON-encoded string, not an object.
    This test pins that behaviour so a refactor cannot silently change it.
    """
    bundle = Bundle()
    bundle_path = tmp_path / "bundle.atb"
    bundle.save(bundle_path)

    first_line = bundle_path.read_text(encoding="utf-8").splitlines()[0]
    record = json.loads(first_line)

    assert isinstance(record["event"]["data"], str)
    manifest = json.loads(record["event"]["data"])
    assert set(manifest) == {"version", "bundle_id", "created_at"}
    assert manifest["version"] == "1"
