"""
ATB hash-chaining implementation.

Each event hash is computed as:
    SHA256(prev_hash_bytes || canonical_json_bytes)

where prev_hash_bytes is the UTF-8 encoding of the previous event's hex hash,
and canonical_json_bytes is the RFC 8785 canonical JSON of the event object.
"""

from __future__ import annotations

import hashlib
from typing import Any

from atb.canonicalize import canonicalize

#: The sentinel previous-hash used for the first event in a bundle.
GENESIS_HASH = "0" * 64

# Public alias for import convenience.
genesis_hash = GENESIS_HASH


def compute_hash(event: dict[str, Any], prev_hash: str) -> str:
    """Compute the SHA-256 hash for *event* given *prev_hash*.

    Args:
        event: The event dict (must include ``seq``, ``prev_hash``, ``type``, ``data``).
        prev_hash: The hex-encoded hash of the preceding event (or GENESIS_HASH).

    Returns:
        The hex-encoded SHA-256 hash string (64 characters).
    """
    canonical_bytes = canonicalize(event)
    digest = hashlib.sha256(prev_hash.encode("utf-8") + canonical_bytes).hexdigest()
    return digest


def chain_events(events: list[dict[str, Any]]) -> list[str]:
    """Compute and return the hash list for a sequence of events.

    Each event dict is mutated in-place to set ``seq`` and ``prev_hash``.

    Args:
        events: List of event dicts with at least ``type`` and ``data`` keys.

    Returns:
        List of hex-encoded SHA-256 hashes, one per event.
    """
    hashes: list[str] = []
    prev = GENESIS_HASH
    for i, event in enumerate(events):
        event["seq"] = i + 1
        event["prev_hash"] = prev
        h = compute_hash(event, prev)
        hashes.append(h)
        prev = h
    return hashes
