"""ATB event model with the current Go runtime fields.

Quick start::

    from atb.event import Event
    event = Event(seq=0, prev_hash="0" * 64, type="atb.bundle.manifest", data={})
    event.to_dict()
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

# Canonical acquisition string fields (all optional, omitted when unset).
_ACQUISITION_STRING_FIELDS = (
    "mode",
    "source_system",
    "source_record_id",
    "source_timestamp",
    "acquired_at",
    "source_digest",
    "adapter",
    "adapter_version",
)

# Canonical checkpoint fields the runtime always emits when a checkpoint is
# present (no omitempty on the Go struct).
_ACQUISITION_CHECKPOINT_REQUIRED_FIELDS = (
    "source_system",
    "acquisition_stream",
    "position",
    "observed_at",
    "adapter",
)


@dataclass
class AcquisitionCheckpoint:
    """Operational checkpoint for one acquisition stream.

    A checkpoint is operational position, not evidence truth.

    Args:
        source_system: Source system the checkpoint belongs to.
        acquisition_stream: Logical stream identifier.
        position: Adapter-defined cursor/position within the stream.
        observed_at: RFC3339 timestamp the position was observed.
        adapter: Adapter identifier that produced the checkpoint.
        adapter_version: Optional adapter version.
    """

    source_system: str
    acquisition_stream: str
    position: str
    observed_at: str
    adapter: str
    adapter_version: str | None = None


@dataclass
class Acquisition:
    """Bounded acquisition provenance for an event.

    Acquisition describes how/from where evidence entered ATB. It is provenance,
    not evidence truth, producer authority, or verified identity.

    Args:
        mode: Acquisition mode (e.g. ``"retrospective"``).
        source_system: Source system the evidence came from.
        source_record_id: Source-native record identifier.
        source_timestamp: RFC3339 timestamp from the source.
        acquired_at: RFC3339 timestamp ATB acquired the evidence.
        source_digest: Digest of the observed source representation.
        adapter: Adapter identifier.
        adapter_version: Optional adapter version.
        checkpoint: Optional operational checkpoint.
    """

    mode: str | None = None
    source_system: str | None = None
    source_record_id: str | None = None
    source_timestamp: str | None = None
    acquired_at: str | None = None
    source_digest: str | None = None
    adapter: str | None = None
    adapter_version: str | None = None
    checkpoint: AcquisitionCheckpoint | dict[str, Any] | None = None


def _require_acquisition_string(
    mapping: dict[str, Any], key: str, label: str
) -> str:
    value = mapping.get(key)
    if not isinstance(value, str):
        raise TypeError(f"{label}.{key} must be a string")
    return value


def _normalize_acquisition_checkpoint(value: Any) -> dict[str, Any]:
    if isinstance(value, AcquisitionCheckpoint):
        raw: dict[str, Any] = {
            "source_system": value.source_system,
            "acquisition_stream": value.acquisition_stream,
            "position": value.position,
            "observed_at": value.observed_at,
            "adapter": value.adapter,
            "adapter_version": value.adapter_version,
        }
    elif isinstance(value, dict):
        raw = dict(value)
    else:
        raise TypeError("acquisition.checkpoint must be a mapping")

    out: dict[str, Any] = {}
    for field_name in _ACQUISITION_CHECKPOINT_REQUIRED_FIELDS:
        out[field_name] = _require_acquisition_string(
            raw, field_name, "acquisition.checkpoint"
        )
    if raw.get("adapter_version") is not None:
        out["adapter_version"] = _require_acquisition_string(
            raw, "adapter_version", "acquisition.checkpoint"
        )
    return out


def normalize_acquisition(value: Any) -> dict[str, Any] | None:
    """Return a canonical acquisition mapping, or ``None`` when unset.

    Fail-closed: only recognised canonical fields are preserved. Unknown
    acquisition/checkpoint fields are dropped rather than passed through, so
    they never become canonical evidence. Malformed values raise ``TypeError``.

    Only ``None`` omits the property. An explicitly supplied empty
    :class:`Acquisition` is preserved as an empty mapping, matching the Go
    pointer and TypeScript behaviour for an explicit empty acquisition.

    Args:
        value: An :class:`Acquisition`, a mapping, or ``None``.

    Returns:
        A fresh mapping with only recognised fields, or ``None`` when *value*
        is ``None``.

    Raises:
        TypeError: When the structure or a field type is invalid.
    """
    if value is None:
        return None
    if isinstance(value, Acquisition):
        raw: dict[str, Any] = {
            "mode": value.mode,
            "source_system": value.source_system,
            "source_record_id": value.source_record_id,
            "source_timestamp": value.source_timestamp,
            "acquired_at": value.acquired_at,
            "source_digest": value.source_digest,
            "adapter": value.adapter,
            "adapter_version": value.adapter_version,
            "checkpoint": value.checkpoint,
        }
    elif isinstance(value, dict):
        raw = dict(value)
    else:
        raise TypeError("acquisition must be an Acquisition or mapping")

    out: dict[str, Any] = {}
    for field_name in _ACQUISITION_STRING_FIELDS:
        if raw.get(field_name) is not None:
            out[field_name] = _require_acquisition_string(
                raw, field_name, "acquisition"
            )
    if raw.get("checkpoint") is not None:
        out["checkpoint"] = _normalize_acquisition_checkpoint(raw["checkpoint"])
    return out


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
        acquisition: Optional bounded acquisition provenance.

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
    acquisition: Acquisition | dict[str, Any] | None = None

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
        if self.acquisition is not None:
            out["acquisition"] = normalize_acquisition(self.acquisition)
        return out
