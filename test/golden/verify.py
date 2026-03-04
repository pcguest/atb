#!/usr/bin/env python3
"""Verify Python canonicalization and hash output against Go golden outputs."""

from __future__ import annotations

import hashlib
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "sdk" / "python"))

from atb.canonicalize import canonicalize  # noqa: E402

GENESIS_HASH = "0" * 64


def main() -> int:
    with Path("input.json").open("r", encoding="utf-8") as f:
        event = json.load(f)

    canonical = canonicalize(event)
    if isinstance(canonical, bytes):
        canonical = canonical.decode("utf-8")

    digest = hashlib.sha256((GENESIS_HASH + canonical).encode("utf-8")).hexdigest()

    Path("output-python.json").write_text(canonical, encoding="utf-8")
    Path("hash-python.txt").write_text(digest, encoding="utf-8")

    go_canonical = Path("output-go.json").read_text(encoding="utf-8")
    go_digest = Path("hash-go.txt").read_text(encoding="utf-8")

    if canonical != go_canonical:
        print("Python canonical output mismatch with Go.")
        print(f"python: {canonical}")
        print(f"go:     {go_canonical}")
        return 1

    if digest != go_digest:
        print("Python hash mismatch with Go.")
        print(f"python: {digest}")
        print(f"go:     {go_digest}")
        return 1

    print("✅ Python matches Go (byte-for-byte)")
    print(f"Python hash: {digest}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
