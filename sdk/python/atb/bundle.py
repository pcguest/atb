"""
ATB Bundle — the primary interface for creating and managing ATB trace bundles.
"""

from __future__ import annotations

import json
import secrets
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from atb.event import Event
from atb.exceptions import ATBVerificationError
from atb.hash import GENESIS_HASH, compute_hash

#: Default bundle directory and file name.
BUNDLE_DIR = "run.atb"
BUNDLE_FILE = "bundle.atb"
MANIFEST_EVENT_TYPE = "atb.bundle.manifest"
MANIFEST_VERSION = "1"


@dataclass
class Record:
    """A single record in an ATB bundle (event + its hash)."""

    event: dict[str, Any]
    hash: str


@dataclass
class Bundle:
    """An in-memory ATB bundle.

    Usage::

        bundle = Bundle()
        bundle.append("dev.session", {"date": "2025-01-15"})
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
        object.__setattr__(self, 'name', goal or name or "untitled")
        object.__setattr__(self, 'records', records if records is not None else [])
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
    ) -> Record:
        """Append a new event to the bundle.

        Args:
            event_type: A dot-namespaced event type string (e.g. ``"dev.session"``).
            data: Arbitrary JSON-serialisable payload.
            actor_id: Optional actor identifier.
            org_id: Optional organization identifier.
            workspace_id: Optional workspace identifier.

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
        )
        event = event_obj.to_dict()
        h = compute_hash(event, prev_hash)
        record = Record(event=event, hash=h)
        self.records.append(record)
        return record

    # ------------------------------------------------------------------
    # Verification
    # ------------------------------------------------------------------

    def verify(self) -> None:
        """Verify the integrity of the entire bundle.

        Raises:
            ATBVerificationError: If any event's hash does not match the
                recomputed value, indicating tampering.
        """
        prev = GENESIS_HASH
        for i, record in enumerate(self.records):
            event = dict(record.event)
            event["prev_hash"] = prev
            if i == 0 and self._has_manifest_record:
                event["seq"] = 0
            elif self._has_manifest_record:
                event["seq"] = i
            else:
                event["seq"] = i + 1
            computed = compute_hash(event, prev)
            if computed != record.hash:
                raise ATBVerificationError(
                    f"Tamper detected at event {i + 1} (seq {event['seq']}): "
                    f"expected {record.hash!r}, computed {computed!r}",
                    event_index=i,
                    expected_hash=record.hash,
                    computed_hash=computed,
                )
            prev = computed

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
        with resolved.open("w", encoding="utf-8") as fh:
            for record in self.records:
                fh.write(json.dumps({"event": record.event, "hash": record.hash}))
                fh.write("\n")
        return resolved

    @classmethod
    def load(cls, path: str | Path | None = None) -> "Bundle":
        """Load a bundle from *path*.

        Args:
            path: File path to read. Defaults to ``run.atb/bundle.atb``.

        Returns:
            A populated :class:`Bundle` instance.
        """
        resolved = Path(path) if path else Path(BUNDLE_DIR) / BUNDLE_FILE
        bundle = cls()
        bundle.records.clear()
        with resolved.open("r", encoding="utf-8") as fh:
            for line in fh:
                line = line.strip()
                if not line:
                    continue
                obj = json.loads(line)
                bundle.records.append(Record(event=obj["event"], hash=obj["hash"]))
        return bundle

    # ------------------------------------------------------------------
    # Encryption
    # ------------------------------------------------------------------

    def encrypt(self, password: str) -> bytes:
        """Encrypt this bundle to ATBE bytes."""
        from atb.encrypt import encrypt_bundle

        return encrypt_bundle(self, password)

    @classmethod
    def decrypt(cls, password: str, data: bytes) -> "Bundle":
        """Decrypt ATBE bytes and return a verified bundle."""
        from atb.encrypt import decrypt_bundle

        return decrypt_bundle(password, data)

    # ------------------------------------------------------------------
    # Convenience
    # ------------------------------------------------------------------

    def __len__(self) -> int:
        return len(self.records)

    def __repr__(self) -> str:
        return f"Bundle(records={len(self.records)})"

    @property
    def _has_manifest_record(self) -> bool:
        return bool(self.records) and self.records[0].event.get("type") == MANIFEST_EVENT_TYPE

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
    return datetime.now(UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z")
