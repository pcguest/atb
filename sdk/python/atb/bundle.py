"""ATB bundle reader and verifier.

This module provides the ``Bundle`` loader, hash-chain verifier, signature
inspection helper, and ``append_events_in_memory`` utility for SDK callers that
need to read ATB bundles, verify chains, and iterate records.

Quick start::

    from atb.bundle import Bundle
    bundle = Bundle.load("run.atb/bundle.atb")
    bundle.verify()
"""

from __future__ import annotations

import base64
import binascii
import contextlib
import hashlib
import json
import math
import os
import re
import secrets
import tempfile
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

from atb.event import Event
from atb.exceptions import ATBError, ATBVerificationError
from atb.hash import GENESIS_HASH, compute_hash

#: Default bundle directory and file name.
BUNDLE_DIR = "run.atb"
BUNDLE_FILE = "bundle.atb"
MANIFEST_EVENT_TYPE = "atb.bundle.manifest"
SIGNATURE_EVENT_TYPE = "atb.bundle.signature"
MANIFEST_VERSION = "1"
#: Highest manifest version this SDK can read. Mirrors
#: ``internal/bundle.ManifestVersionMax``. Versions above this are rejected
#: before any hash or signature evidence is produced.
MAX_MANIFEST_VERSION = 3
# Reported for versions that are known to be unsupported but whose integer part
# is not meaningful (non-finite numbers, oversized digit strings, or fractions
# just above the maximum). It must be greater than MAX_MANIFEST_VERSION.
_UNSUPPORTED_VERSION_SENTINEL = MAX_MANIFEST_VERSION + 1
MAX_LINE_SIZE_BYTES = 16 * 1024 * 1024
MAX_BUNDLE_SIZE_BYTES = 512 * 1024 * 1024
MAX_BUNDLE_RECORDS = 1_000_000

# Mirrors internal/bundle/bundle.go::eventTypeRegexp. Used by the
# in-memory append helper to surface invalid event types as ValueError
# before they reach the hash layer.
_EVENT_TYPE_PATTERN = re.compile(r"^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$")


class UnsupportedAlgorithmError(ValueError):
    """Raised when a bundle signature uses an unsupported algorithm.

    Args:
        *args: Positional error message arguments passed to ``ValueError``.

    Returns:
        None.

    Raises:
        None.
    """


class BundleResourceLimitError(ValueError):
    """Raised when a bundle exceeds a configured safe-loading limit."""


class UnsupportedManifestVersionError(ATBError, ValueError):
    """Raised when a bundle declares a manifest version this SDK cannot read.

    This is deliberately not an :class:`~atb.exceptions.ATBVerificationError`:
    the bundle may be intact, but this SDK is too old to interpret it. Callers
    should upgrade the SDK rather than treat the bundle as tampered. It remains
    catchable as :class:`~atb.exceptions.ATBError` and as ``ValueError``.

    Args:
        version: The determinable integer manifest version that is unsupported.

    Returns:
        None.

    Raises:
        None.
    """

    def __init__(self, version: int) -> None:
        self.version = version
        super().__init__(
            f"atb: unsupported manifest version {version}: "
            "this bundle requires a newer version of the SDK"
        )


@dataclass
class Record:
    """A single record in an ATB bundle.

    Args:
        event: Canonical event object from the bundle line.
        hash: Hex-encoded chain hash for ``event``.

    Returns:
        A dataclass instance.

    Raises:
        None.
    """

    event: dict[str, Any]
    hash: str


