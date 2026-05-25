"""Policy decision profile recorder."""

from __future__ import annotations

from collections.abc import Mapping
from dataclasses import dataclass
from typing import Any

from atb.workflow_common import WorkflowContext, canonical_digest, new_action_id


@dataclass(frozen=True)
class PolicyDecisionActionInput:
    action_type: str
    action_parameters: Mapping[str, Any]
    action_id: str | None = None
    subject_id: str | None = None
    request_id: str | None = None


class PolicyDecisionRecorder:
    def __init__(
        self,
        bundle=None,
        *,
        auto_save: bool = False,
        save_path: str | None = None,
        actor_id: str | None = None,
        org_id: str | None = None,
        workspace_id: str | None = None,
    ) -> None:
        self.ctx = WorkflowContext(
            bundle,
            auto_save=auto_save,
            save_path=save_path,
            actor_id=actor_id,
            org_id=org_id,
            workspace_id=workspace_id,
        )

    @property
    def bundle(self):
        return self.ctx.bundle

    def record(self, action: PolicyDecisionActionInput, decision: Mapping[str, Any]) -> str:
        self.ctx.bootstrap_request("policy_decision", action.request_id)
        action_id = new_action_id(action.action_id)
        self.ctx.emit(
            "ai.action.precommit",
            {
                "action_id": action_id,
                "action_type": action.action_type,
                "action_parameters_digest": canonical_digest(dict(action.action_parameters)),
            },
        )
        self.ctx.emit(
            "ai.policy.decision",
            self.ctx.policy_payload(action_id, decision, action.subject_id),
        )
        return action_id
