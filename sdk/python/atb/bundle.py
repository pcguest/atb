"""
ATB Bundle — the primary interface for creating and managing ATB trace bundles.
"""

from __future__ import annotations

import json
import os
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from atb.exceptions import ATBVerificationError
from atb.hash import GENESIS_HASH, compute_hash

#: Default bundle directory and file name.
BUNDLE_DIR = "run.atb"
BUNDLE_FILE = "bundle.atb"


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

    records: list[Record] = field(default_factory=list)

    # ------------------------------------------------------------------
    # Mutation
    # ------------------------------------------------------------------

    def append(self, event_type: str, data: Any) -> Record:
        """Append a new event to the bundle.

        Args:
            event_type: A dot-namespaced event type string (e.g. ``"dev.session"``).
            data: Arbitrary JSON-serialisable payload.

        Returns:
            The newly created :class:`Record`.
        """
        prev_hash = self.records[-1].hash if self.records else GENESIS_HASH
        event: dict[str, Any] = {
            "seq": len(self.records) + 1,
            "prev_hash": prev_hash,
            "type": event_type,
            "data": data,
        }
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
        with resolved.open("r", encoding="utf-8") as fh:
            for line in fh:
                line = line.strip()
                if not line:
                    continue
                obj = json.loads(line)
                bundle.records.append(Record(event=obj["event"], hash=obj["hash"]))
        return bundle

    # ------------------------------------------------------------------
    # Convenience
    # ------------------------------------------------------------------

    def __len__(self) -> int:
        return len(self.records)

    def __repr__(self) -> str:
        return f"Bundle(records={len(self.records)})"