@dataclass
class Bundle:
    """An in-memory ATB bundle.

    Usage::

        from atb.event_types import AI_REQUEST_RECEIVED_EVENT_TYPE

        bundle = Bundle()
        bundle.append(AI_REQUEST_RECEIVED_EVENT_TYPE, {"request_id": "req-001"})
        bundle.save()

        # Reload and verify
        b = Bundle.load()
        b.verify()
    """

    def __init__(
        self,
        goal: str | None = None,
        name: str | None = None,
        records: list[Record] | None = None,
    ):
        """Initialize a new ATB bundle.

        Args:
            goal: Optional goal/description for the bundle (alias for name)
            name: Optional name for the bundle
            records: Optional list of initial records
        """
        object.__setattr__(self, "name", goal or name or "untitled")
        object.__setattr__(self, "records", records if records is not None else [])
        object.__setattr__(self, "_raw_bytes", None)
        if records is None:
            self._append_manifest()

    name: str = "untitled"

    records: list[Record] = field(default_factory=list)

    # ------------------------------------------------------------------
    # Mutation
    # ------------------------------------------------------------------

    def append(
        self,
        event_type: str,
        data: Any,
        *,
        actor_id: str | None = None,
        org_id: str | None = None,
        workspace_id: str | None = None,
        timestamp: str | None = None,
        trace_id: str | None = None,
        span_id: str | None = None,
        parent_span_id: str | None = None,
        acquisition: Any | None = None,
    ) -> Record:
        """Append a new event to the bundle.

        Args:
            event_type: A dot-namespaced event type string (e.g. ``"dev.session"``).
            data: Arbitrary JSON-serialisable payload.
            actor_id: Optional actor identifier.
            org_id: Optional organization identifier.
            workspace_id: Optional workspace identifier.
            acquisition: Optional bounded acquisition provenance (an
                :class:`~atb.event.Acquisition` or mapping). Unknown fields are
                dropped and malformed values raise ``TypeError``.

        Returns:
            The newly created :class:`Record`.
        """
        prev_hash = self.records[-1].hash if self.records else GENESIS_HASH
        sequence = len(self.records) + 1
        if self._has_manifest_record:
            sequence = len(self.records)
        event_obj = Event(
            seq=sequence,
            prev_hash=prev_hash,
            type=event_type,
            data=data,
            hash_algo="sha256",
            actor_id=_normalize_optional_identity(actor_id),
            org_id=_normalize_optional_identity(org_id),
            workspace_id=_normalize_optional_identity(workspace_id),
            timestamp=_normalize_optional_field(timestamp),
            trace_id=_normalize_optional_field(trace_id),
            span_id=_normalize_optional_field(span_id),
            parent_span_id=_normalize_optional_field(parent_span_id),
            acquisition=acquisition,
        )
        event = event_obj.to_dict()
        h = compute_hash(event, prev_hash)
        record = Record(event=event, hash=h)
        self.records.append(record)
        return record

    # ------------------------------------------------------------------
    # Verification
    # ------------------------------------------------------------------

    def verify(self) -> dict[str, Any]:
        """Verify the integrity of the entire bundle.

        Raises:
            UnsupportedManifestVersionError: If the manifest declares a version
                newer than this SDK supports.
            ATBVerificationError: If any event's hash does not match the
                recomputed value, indicating tampering, or if a record's stored
                ``seq``/``prev_hash`` does not match its expected chain value.
        """
        self._assert_supported_manifest_version()
        prev = GENESIS_HASH
        for i, record in enumerate(self.records):
            event = dict(record.event)
            stored_seq = record.event.get("seq")
            stored_prev_hash = record.event.get("prev_hash")
            if i == 0 and self._has_manifest_record:
                expected_seq = 0
            elif self._has_manifest_record:
                expected_seq = i
            else:
                expected_seq = i + 1
            event["prev_hash"] = prev
            event["seq"] = expected_seq
            computed = compute_hash(event, prev)
            if stored_seq != expected_seq:
                raise ATBVerificationError(
                    f"sequence mismatch at index {i}: expected seq "
                    f"{expected_seq}, got seq {stored_seq!r}",
                    event_index=i,
                    expected_hash=record.hash,
                    computed_hash=computed,
                )
            if stored_prev_hash != prev:
                raise ATBVerificationError(
                    f"prev_hash mismatch at index {i}: expected {prev}, "
                    f"got {stored_prev_hash!r}",
                    event_index=i,
                    expected_hash=record.hash,
                    computed_hash=computed,
                )
            if computed != record.hash:
                raise ATBVerificationError(
                    f"Tamper detected at event {i + 1} (seq {expected_seq}): "
                    f"expected {record.hash!r}, computed {computed!r}",
                    event_index=i,
                    expected_hash=record.hash,
                    computed_hash=computed,
                )
            prev = computed
        return {"signatures": self.signatures()}

    def signatures(self, path: str | Path | None = None) -> list[dict[str, Any]]:
        """Return per-signature provenance and verification results.

        Args:
            path: Optional bundle path to use as the raw signature source.

        Returns:
            A list of signature evidence dictionaries.

        Raises:
            UnsupportedManifestVersionError: If the manifest declares a version
                newer than this SDK supports.
            UnsupportedAlgorithmError: If a signature record declares an
                unsupported algorithm.
            OSError: If ``path`` is supplied and cannot be read.
        """
        self._assert_supported_manifest_version()
        raw = self._signature_source_bytes(path)
        results: list[dict[str, Any]] = []
        for index, record in enumerate(self.records):
            if record.event.get("type") != SIGNATURE_EVENT_TYPE:
                continue
            data = record.event.get("data")
            if not isinstance(data, dict):
                continue
            pubkey = _string_field(data, "pubkey") or _string_field(data, "public_key")
            bundle_hash = _string_field(data, "bundle_hash")
            valid = _verify_signature_record(data, raw, index)
            results.append(
                {
                    "sequence": int(record.event.get("seq", 0)),
                    "backend": _string_field(data, "backend") or "local",
                    "key_id": _string_field(data, "key_id"),
                    "signed_at": _string_field(data, "signed_at"),
                    "pubkey": pubkey,
                    "bundle_hash": bundle_hash,
                    "valid": valid,
                }
            )
        return results

    # ------------------------------------------------------------------
    # Signing
    # ------------------------------------------------------------------

    def sign_local(
        self,
        private_key_pem: str | bytes | Path,
        path: str | Path | None = None,
        *,
        key_id: str = "",
    ) -> Record:
        """Append a local Ed25519 bundle signature record to ``path``.

        Args:
            private_key_pem: PEM bytes, PEM text, or path to an Ed25519
                private key.
            path: Bundle path to sign. Defaults to ``run.atb/bundle.atb``.
            key_id: Optional key identifier to store in the signature payload.

        Returns:
            The appended signature ``Record``.

        Raises:
            TypeError: If the private key is not Ed25519.
            OSError: If the key or bundle file cannot be read or written.
        """
        from cryptography.hazmat.primitives import serialization

        resolved = Path(path) if path else Path(BUNDLE_DIR) / BUNDLE_FILE
        if not resolved.exists():
            self.save(resolved)

        raw = resolved.read_bytes()
        digest = hashlib.sha256(raw).digest()
        private_key = _load_private_key(private_key_pem)
        signature = private_key.sign(digest)
        public_key = private_key.public_key().public_bytes(
            encoding=serialization.Encoding.Raw,
            format=serialization.PublicFormat.Raw,
        )

        signed_at = _now_rfc3339()
        record = self.append(
            SIGNATURE_EVENT_TYPE,
            {
                "bundle_hash": digest.hex(),
                "signature": base64.b64encode(signature).decode("ascii"),
                "pubkey": base64.b64encode(public_key).decode("ascii"),
                "key_id": key_id,
                "backend": "local",
                "signed_at": signed_at,
            },
            timestamp=signed_at,
        )

        encoded_record = json.dumps(
            {"event": record.event, "hash": record.hash}
        ).encode("utf-8")
        payload = raw
        if payload and not payload.endswith(b"\n"):
            payload += b"\n"
        payload += encoded_record + b"\n"
        resolved.parent.mkdir(parents=True, exist_ok=True)
        _save_atomic(resolved, payload)
        object.__setattr__(self, "_raw_bytes", payload)
        return record

    # ------------------------------------------------------------------
    # Persistence
    # ------------------------------------------------------------------

    def save(self, path: str | Path | None = None) -> Path:
        """Save the bundle to *path* in NDJSON format.

        Args:
            path: File path to write. Defaults to ``run.atb/bundle.atb``.

        Returns:
            The resolved :class:`~pathlib.Path` that was written.
        """
        resolved = Path(path) if path else Path(BUNDLE_DIR) / BUNDLE_FILE
        resolved.parent.mkdir(parents=True, exist_ok=True)
        payload = _serialise_records(self.records)
        _save_atomic(resolved, payload)
        object.__setattr__(self, "_raw_bytes", payload)
        return resolved

    @classmethod
    def load(
        cls,
        path: str | Path | None = None,
        *,
        max_bytes: int = MAX_BUNDLE_SIZE_BYTES,
        max_records: int = MAX_BUNDLE_RECORDS,
    ) -> "Bundle":
        """Load a bundle from *path*.

        Args:
            path: File path to read. Defaults to ``run.atb/bundle.atb``.
            max_bytes: Maximum encoded bundle size accepted.
            max_records: Maximum number of non-empty records accepted.

        Returns:
            A populated :class:`Bundle` instance.
        """
        if max_bytes <= 0 or max_records <= 0:
            raise ValueError("bundle load limits must be positive")
        resolved = Path(path) if path else Path(BUNDLE_DIR) / BUNDLE_FILE
        bundle = cls()
        bundle.records.clear()
        raw = resolved.read_bytes()
        if len(raw) > max_bytes:
            raise BundleResourceLimitError(
                f"bundle exceeds maximum size of {max_bytes} bytes"
            )
        for line_number, encoded_line in enumerate(raw.splitlines(), start=1):
            if len(encoded_line) > MAX_LINE_SIZE_BYTES:
                raise BundleResourceLimitError(
                    f"bundle record at line {line_number} exceeds "
                    f"{MAX_LINE_SIZE_BYTES} bytes"
                )
            line = encoded_line.strip()
            if not line:
                continue
            if len(bundle.records) >= max_records:
                raise BundleResourceLimitError(
                    f"bundle exceeds maximum record count of {max_records}"
                )
            obj = json.loads(line.decode("utf-8"))
            bundle.records.append(Record(event=obj["event"], hash=obj["hash"]))
        bundle._assert_supported_manifest_version()
        object.__setattr__(bundle, "_raw_bytes", raw)
        return bundle

    # ------------------------------------------------------------------
    # Encryption
    # ------------------------------------------------------------------

    def encrypt(self, password: str) -> bytes:
        """Encrypt this bundle to ATBE bytes.

        Args:
            password: Password used to derive the encryption key.

        Returns:
            Encrypted ATBE wire bytes.

        Raises:
            UnsupportedManifestVersionError: If the manifest declares a version
                newer than this SDK supports.
            ATBVerificationError: If the bundle does not verify before
                encryption.
            ATBEncryptionError: If encryption fails.
        """
        self._assert_supported_manifest_version()
        from atb.encrypt import encrypt_bundle

        return encrypt_bundle(self, password)

    @classmethod
    def decrypt(cls, password: str, data: bytes) -> "Bundle":
        """Decrypt ATBE bytes and return a verified bundle.

        Args:
            password: Password used to derive the decryption key.
            data: ATBE wire bytes.

        Returns:
            A verified ``Bundle``.

        Raises:
            UnsupportedManifestVersionError: If the decrypted manifest declares
                a version newer than this SDK supports.
            ATBDecryptionError: If authentication, decoding, or verification
                fails.
        """
        from atb.encrypt import decrypt_bundle

        bundle = decrypt_bundle(password, data)
        bundle._assert_supported_manifest_version()
        return bundle

    # ------------------------------------------------------------------
    # Convenience
    # ------------------------------------------------------------------

    def __len__(self) -> int:
        return len(self.records)

    def __repr__(self) -> str:
        return f"Bundle(records={len(self.records)})"

    def _signature_source_bytes(self, path: str | Path | None = None) -> bytes:
        if path is not None:
            return Path(path).read_bytes()
        raw = getattr(self, "_raw_bytes", None)
        if isinstance(raw, bytes):
            return raw
        return _serialise_records(self.records)

    @property
    def _has_manifest_record(self) -> bool:
        return (
            bool(self.records)
            and self.records[0].event.get("type") == MANIFEST_EVENT_TYPE
        )

    def _assert_supported_manifest_version(self) -> None:
        """Reject manifests that declare a version newer than this SDK reads.

        A finite numeric version above the maximum, and a string whose leading
        decimal-digit run is above the maximum, are rejected. A fractional
        version just above the maximum is rejected even if truncation would be
        supported. Absent or empty versions are legacy v1. Shapes without a
        determinable leading integer are left to the existing verification path
        rather than being reinterpreted here.

        Raises:
            UnsupportedManifestVersionError: If the version is greater than
                :data:`MAX_MANIFEST_VERSION`.
        """
        version = _recorded_manifest_version(self.records)
        if version is not None and version > MAX_MANIFEST_VERSION:
            raise UnsupportedManifestVersionError(version)

    def _append_manifest(self) -> None:
        created_at = _now_rfc3339()
        payload = json.dumps(
            {
                "version": MANIFEST_VERSION,
                "created_at": created_at,
                "bundle_id": secrets.token_hex(16),
            },
            separators=(",", ":"),
        )
        event = Event(
            seq=0,
            prev_hash=GENESIS_HASH,
            type=MANIFEST_EVENT_TYPE,
            data=payload,
            hash_algo="sha256",
            timestamp=created_at,
        ).to_dict()
        self.records.append(Record(event=event, hash=compute_hash(event, GENESIS_HASH)))


