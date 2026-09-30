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


def _validate_acquisition_string(value: Any, label: str) -> str:
    if not isinstance(value, str):
        raise TypeError(f"{label} must be a string")
    return value


def _is_blank(value: str) -> bool:
    return value.strip() == ""


def _normalize_acquisition_checkpoint(value: Any) -> dict[str, Any]:
    if isinstance(value, AcquisitionCheckpoint):
        raw: dict[str, Any] = {}
        for field_name in _ACQUISITION_CHECKPOINT_REQUIRED_FIELDS:
            field_value = getattr(value, field_name)
            if field_value is not None:
                raw[field_name] = field_value
        if value.adapter_version is not None:
            raw["adapter_version"] = value.adapter_version
    elif isinstance(value, dict):
        raw = dict(value)
    else:
        raise TypeError("acquisition.checkpoint must be a dict")

    out: dict[str, Any] = {}
    # Required checkpoint fields have no `omitempty` in the Go runtime and are
    # always emitted, even when blank.
    for field_name in _ACQUISITION_CHECKPOINT_REQUIRED_FIELDS:
        if field_name not in raw:
            raise TypeError(f"acquisition.checkpoint.{field_name} must be a string")
        out[field_name] = _validate_acquisition_string(
            raw[field_name], f"acquisition.checkpoint.{field_name}"
        )
    if "adapter_version" in raw:
        # adapter_version has `omitempty` in Go: blank is treated as unset.
        adapter_version = _validate_acquisition_string(
            raw["adapter_version"], "acquisition.checkpoint.adapter_version"
        )
        if not _is_blank(adapter_version):
            out["adapter_version"] = adapter_version
    return out


def normalize_acquisition(value: Any) -> dict[str, Any] | None:
    """Return a canonical acquisition mapping, or ``None`` when unset.

    Fail-closed: only recognised canonical fields are preserved. Unknown
    acquisition/checkpoint fields are dropped rather than passed through, so
    they never become canonical evidence. Malformed values raise ``TypeError``,
    including explicit ``None`` for a recognised field (distinct from an absent
    key, which is treated as unset).

    ``None`` omits the property. Every top-level acquisition field and
    ``checkpoint.adapter_version`` have ``omitempty`` in the Go runtime, so
    blank values are treated as unset and omitted. The five required checkpoint
    fields have no ``omitempty`` in Go and are always emitted.

    Args:
        value: An :class:`Acquisition`, a :class:`dict`, or ``None``.

    Returns:
        A fresh mapping with only recognised fields, or ``None`` when *value*
        is ``None``.

    Raises:
        TypeError: When the structure or a field type is invalid.
    """
    if value is None:
        return None
    if isinstance(value, Acquisition):
        raw: dict[str, Any] = {}
        for field_name in _ACQUISITION_STRING_FIELDS:
            field_value = getattr(value, field_name)
            if field_value is not None:
                raw[field_name] = field_value
        if value.checkpoint is not None:
            raw["checkpoint"] = value.checkpoint
    elif isinstance(value, dict):
        raw = dict(value)
    else:
        raise TypeError("acquisition must be an Acquisition or dict")

    out: dict[str, Any] = {}
    for field_name in _ACQUISITION_STRING_FIELDS:
        if field_name not in raw:
            continue
        field_value = _validate_acquisition_string(
            raw[field_name], f"acquisition.{field_name}"
        )
        if not _is_blank(field_value):
            out[field_name] = field_value
    if "checkpoint" in raw:
        checkpoint_value = raw["checkpoint"]
        if checkpoint_value is None:
            raise TypeError("acquisition.checkpoint must be a dict")
        out["checkpoint"] = _normalize_acquisition_checkpoint(checkpoint_value)
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
            TypeError: When ``acquisition`` is present but malformed.
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
