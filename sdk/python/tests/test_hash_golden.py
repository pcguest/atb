"""Cross-language pinned-hash test for the Python SDK.

Loads internal/hash/testdata/golden_corpus.json (produced from the Go runtime,
the source of truth) and asserts that the Python SDK's canonicaliser and
hash function produce identical output.

A failure here means the Python SDK has diverged from Go. Bundles signed in
Go would no longer re-verify in Python — a breaking schema change. Bump
ManifestVersion, regenerate the corpus, and update both sides in lockstep.
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from atb.canonicalize import canonicalize
from atb.hash import compute_hash

CORPUS_PATH = (
    Path(__file__).resolve().parents[3]
    / "internal"
    / "hash"
    / "testdata"
    / "golden_corpus.json"
)


def _load_corpus() -> list[dict]:
    if not CORPUS_PATH.is_file():
        pytest.fail(f"golden corpus missing at {CORPUS_PATH}")
    return json.loads(CORPUS_PATH.read_text())


@pytest.mark.parametrize("entry", _load_corpus(), ids=lambda e: e["description"])
def test_hash_golden_corpus(entry: dict) -> None:
    event = entry["event"]
    expected_canonical = bytes.fromhex(entry["canonical_hex"])
    expected_hash = entry["hash_hex"]

    got_canonical = canonicalize(event)
    assert got_canonical == expected_canonical, (
        f"canonical bytes changed — this is a breaking schema change, "
        f"bump ManifestVersion and update the golden corpus\n"
        f"  expected: {expected_canonical.hex()}\n"
        f"  got:      {got_canonical.hex()}"
    )

    got_hash = compute_hash(event, event["prev_hash"])
    assert got_hash == expected_hash, (
        f"hash changed — this is a breaking schema change, "
        f"bump ManifestVersion and update the golden corpus\n"
        f"  expected: {expected_hash}\n"
        f"  got:      {got_hash}"
    )