def append_events_in_memory(bundle: "Bundle", events: list[Any]) -> int:
    """Append *events* to *bundle* without loading or saving.

    Mirrors ``capture.AppendEventsToBundleInMemory`` in the Go reference
    implementation: stops at the first invalid event and reports the
    number of events that *did* land in ``bundle.records``. Callers that
    require all-or-nothing semantics must avoid persisting *bundle* on
    error.

    Args:
        bundle: Target :class:`Bundle`. Must not be ``None``.
        events: An iterable of event-spec mappings or :class:`Event`
            instances. Each entry must expose ``type`` (str) and
            ``data`` (Any); ``timestamp`` is optional.

    Returns:
        The number of events successfully appended in memory.

    Raises:
        ValueError: when *bundle* is ``None``, when an entry is missing
            a ``type`` field, or when the ``type`` does not match the
            ATB event-type pattern.
    """
    if bundle is None:
        raise ValueError("append_events_in_memory: bundle is None")

    appended = 0
    for index, spec in enumerate(events):
        event_type, data, timestamp, acquisition = _coerce_event_spec(index, spec)
        if not _EVENT_TYPE_PATTERN.match(event_type):
            raise ValueError(
                f"append_events_in_memory: event {index} type "
                f"{event_type!r} does not match required pattern "
                '(e.g. "ai.tool.exec")'
            )
        bundle.append(event_type, data, timestamp=timestamp, acquisition=acquisition)
        appended += 1
    return appended


