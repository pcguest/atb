"""
ATB bundle encryption helpers.

Wire format:
    [4 bytes magic "ATBE"]
    [1 byte version]
    [16 bytes salt]
    [12 bytes nonce]
    [16 bytes auth tag]
    [N bytes ciphertext]

Quick start::

    from atb import Bundle, encrypt_bundle
    encrypted = encrypt_bundle(Bundle.load("run.atb/bundle.atb"), "password")
    assert encrypted.startswith(b"ATBE")
"""

from __future__ import annotations

import json
import os
from typing import TYPE_CHECKING

from cryptography.exceptions import InvalidTag
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC

from atb.canonicalize import canonicalize
from atb.exceptions import ATBError, ATBVerificationError
from atb.hash import GENESIS_HASH

if TYPE_CHECKING:
    from atb.bundle import Bundle


MAGIC = b"ATBE"
LEGACY_VERSION = 0x01
VERSION = 0x02
SALT_SIZE = 16
NONCE_SIZE = 12
TAG_SIZE = 16
KEY_SIZE = 32
LEGACY_PBKDF2_ITERATIONS = 100_000
PBKDF2_ITERATIONS = 600_000
HEADER_SIZE = len(MAGIC) + 1 + SALT_SIZE + NONCE_SIZE + TAG_SIZE


class ATBEncryptionError(ATBError):
    """Raised when encryption fails.

    Args:
        *args: Positional error message arguments passed to ``Exception``.

    Returns:
        None.

    Raises:
        None.
    """


class ATBDecryptionError(ATBError):
    """Raised when decryption fails.

    Args:
        *args: Positional error message arguments passed to ``Exception``.

    Returns:
        None.

    Raises:
        None.
    """


def _derive_key(
    password: str,
    salt: bytes,
    iterations: int,
    *,
    error_cls: type[ATBError],
) -> bytes:
    if not password:
        raise error_cls("password cannot be empty")
    if len(salt) != SALT_SIZE:
        raise error_cls(f"salt must be {SALT_SIZE} bytes")
    kdf = PBKDF2HMAC(
        algorithm=hashes.SHA256(),
        length=KEY_SIZE,
        salt=salt,
        iterations=iterations,
    )
    return kdf.derive(password.encode("utf-8"))


def _kdf_iterations_for_version(version: int, *, error_cls: type[ATBError]) -> int:
    if version == LEGACY_VERSION:
        return LEGACY_PBKDF2_ITERATIONS
    if version == VERSION:
        return PBKDF2_ITERATIONS
    raise error_cls(f"unsupported version: 0x{version:02x}")


def _encrypt_raw_with_version(
    plaintext: bytes,
    password: str,
    *,
    version: int,
    salt: bytes | None = None,
    nonce: bytes | None = None,
) -> bytes:
    salt_bytes = salt if salt is not None else os.urandom(SALT_SIZE)
    nonce_bytes = nonce if nonce is not None else os.urandom(NONCE_SIZE)
    if len(salt_bytes) != SALT_SIZE:
        raise ATBEncryptionError(f"salt must be {SALT_SIZE} bytes")
    if len(nonce_bytes) != NONCE_SIZE:
        raise ATBEncryptionError(f"nonce must be {NONCE_SIZE} bytes")

    iterations = _kdf_iterations_for_version(version, error_cls=ATBEncryptionError)
    key = _derive_key(
        password,
        salt_bytes,
        iterations,
        error_cls=ATBEncryptionError,
    )
    aesgcm = AESGCM(key)
    sealed = aesgcm.encrypt(nonce_bytes, plaintext, None)
    if len(sealed) < TAG_SIZE:
        raise ATBEncryptionError("invalid AES-GCM output")
    ciphertext = sealed[:-TAG_SIZE]
    tag = sealed[-TAG_SIZE:]
    return MAGIC + bytes([version]) + salt_bytes + nonce_bytes + tag + ciphertext


def encrypt_raw(
    plaintext: bytes,
    password: str,
    *,
    salt: bytes | None = None,
    nonce: bytes | None = None,
) -> bytes:
    """Encrypt plaintext bytes with AES-256-GCM.

    Optional salt/nonce are intended for deterministic tests and golden vectors.

    Args:
        plaintext: Plain bytes to encrypt.
        password: Password used for PBKDF2 key derivation.
        salt: Optional deterministic salt for tests.
        nonce: Optional deterministic nonce for tests.

    Returns:
        ATBE wire bytes.

    Raises:
        ATBEncryptionError: If the password, salt, nonce, or AES-GCM output is
            invalid.
    """
    return _encrypt_raw_with_version(
        plaintext,
        password,
        version=VERSION,
        salt=salt,
        nonce=nonce,
    )


