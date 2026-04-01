from __future__ import annotations

import hashlib
import time
import uuid
from collections.abc import Awaitable, Callable, Mapping
from dataclasses import dataclass, field
from typing import Any, Literal, TypeVar

from atb.bundle import Bundle
from atb.canonicalize import canonicalize
from atb.exceptions import ATBError

T = TypeVar("T")


class ActionGateDeniedError(ATBError):
    pass


@dataclass(frozen=True)
class ActionGateInput:
    action_type: str
    target_resource_id: str
    intended_effect: str
    action_parameters: Mapping[str, Any]
    subject_id: str | None = None
    action_id: str | None = None
    policy_context: Mapping[str, Any] | None = None


@dataclass(frozen=True)
class ActionGateDecision:
    decision: Literal["allow", "deny"]
    reason_codes: tuple[str, ...] = field(default_factory=tuple)
    policy_id: str = "local.action_gate"
    policy_version: str = "v1"


def _default_policy(_: ActionGateInput) -> ActionGateDecision:
    return ActionGateDecision(decision="allow")


class ActionGate:
    def __init__(
        self,
        bundle: Bundle | None = None,
        *,
        mode: Literal["log_only", "enforce"] = "log_only",
        policy: Callable[[ActionGateInput], ActionGateDecision] | None = None,
        auto_save: bool = False,
        save_path: str | None = None,
        actor_id: str | None = None,
        org_id: str | None = None,
        workspace_id: str | None = None,
    ) -> None:
        if mode not in {"log_only", "enforce"}:
            raise ValueError("mode must be one of: log_only, enforce")
        self.bundle = bundle if bundle is not None else Bundle()
        self.mode = mode
        self.policy = policy if policy is not None else _default_policy
        self.auto_save = auto_save
        self.save_path = save_path
        self.actor_id = actor_id
        self.org_id = org_id
        self.workspace_id = workspace_id

    def run(self, action: ActionGateInput, fn: Callable[[], T]) -> T:
        action_id = self._action_id(action)
        self._emit(
            "ai.action.precommit",
            {
                "action_id": action_id,
                "action_type": action.action_type,
                "action_parameters_digest": _canonical_digest(dict(action.action_parameters)),
                "target_resource_id": action.target_resource_id,
                "intended_effect": action.intended_effect,
            },
        )

        decision = self.policy(action)
        self._emit("ai.policy.decision", self._policy_payload(action, action_id, decision))
        if decision.decision == "deny" and self.mode == "enforce":
            raise ActionGateDeniedError(f"action denied by policy for action_id {action_id}")

        started_at = time.perf_counter()
        try:
            result = fn()
        except Exception as exc:
            self._emit("ai.action.executed", self._executed_payload(action, action_id, started_at, exc, "error"))
            raise

        self._emit("ai.action.executed", self._executed_payload(action, action_id, started_at, result, "success"))
        return result

    async def arun(self, action: ActionGateInput, fn: Callable[[], Awaitable[T]]) -> T:
        action_id = self._action_id(action)
        self._emit(
            "ai.action.precommit",
            {
                "action_id": action_id,
                "action_type": action.action_type,
                "action_parameters_digest": _canonical_digest(dict(action.action_parameters)),
                "target_resource_id": action.target_resource_id,
                "intended_effect": action.intended_effect,
            },
        )

        decision = self.policy(action)
        self._emit("ai.policy.decision", self._policy_payload(action, action_id, decision))
        if decision.decision == "deny" and self.mode == "enforce":
            raise ActionGateDeniedError(f"action denied by policy for action_id {action_id}")

        started_at = time.perf_counter()
        try:
            result = await fn()
        except Exception as exc:
            self._emit("ai.action.executed", self._executed_payload(action, action_id, started_at, exc, "error"))
            raise

        self._emit("ai.action.executed", self._executed_payload(action, action_id, started_at, result, "success"))
        return result

    def _action_id(self, action: ActionGateInput) -> str:
        if action.action_id is not None and action.action_id.strip() != "":
            return action.action_id.strip()
        return f"act_{uuid.uuid4().hex}"

    def _emit(self, event_type: str, payload: dict[str, Any]) -> None:
        self.bundle.append(
            event_type,
            payload,
            actor_id=self.actor_id,
            org_id=self.org_id,
            workspace_id=self.workspace_id,
        )
        if self.auto_save:
            self.bundle.save(self.save_path)

    def _policy_payload(
        self,
        action: ActionGateInput,
        action_id: str,
        decision: ActionGateDecision,
    ) -> dict[str, Any]:
        payload: dict[str, Any] = {
            "decision_id": action_id,
            "action_id": action_id,
            "policy_id": decision.policy_id,
            "policy_version": decision.policy_version,
            "decision": decision.decision,
            "decision_reason_codes": list(decision.reason_codes),
        }
        subject_id_hash = _subject_hash(action.subject_id, self.actor_id)
        if subject_id_hash is not None:
            payload["subject_id_hash"] = subject_id_hash
        return payload

    def _executed_payload(
        self,
        action: ActionGateInput,
        action_id: str,
        started_at: float,
        receipt: Any,
        execution_outcome: str,
    ) -> dict[str, Any]:
        return {
            "action_id": action_id,
            "action_type": action.action_type,
            "tool_receipt_digest": _value_digest(receipt),
            "execution_duration_ms": max(0, int((time.perf_counter() - started_at) * 1000)),
            "execution_outcome": execution_outcome,
        }


def _subject_hash(subject_id: str | None, actor_id: str | None) -> str | None:
    candidate = _normalize(subject_id)
    if candidate is None:
        candidate = _normalize(actor_id)
    if candidate is None:
        return None
    return _sha256(candidate.encode("utf-8"))


def _value_digest(value: Any) -> str:
    try:
        encoded = canonicalize(value)
    except (TypeError, ValueError):
        encoded = repr(value).encode("utf-8")
    return _sha256(encoded)


def _canonical_digest(value: Any) -> str:
    return _sha256(canonicalize(value))


def _sha256(value: bytes) -> str:
    return "sha256:" + hashlib.sha256(value).hexdigest()


def _normalize(value: str | None) -> str | None:
    if value is None:
        return None
    trimmed = value.strip()
    if trimmed == "":
        return None
    return trimmed