def _coerce_event_spec(index: int, spec: Any) -> tuple[str, Any, str | None, Any]:
    if isinstance(spec, Event):
        return spec.type, spec.data, spec.timestamp, spec.acquisition
    if isinstance(spec, dict):
        if "type" not in spec:
            raise ValueError(
                f"append_events_in_memory: event {index} missing 'type' field"
            )
        event_type = spec["type"]
        if not isinstance(event_type, str):
            raise ValueError(
                f"append_events_in_memory: event {index} 'type' must be str, "
                f"got {type(event_type).__name__}"
            )
        return (
            event_type,
            spec.get("data"),
            spec.get("timestamp"),
            spec.get("acquisition"),
        )
    raise ValueError(
        f"append_events_in_memory: event {index} must be a dict or Event, "
        f"got {type(spec).__name__}"
    )


def _normalize_optional_identity(value: str | None) -> str | None:
    """Treat None/empty identity values as unset for canonical compatibility."""
    if value is None:
        return None
    trimmed = value.strip()
    if trimmed == "":
        return None
    return trimmed


def _normalize_optional_field(value: str | None) -> str | None:
    if value is None:
        return None
    trimmed = value.strip()
    if trimmed == "":
        return None
    return trimmed


def _now_rfc3339() -> str:
    return (
        datetime.now(timezone.utc)
        .replace(microsecond=0)
        .isoformat()
        .replace("+00:00", "Z")
    )


