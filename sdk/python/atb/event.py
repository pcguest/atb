"""ATB event model with the current Go runtime fields.

Quick start::

    from atb.event import Event
    event = Event(seq=0, prev_hash="0" * 64, type="atb.bundle.manifest", data={})
    event.to_dict()
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any


@dataclass
class Event:
    """Canonical ATB event model used for hashing.

    Args:
        seq: Event sequence number.
        prev_hash: Hex-encoded previous record hash.
        type: Event type identifier.
        data: JSON-like event payload.
        hash_algo: Optional hash algorithm marker.
        actor_id: Optional actor identity.
        org_id: Optional organisation identity.
        workspace_id: Optional workspace identity.
        timestamp: Optional RFC3339 timestamp.
        trace_id: Optional W3C trace identifier.
        span_id: Optional W3C span identifier.
        parent_span_id: Optional parent span identifier.

    Returns:
        A dataclass instance.

    Raises:
        None.
    """

    seq: int
    prev_hash: str
    type: str
    data: Any
    hash_algo: str | None = None
    actor_id: str | None = None
    org_id: str | None = None
    workspace_id: str | None = None
    timestamp: str | None = None
    trace_id: str | None = None
    span_id: str | None = None
    parent_span_id: str | None = None

    def to_dict(self) -> dict[str, Any]:
        """Serialise event to a dictionary.

        Args:
            None.

        Returns:
            Event dictionary with unset optional fields omitted.

        Raises:
            None.
        """
        out: dict[str, Any] = {
            "seq": self.seq,
            "prev_hash": self.prev_hash,
            "type": self.type,
            "data": self.data,
        }
        optional_fields = {
            "hash_algo": self.hash_algo,
            "actor_id": self.actor_id,
            "org_id": self.org_id,
            "workspace_id": self.workspace_id,
            "timestamp": self.timestamp,
            "trace_id": self.trace_id,
            "span_id": self.span_id,
            "parent_span_id": self.parent_span_id,
        }
        for key, value in optional_fields.items():
            if value is not None:
                out[key] = value
        return out
