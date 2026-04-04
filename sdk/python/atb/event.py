"""ATB event model with the current Go runtime fields."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

@dataclass
class Event:
    """Canonical ATB event model used for hashing."""

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
        """Serialize event to dict, omitting unset optional fields."""
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
