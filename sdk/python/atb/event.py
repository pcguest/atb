"""ATB event model with optional multi-tenant identity fields."""

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
    actor_id: str | None = None
    org_id: str | None = None
    workspace_id: str | None = None

    def to_dict(self) -> dict[str, Any]:
        """Serialize event to dict, omitting unset optional identity fields."""
        out: dict[str, Any] = {
            "seq": self.seq,
            "prev_hash": self.prev_hash,
            "type": self.type,
            "data": self.data,
        }
        if self.actor_id is not None:
            out["actor_id"] = self.actor_id
        if self.org_id is not None:
            out["org_id"] = self.org_id
        if self.workspace_id is not None:
            out["workspace_id"] = self.workspace_id
        return out
