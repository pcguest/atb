"""Encryption tests for the ATB Python SDK."""

from __future__ import annotations

from copy import deepcopy

import pytest

from atb import Bundle
from atb.encrypt import (
    HEADER_SIZE,
    MAGIC,
    VERSION,
    ATBDecryptionError,
    decrypt_bundle,
    decrypt_raw,
    encrypt_bundle,
    encrypt_raw,
)

EXPECTED_GO_VECTOR_HEX = (
    "4154424501101112131415161718191a1b1c1d1e1f202122232425262728292a2b6281773a2bf9c95f7f0a51a057f7ea5a5584ed630df2312b4c45ed3c104553e2312f48ba948ba4790d75858ae40a518b4289c7cde8b775504d2e0727a6ee891a09d53d8b4a1467dc4a74c9e7a8012c270b3c4107621caa8eea6ec779bd31073c1b336c3da5f0e5d4a34770dc5109cfb93a4d90d9cfd6307b6a487089c69d5f50c1714bfc26b3107f8540ac31e366fe389e77ad8c5fe78aea46dedd7d0c2d235053a62625e67c094075055cdb077a9f81f22e8dd5d0f01e927a8307032ebe0df5139ec5e319eac1cfefbd3ac3f8a76c9cd19c92278f984f2804da2a660a23c0a15115b939fcd095ebac3b2662e50f2adb683a842a66a8557e4ef26beb07600a35ef6f86d2768f2f0ceee0b465a5e13fbb14a26c74c0df04cf069209815c502fe9f92b628ecf104e719e9b931bd196989952f87399af49446b7ac0a188338c46f05a781b4a30cfdf59a7b5537f53d2263b4ae7192398ff920162e4c071214138e83ddae16a3b3dd5ee35e1593c4d8c3ce4cf591f95c5a9f81a4d43bfcf78c4e54a8398cd57e2741e14a1c97da69b09160224026745b66e312e80900e6205b06cd72aa6026b2b85b734452e73da56f2c6f99b62d9b98e19c1c87d868fdf678cef5119eae4d324234214cba7726f1981af65aebe36f37542ab8d0f94007a1765a12116bc9abfb7b833fe89333da6bf268126b2f2689b7561b2ba0d7b174bf089b7d49612ded3cb0e4ea30166331382370b31d6f5381c490871035e5f632ce6"
)


def _fixed_salt() -> bytes:
    return bytes(
        [
            0x10,
            0x11,
            0x12,
            0x13,
            0x14,
            0x15,
            0x16,
            0x17,
            0x18,
            0x19,
            0x1A,
            0x1B,
            0x1C,
            0x1D,
            0x1E,
            0x1F,
        ]
    )


def _fixed_nonce() -> bytes:
    return bytes(
        [
            0x20,
            0x21,
            0x22,
            0x23,
            0x24,
            0x25,
            0x26,
            0x27,
            0x28,
            0x29,
            0x2A,
            0x2B,
        ]
    )


def _sample_bundle() -> Bundle:
    bundle = Bundle(goal="encrypt-tests")
    bundle.append("dev.session", {"msg": "hello"})
    bundle.append("decision", {"choice": "ship"})
    return bundle


def test_encrypt_decrypt_roundtrip_deterministic() -> None:
    bundle = _sample_bundle()
    first = encrypt_bundle(
        bundle,
        "test123",
        salt=_fixed_salt(),
        nonce=_fixed_nonce(),
    )
    second = encrypt_bundle(
        bundle,
        "test123",
        salt=_fixed_salt(),
        nonce=_fixed_nonce(),
    )

    assert first == second
    assert first[: len(MAGIC)] == MAGIC
    assert first[len(MAGIC)] == VERSION
    assert len(first) > HEADER_SIZE

    decrypted = decrypt_bundle("test123", first)
    assert len(decrypted.records) == len(bundle.records)
    assert [deepcopy(r.event) for r in decrypted.records] == [
        deepcopy(r.event) for r in bundle.records
    ]
    assert [r.hash for r in decrypted.records] == [r.hash for r in bundle.records]


def test_decrypt_wrong_password_fails() -> None:
    encrypted = encrypt_raw(
        b'{"head_hash":"abc","records":[]}',
        "test123",
        salt=_fixed_salt(),
        nonce=_fixed_nonce(),
    )
    with pytest.raises(ATBDecryptionError, match="authentication failed"):
        decrypt_raw(encrypted, "wrong-pass")


def test_decrypt_tampered_ciphertext_fails() -> None:
    encrypted = bytearray(
        encrypt_raw(
            b'{"head_hash":"abc","records":[]}',
            "test123",
            salt=_fixed_salt(),
            nonce=_fixed_nonce(),
        )
    )
    encrypted[-1] ^= 0x01
    with pytest.raises(ATBDecryptionError, match="authentication failed"):
        decrypt_raw(bytes(encrypted), "test123")


def test_decrypt_rejects_unsupported_version() -> None:
    encrypted = bytearray(
        encrypt_raw(
            b'{"head_hash":"abc","records":[]}',
            "test123",
            salt=_fixed_salt(),
            nonce=_fixed_nonce(),
        )
    )
    encrypted[len(MAGIC)] = 0x02
    with pytest.raises(ATBDecryptionError, match="unsupported version"):
        decrypt_raw(bytes(encrypted), "test123")


def test_decrypt_then_verify_head_hash_mismatch() -> None:
    bundle = _sample_bundle()
    encrypted = bytearray(
        encrypt_bundle(bundle, "test123", salt=_fixed_salt(), nonce=_fixed_nonce())
    )
    # Corrupt one byte in the encrypted payload and restore tag by re-encrypting:
    # we need a structurally valid payload with an invalid head_hash for this check.
    decrypted = decrypt_bundle("test123", bytes(encrypted))
    decrypted_payload = {
        "head_hash": "0" * 64,
        "records": [{"event": r.event, "hash": r.hash} for r in decrypted.records],
    }
    from atb.canonicalize import canonicalize

    tampered_plaintext = canonicalize(decrypted_payload)
    tampered = encrypt_raw(
        tampered_plaintext,
        "test123",
        salt=_fixed_salt(),
        nonce=_fixed_nonce(),
    )
    with pytest.raises(ATBDecryptionError, match="head_hash mismatch"):
        decrypt_bundle("test123", tampered)


def test_bundle_methods_roundtrip() -> None:
    bundle = _sample_bundle()
    encrypted = bundle.encrypt("test123")
    decrypted = Bundle.decrypt("test123", encrypted)
    assert [r.hash for r in decrypted.records] == [r.hash for r in bundle.records]


def test_go_parity_vector_fixed_salt_nonce() -> None:
    bundle = _sample_bundle()
    encrypted = encrypt_bundle(
        bundle,
        "test123",
        salt=_fixed_salt(),
        nonce=_fixed_nonce(),
    )
    assert encrypted.hex() == EXPECTED_GO_VECTOR_HEX