def decrypt_raw(data: bytes, password: str) -> bytes:
    """Decrypt encrypted bytes in ATBE wire format.

    Args:
        data: ATBE wire bytes.
        password: Password used for PBKDF2 key derivation.

    Returns:
        Decrypted plaintext bytes.

    Raises:
        ATBDecryptionError: If the format is invalid or authentication fails.
    """
    if len(data) < HEADER_SIZE:
        raise ATBDecryptionError("invalid format")
    if data[: len(MAGIC)] != MAGIC:
        raise ATBDecryptionError("invalid format")
    version = data[len(MAGIC)]
    if not password:
        raise ATBDecryptionError("password cannot be empty")
    iterations = _kdf_iterations_for_version(version, error_cls=ATBDecryptionError)

    offset = len(MAGIC) + 1
    salt = data[offset : offset + SALT_SIZE]
    offset += SALT_SIZE
    nonce = data[offset : offset + NONCE_SIZE]
    offset += NONCE_SIZE
    tag = data[offset : offset + TAG_SIZE]
    offset += TAG_SIZE
    ciphertext = data[offset:]

    key = _derive_key(
        password,
        salt,
        iterations,
        error_cls=ATBDecryptionError,
    )
    aesgcm = AESGCM(key)
    try:
        return aesgcm.decrypt(nonce, ciphertext + tag, None)
    except InvalidTag as exc:
        raise ATBDecryptionError("authentication failed") from exc


def _bundle_payload(bundle: Bundle) -> dict[str, object]:
    records = [{"event": r.event, "hash": r.hash} for r in bundle.records]
    head_hash = records[-1]["hash"] if records else GENESIS_HASH
    return {"head_hash": head_hash, "records": records}


def encrypt_bundle(
    bundle: Bundle,
    password: str,
    *,
    salt: bytes | None = None,
    nonce: bytes | None = None,
) -> bytes:
    """Encrypt a bundle to ATBE bytes.

    Args:
        bundle: Bundle to verify and encrypt.
        password: Password used for PBKDF2 key derivation.
        salt: Optional deterministic salt for tests.
        nonce: Optional deterministic nonce for tests.

    Returns:
        ATBE wire bytes.

    Raises:
        ATBVerificationError: If the bundle fails hash-chain verification.
        ATBEncryptionError: If encryption fails.
    """
    bundle.verify()
    payload = _bundle_payload(bundle)
    canonical = canonicalize(payload)
    return encrypt_raw(canonical, password, salt=salt, nonce=nonce)


def decrypt_bundle(password: str, data: bytes) -> Bundle:
    """Decrypt bytes into a verified bundle.

    Args:
        password: Password used for PBKDF2 key derivation.
        data: ATBE wire bytes.

    Returns:
        A verified ``Bundle``.

    Raises:
        ATBDecryptionError: If decoding, authentication, or verification fails.
    """
    plaintext = decrypt_raw(data, password)
    try:
        payload = json.loads(plaintext.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise ATBDecryptionError("decrypted payload is not valid JSON") from exc
    if not isinstance(payload, dict):
        raise ATBDecryptionError("decrypted payload must be an object")

    raw_records = payload.get("records")
    raw_head_hash = payload.get("head_hash")
    if not isinstance(raw_records, list) or not isinstance(raw_head_hash, str):
        raise ATBDecryptionError("decrypted payload missing required fields")

    from atb.bundle import Bundle, Record

    records: list[Record] = []
    for item in raw_records:
        if not isinstance(item, dict):
            raise ATBDecryptionError("record must be an object")
        event = item.get("event")
        record_hash = item.get("hash")
        if not isinstance(event, dict) or not isinstance(record_hash, str):
            raise ATBDecryptionError("record must include event object and hash string")
        records.append(Record(event=event, hash=record_hash))

    bundle = Bundle(records=records)
    try:
        bundle.verify()
    except ATBVerificationError as exc:
        raise ATBDecryptionError("decrypted payload failed hash-chain verification") from exc

    recomputed_head = bundle.records[-1].hash if bundle.records else GENESIS_HASH
    if raw_head_hash != recomputed_head:
        raise ATBDecryptionError(
            f"decrypted payload head_hash mismatch: expected {raw_head_hash}, got {recomputed_head}"
        )
    return bundle
