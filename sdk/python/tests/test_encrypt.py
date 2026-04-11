"""Encryption tests for the ATB Python SDK."""

from __future__ import annotations

from copy import deepcopy

import pytest

from atb import Bundle, Event, compute_hash
from atb.bundle import Record
from atb.encrypt import (
    HEADER_SIZE,
    MAGIC,
    LEGACY_VERSION,
    VERSION,
    _encrypt_raw_with_version,
    ATBDecryptionError,
    decrypt_bundle,
    decrypt_raw,
    encrypt_bundle,
    encrypt_raw,
)
from atb.hash import GENESIS_HASH

EXPECTED_GO_VECTOR_HEX = (
    "4154424502101112131415161718191a1b1c1d1e1f202122232425262728292a2bf688fd757a322898adb2343fb39d4a7ddda25540f2da650d5b271d17f6cbd9f6ac74cdf11fccaf54d4604971572033830fd35fa1b3fd443a50108981b5fd452fc2d94ca72fbea39302f45110fc49360bdba052ba58aebdff25e32caa6d513d1cd7449e8a16a4e520f7605280b4a20dd76a255c6db73e3fc9fed990e1297618dcfbc9cd000368cc7b5a13cf1894532ff97ff5de70fb5f3de6fae36e8ed799bdc0465b7b321ae187588d5828b284feb22ff91bd02126d69c6030032686ea73b58692fe142737ca63bd9d4c6e8610433701b0b29f97c4e3e0604adf68b1298dda1f4b57d851a75eb210fd7daba7c3cb497a4d5b6b07090d550b666d1a55c7b9a41b77041ceabc607307a4ac7e0e4e40535292aaa0e8e73d745b8cc57480b31b391ffeb0efa3432df96ae9183db9b18f4144a8968c4565823ddce7f803993a8de349dcb38e31e142c69c22a31b9ec79779b2b672d8653aded11cc86476989eaea5d87bef3ee1e997676d26ed27e02b192202abe83a88a3598aa3d40e85bc2100ccfe433de10cfa8535f78a67c098e960a4ee3986486b506473bcf7329b7e8f36e045a1c136c3d9a169ca875bb8663ee4178105a466ab557ac960378a3c43ffe44b55309de627c3df2427d6c38ef3c45821c6878854520de96f88a4fe134774e5ac6aa773ffa52d4686e1d8c834fed2ae7897258edb299b9943a280ef82ac010bc3b73275fa1201791c01456d90cfbd3f81ad0d795397a346dc38c85e3f1046a3cca49845cbf343e4c599a74f313e9dc041a3d054c7cbed5280582f408d3fce74eabc6479d7726a76acabf8a918f12d3b717204e4a06d54f93e3468cb0b34ba26fad8b0c54792b719e11799006a08efced58177eda5d85ce0197dd14158526a4f5b4ad574c93916ebbb7f30a441dd8b5279556b703444b428a017a9a46c344223ace9479dbce73df5c8b8ddb4947966f1feccb0518f15bc03c15443af40ee88b7a5a1c5ff9fca9955b5e4a7ea5550ec28656d55f8a1c009a86d86215cfd73c234d0aedf8f10487eb0439f6302576327e693b0ea96013a8d3cda30dfa4a446da10c57cba3df06653e46fb6a080802b4f195796a4620e01d14130c819eb6bdb0b689cabec69973c22cbae9fa63d026dc5db46d7214a4e60c4095a744cc8434c3663fa8ceae1821f55598f509c00c204295d42c4371d5a5e30922c6f2dee8fdaf3882ada245877a013f80e36eb990eb9cbdcb77e821b20205ea5f00836ec70c783c91be158277ba5baf1ffe1251ceb161ef1ea23acb6b6aa36d2c60fe2b151df64e0d547a2f0ddf2ca3ad48005ecfec88dc147e369773341d796317c84d2253569f3113427"
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
    manifest_timestamp = "2026-04-01T00:00:00Z"
    manifest_event = Event(
        seq=0,
        prev_hash=GENESIS_HASH,
        type="atb.bundle.manifest",
        data='{"version":"1","created_at":"2026-04-01T00:00:00Z","bundle_id":"00112233445566778899aabbccddeeff"}',
        hash_algo="sha256",
        timestamp=manifest_timestamp,
    ).to_dict()
    bundle = Bundle(
        goal="encrypt-tests",
        records=[Record(event=manifest_event, hash=compute_hash(manifest_event, GENESIS_HASH))],
    )
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
    encrypted[len(MAGIC)] = 0x03
    with pytest.raises(ATBDecryptionError, match="unsupported version"):
        decrypt_raw(bytes(encrypted), "test123")


def test_decrypt_legacy_version_roundtrip() -> None:
    plaintext = b'{"head_hash":"abc","records":[]}'
    encrypted = _encrypt_raw_with_version(
        plaintext,
        "test123",
        version=LEGACY_VERSION,
        salt=_fixed_salt(),
        nonce=_fixed_nonce(),
    )

    assert encrypted[len(MAGIC)] == LEGACY_VERSION
    assert decrypt_raw(encrypted, "test123") == plaintext


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