def _load_private_key(private_key_pem: str | bytes | Path) -> "Ed25519PrivateKey":
    from cryptography.hazmat.primitives import serialization
    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

    if isinstance(private_key_pem, Path):
        data = private_key_pem.read_bytes()
    elif isinstance(private_key_pem, bytes):
        data = private_key_pem
    else:
        if "-----BEGIN" in private_key_pem:
            data = private_key_pem.encode("utf-8")
        else:
            data = Path(private_key_pem).read_bytes()
    key = serialization.load_pem_private_key(data, password=None)
    if not isinstance(key, Ed25519PrivateKey):
        raise TypeError("ATB local signing requires an Ed25519 private key")
    return key


def _save_atomic(path: Path, data: bytes) -> None:
    """Write data atomically using a temp file and rename.

    The temporary file lives beside the target so the final rename stays on
    the same filesystem. The temp file is fsynced before replacement; on any
    failure the original file is left untouched and the temp file is removed.
    """
    parent = path.parent
    fd, tmp_path = tempfile.mkstemp(dir=parent, suffix=".atb.tmp")
    try:
        with os.fdopen(fd, "wb") as f:
            f.write(data)
            f.flush()
            os.fsync(f.fileno())
        os.replace(tmp_path, path)
    except BaseException:
        with contextlib.suppress(OSError):
            os.unlink(tmp_path)
        raise


