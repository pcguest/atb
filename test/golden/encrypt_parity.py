#!/usr/bin/env python3
"""Deterministic encryption parity helper for Go test harness."""

from __future__ import annotations

import base64
import binascii
import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "sdk" / "python"))

from atb.encrypt import encrypt_raw  # noqa: E402


def _required(name: str) -> str:
    value = os.environ.get(name, "")
    if not value:
        raise RuntimeError(f"missing required env var {name}")
    return value


def main() -> int:
    plaintext = base64.b64decode(_required("ATB_PARITY_PLAINTEXT_B64"))
    password = _required("ATB_PARITY_PASSWORD")
    salt = bytes.fromhex(_required("ATB_PARITY_SALT_HEX"))
    nonce = bytes.fromhex(_required("ATB_PARITY_NONCE_HEX"))

    encrypted = encrypt_raw(plaintext, password, salt=salt, nonce=nonce)
    sys.stdout.write(binascii.hexlify(encrypted).decode("ascii"))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
