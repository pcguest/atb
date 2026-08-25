"""Cross-language canonical-hash golden test for the Python SDK.

Loads internal/hash/testdata/canonical_vectors.json (the same fixture used
by the Go runtime and the TypeScript SDK) and asserts that the Python SDK's
canonicaliser and hash function produce identical output.

This test pins the Python SDK's wire format to the Go source of truth.
Any drift here means the Python SDK has diverged and bundles signed by Go
will not re-verify in Python.
"""

from __future__ import annotations

import base64
import json
from pathlib import Path

import pytest

from atb.canonicalize import canonicalize
from atb.hash import compute_hash
from atb.workflow_common import WorkflowContext

TESTDATA_DIR = Path(__file__).resolve().parents[3] / "internal" / "hash" / "testdata"
VECTORS_PATH = TESTDATA_DIR / "canonical_vectors.json"
POLICY_DEFAULTS_PATH = TESTDATA_DIR / "policy_decision_defaults.json"


def _load_vectors() -> list[dict]:
    if not VECTORS_PATH.is_file():
        pytest.fail(
            f"golden vectors missing at {VECTORS_PATH}\n"
            "Regenerate with: go test ./internal/hash/... "
            "-run TestGoldenVectors -update-vectors"
        )
    return json.loads(VECTORS_PATH.read_text(encoding="utf-8"))


@pytest.mark.parametrize("vector", _load_vectors(), ids=lambda v: v["description"])
def test_canonical_vector(vector: dict) -> None:
    event = vector["event"]
    prev_hash = vector["prev_hash"]
    expected_canonical = base64.b64decode(vector["expected_canonical_bytes"])
    expected_hash = vector["expected_hash"]

    got_canonical = canonicalize(event)
    if got_canonical != expected_canonical:
        diff = next(
            (
                i
                for i, (a, b) in enumerate(zip(got_canonical, expected_canonical))
                if a != b
            ),
            min(len(got_canonical), len(expected_canonical)),
        )
        raise AssertionError(
            f"canonical bytes drift in {vector['description']!r}\n"
            f"  first diff at byte {diff}\n"
            f"  expected: {expected_canonical.hex()}\n"
            f"  got:      {got_canonical.hex()}"
        )

    got_hash = compute_hash(event, prev_hash)
    assert got_hash == expected_hash, (
        f"hash drift in {vector['description']!r}\n"
        f"  expected: {expected_hash}\n"
        f"  got:      {got_hash}"
    )


def test_policy_decision_defaults_match_shared_fixture() -> None:
    """A policy decision with only the required field must resolve to the
    shared default payload. The TypeScript SDK asserts the same fixture, so
    drift in either SDK's defaults fails the golden gate."""
    fixture = json.loads(POLICY_DEFAULTS_PATH.read_text(encoding="utf-8"))
    payload = WorkflowContext().policy_payload(fixture["action_id"], fixture["input"])
    assert payload == fixture["expected_payload"]
