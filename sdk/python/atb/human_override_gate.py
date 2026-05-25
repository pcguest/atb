"""Human override profile workflow helper."""

from __future__ import annotations

import time
from collections.abc import Callable, Mapping
from dataclasses import dataclass
from typing import Any, Literal, TypeVar

from atb.exceptions import ATBError
from atb.workflow_common import (
    WorkflowContext,
    canonical_digest,
    new_action_id,
    new_approval_id,
    value_digest,
)

T = TypeVar("T")


class HumanOverrideDeniedError(ATBError):
    """Raised when a human override is denied in enforce mode."""


@dataclass(frozen=True)
class HumanOverrideActionInput:
    action_type: str
    target_resource_id: str
    intended_effect: str
    action_parameters: Mapping[str, Any]
    action_id: str | None = None
    request_id: str | None = None


@dataclass(frozen=True)
class HumanOverrideApprovalInput:
    approver_id_hash: str
    approval_outcome: Literal["approved", "denied"] = "approved"
    justification_digest: str | None = None
    approval_id: str | None = None


class HumanOverrideGate:
    def __init__(
        self,
        bundle=None,
        *,
        mode: Literal["log_only", "enforce"] = "log_only",
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

    @property
    def bundle(self):
        return self.ctx.bundle

    def run(
        self,
        action: HumanOverrideActionInput,
        approval: HumanOverrideApprovalInput,
        fn: Callable[[], T],
    ) -> T:
        self.ctx.bootstrap_request("human_override", action.request_id)
        action_id = new_action_id(action.action_id)

        self.ctx.emit(
            "ai.human.approval",
            {
                "approval_id": new_approval_id(approval.approval_id),
                "approver_id_hash": approval.approver_id_hash,
                "approval_outcome": approval.approval_outcome,
                "justification_digest": approval.justification_digest
                or canonical_digest({"reason": "human override"}),
                "action_id": action_id,
            },
        )
        self.ctx.emit(
            "ai.action.precommit",
            {
                "action_id": action_id,
                "action_type": action.action_type,
                "action_parameters_digest": canonical_digest(dict(action.action_parameters)),
                "target_resource_id": action.target_resource_id,
                "intended_effect": action.intended_effect,
            },
        )

        if approval.approval_outcome == "denied" and self.mode == "enforce":
            raise HumanOverrideDeniedError(f"human override denied for action_id {action_id}")

        started = time.perf_counter()
        try:
            result = fn()
            self.ctx.emit(
                "ai.action.executed",
                {
                    "action_id": action_id,
                    "execution_outcome": "success",
                    "tool_receipt_digest": value_digest(result),
                    "execution_duration_ms": max(0, int((time.perf_counter() - started) * 1000)),
                },
            )
            return result
        except Exception as exc:
            self.ctx.emit(
                "ai.action.executed",
                {
                    "action_id": action_id,
                    "execution_outcome": "error",
                    "tool_receipt_digest": value_digest(exc),
                    "execution_duration_ms": max(0, int((time.perf_counter() - started) * 1000)),
                },
            )
            raise
