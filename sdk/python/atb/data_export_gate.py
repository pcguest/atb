"""Data export profile workflow helper."""

from __future__ import annotations

import time
from collections.abc import Callable, Mapping
from dataclasses import dataclass
from typing import Any, Literal, TypeVar

from atb.exceptions import ATBError
from atb.workflow_common import (
    WorkflowContext,
    actor_id_hash,
    canonical_digest,
    new_action_id,
    new_approval_id,
    value_digest,
)

T = TypeVar("T")


class DataExportDeniedError(ATBError):
    """Raised when a data export is denied in enforce mode."""


@dataclass(frozen=True)
class DataExportInput:
    action_type: str
    target_resource_id: str
    intended_effect: str
    action_parameters: Mapping[str, Any]
    subject_id: str | None = None
    action_id: str | None = None
    request_id: str | None = None


class DataExportGate:
    def __init__(
        self,
        bundle=None,
        *,
        mode: Literal["log_only", "enforce"] = "log_only",
        policy: Callable[[DataExportInput], Mapping[str, Any]] | None = None,
        record_approval: bool = True,
        auto_save: bool = False,
        save_path: str | None = None,
        actor_id: str | None = None,
        org_id: str | None = None,
        workspace_id: str | None = None,
    ) -> None:
        if mode not in {"log_only", "enforce"}:
            raise ValueError("mode must be one of: log_only, enforce")
        self.ctx = WorkflowContext(
            bundle,
            auto_save=auto_save,
            save_path=save_path,
            actor_id=actor_id,
            org_id=org_id,
            workspace_id=workspace_id,
        )
        self.mode = mode
        self.policy = policy or (lambda _: {"decision": "allow", "reason_codes": []})
        self.record_approval = record_approval
        self.actor_id = actor_id

    @property
    def bundle(self):
        return self.ctx.bundle

    def run(self, export_action: DataExportInput, fn: Callable[[], T]) -> T:
        self.ctx.bootstrap_request("data_export", export_action.request_id)
        action_id = new_action_id(export_action.action_id)

        self.ctx.emit(
            "data.export.precommit",
            {
                "action_id": action_id,
                "action_type": export_action.action_type,
                "action_parameters_digest": canonical_digest(dict(export_action.action_parameters)),
                "target_resource_id": export_action.target_resource_id,
                "intended_effect": export_action.intended_effect,
            },
        )

        decision = self.policy(export_action)
        self.ctx.emit(
            "ai.policy.decision",
            self.ctx.policy_payload(action_id, decision, export_action.subject_id),
        )
        if decision.get("decision") == "deny" and self.mode == "enforce":
            raise DataExportDeniedError(f"data export denied by policy for action_id {action_id}")

        started = time.perf_counter()
        try:
            result = fn()
            self.ctx.emit(
                "data.export.executed",
                {
                    "action_id": action_id,
                    "execution_outcome": "success",
                    "tool_receipt_digest": value_digest(result),
                    "execution_duration_ms": max(0, int((time.perf_counter() - started) * 1000)),
                },
            )
            self._maybe_record_approval(action_id)
            return result
        except Exception as exc:
            self.ctx.emit(
                "data.export.executed",
                {
                    "action_id": action_id,
                    "execution_outcome": "error",
                    "tool_receipt_digest": value_digest(exc),
                    "execution_duration_ms": max(0, int((time.perf_counter() - started) * 1000)),
                },
            )
            self._maybe_record_approval(action_id)
            raise

    def _maybe_record_approval(self, action_id: str) -> None:
        if not self.record_approval:
            return
        self.ctx.emit(
            "ai.human.approval",
            {
                "approval_id": new_approval_id(),
                "approver_id_hash": actor_id_hash(self.actor_id),
                "approval_outcome": "approved",
                "justification_digest": canonical_digest({"reason": "export approved"}),
                "action_id": action_id,
            },
        )