def _verify_signature_record(data: dict[str, Any], raw: bytes, index: int) -> bool:
    try:
        from cryptography.exceptions import InvalidSignature
        from cryptography.hazmat.primitives import hashes
        from cryptography.hazmat.primitives.asymmetric import ec, utils
        from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey
    except ModuleNotFoundError as exc:
        raise ModuleNotFoundError(
            "ATB signature verification requires the 'cryptography' package"
        ) from exc

    try:
        stored_hash = bytes.fromhex(_string_field(data, "bundle_hash"))
        signature = base64.b64decode(_string_field(data, "signature"), validate=True)
        pubkey = base64.b64decode(
            _string_field(data, "pubkey") or _string_field(data, "public_key"),
            validate=True,
        )
        prefix = _raw_prefix_before_record(raw, index)
        if hashlib.sha256(prefix).digest() != stored_hash:
            return False
        algorithm = (_string_field(data, "algorithm") or "ed25519").lower()
        if algorithm == "ed25519":
            Ed25519PublicKey.from_public_bytes(pubkey).verify(signature, stored_hash)
        elif algorithm == "ecdsa-p256":
            public_key = ec.EllipticCurvePublicKey.from_encoded_point(
                ec.SECP256R1(), pubkey
            )
            public_key.verify(
                signature,
                stored_hash,
                ec.ECDSA(utils.Prehashed(hashes.SHA256())),
            )
        else:
            raise UnsupportedAlgorithmError(
                f"unsupported bundle signature algorithm {algorithm!r}"
            )
    except UnsupportedAlgorithmError:
        raise
    except (binascii.Error, ValueError, TypeError, InvalidSignature):
        return False
    return True


def _raw_prefix_before_record(raw: bytes, target_index: int) -> bytes:
    record_index = 0
    line_start = 0
    while line_start < len(raw):
        line_end = raw.find(b"\n", line_start)
        if line_end == -1:
            line_end = len(raw)
        if line_end > line_start:
            if record_index == target_index:
                return raw[:line_start]
            record_index += 1
        if line_end == len(raw):
            break
        line_start = line_end + 1
    return b""


def _serialise_records(records: list[Record]) -> bytes:
    lines = [
        json.dumps({"event": record.event, "hash": record.hash}).encode("utf-8")
        for record in records
    ]
    return b"".join(line + b"\n" for line in lines)


def _recorded_manifest_version(records: list[Record]) -> int | None:
    """Return a comparable manifest version, if one is determinable.

    Returns 1 when there is no manifest record or the version field is absent
    or empty. Returns a value greater than :data:`MAX_MANIFEST_VERSION` when
    the recorded shape is determinable as unsupported. Returns ``None`` when
    the recorded value is not determinable, so callers do not broaden parsing
    or infer a version from payload shape.
    """
    if not records:
        return 1
    first = records[0].event
    if first.get("type") != MANIFEST_EVENT_TYPE:
        return 1

    data = first.get("data")
    if isinstance(data, str):
        try:
            parsed = json.loads(data)
        except (TypeError, ValueError):
            return None
        if not isinstance(parsed, dict):
            return None
        return _coerce_manifest_version(parsed.get("version"))
    if isinstance(data, dict):
        return _coerce_manifest_version(data.get("version"))
    return None


def _coerce_manifest_version(value: Any) -> int | None:
    if value is None:
        return 1
    if isinstance(value, bool):
        return None
    if value == "":
        return 1
    if isinstance(value, int):
        return value
    if isinstance(value, float):
        if not math.isfinite(value):
            return _UNSUPPORTED_VERSION_SENTINEL
        if value > MAX_MANIFEST_VERSION:
            truncated = int(value)
            # A fraction just above the maximum (e.g. 3.5) truncates to a
            # supported integer, but the recorded value itself is still newer
            # than this SDK supports, so report the sentinel.
            if truncated > MAX_MANIFEST_VERSION:
                return truncated
            return _UNSUPPORTED_VERSION_SENTINEL
        if value.is_integer():
            return int(value)
        return None
    if isinstance(value, str):
        # Mirror Go's Sscanf("%d") behaviour: trim surrounding whitespace and
        # read the leading decimal-digit run, ignoring any suffix. A missing
        # digit run remains non-determinable for legacy compatibility.
        match = re.match(r"[0-9]+", value.strip())
        if match is None:
            return None
        digits = match.group(0)
        # Avoid Python's int-string conversion limit for pathologically long
        # versions; any run this long is far above the supported maximum.
        if len(digits) > 18:
            return _UNSUPPORTED_VERSION_SENTINEL
        return int(digits)
    return None


def _string_field(data: dict[str, Any], key: str) -> str:
    value = data.get(key, "")
    return value if isinstance(value, str) else ""
