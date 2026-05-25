"""Shared helpers for profile workflow SDK recorders."""

from __future__ import annotations

import hashlib
import uuid
from collections.abc import Mapping
from datetime import UTC, datetime
from typing import Any, Literal

from atb.bundle import Bundle
from atb.canonicalize import canonicalize

DEFAULT_WORKFLOW_SAVE_PATH = "run.atb/bundle.atb"


class WorkflowContext:
    """Append profile events with optional auto-save."""

    def __init__(
        self,
        bundle: Bundle | None = None,
        *,
        auto_save: bool = False,
        save_path: str | None = None,
        actor_id: str | None = None,
        org_id: str | None = None,
        workspace_id: str | None = None,
    ) -> None:
        self.bundle = bundle if bundle is not None else Bundle()
        self.auto_save = auto_save
        self.save_path = save_path
        self.actor_id = actor_id
        self.org_id = org_id
        self.workspace_id = workspace_id
        self._request_bootstrapped = False

    def emit(self, event_type: str, payload: Mapping[str, Any]) -> None:
        self.bundle.append(
            event_type,
            dict(payload),
            actor_id=self.actor_id,
            org_id=self.org_id,
            workspace_id=self.workspace_id,
            timestamp=_now_rfc3339(),
        )
        if self.auto_save:
            self.bundle.save(self.save_path)

    def bootstrap_request(self, purpose_tag: str, request_id: str | None = None) -> str:
        rid = (request_id or "").strip() or f"req_{uuid.uuid4().hex}"
        if not self._request_bootstrapped:
            self.emit(
                "ai.request.received",
                {
                    "request_id": rid,
                    "actor_id_hash": actor_id_hash(self.actor_id),
                    "purpose_tag": purpose_tag,
                },
            )
            self._request_bootstrapped = True
        return rid

    def policy_payload(
        self,
        action_id: str,
        decision: Mapping[str, Any],
        subject_id: str | None = None,
    ) -> dict[str, Any]:
        payload: dict[str, Any] = {
            "decision_id": action_id,
            "action_id": action_id,
            "policy_id": decision.get("policy_id", "local.workflow"),
            "policy_version": decision.get("policy_version", "v1"),
            "decision": decision["decision"],
            "decision_reason_codes": list(decision.get("reason_codes", [])),
        }
        subject_id_hash = subject_hash(subject_id, self.actor_id)
        if subject_id_hash is not None:
            payload["subject_id_hash"] = subject_id_hash
        return payload


def new_action_id(action_id: str | None = None) -> str:
    if action_id is not None and action_id.strip() != "":
        return action_id.strip()
    return f"act_{uuid.uuid4().hex}"


def new_job_id(job_id: str | None = None) -> str:
    if job_id is not None and job_id.strip() != "":
        return job_id.strip()
    return f"job_{uuid.uuid4().hex}"


def new_approval_id(approval_id: str | None = None) -> str:
    if approval_id is not None and approval_id.strip() != "":
        return approval_id.strip()
    return f"appr_{uuid.uuid4().hex}"


def canonical_digest(value: Any) -> str:
    return _sha256(canonicalize(value))


def value_digest(value: Any) -> str:
    try:
        encoded = canonicalize(value)
    except (TypeError, ValueError):
        encoded = repr(value).encode("utf-8")
    return _sha256(encoded)


def subject_hash(subject_id: str | None, actor_id: str | None) -> str | None:
    candidate = _normalize(subject_id)
    if candidate is None:
        candidate = _normalize(actor_id)
    if candidate is None:
        return None
    return _sha256(candidate.encode("utf-8"))


def actor_id_hash(actor_id: str | None) -> str:
    return subject_hash(None, actor_id) or _sha256(b"unknown")


def _sha256(value: bytes) -> str:
    return "sha256:" + hashlib.sha256(value).hexdigest()


def _normalize(value: str | None) -> str | None:
    if value is None:
        return None
    trimmed = value.strip()
    return trimmed or None


def now_rfc3339() -> str:
    return _now_rfc3339()


def _now_rfc3339() -> str:
    return datetime.now(UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z")
