"""Action-gating helpers that record precommit, policy, and execution events.

The module exposes ``ActionGate`` for wrapping local operations,
``ActionGateInput`` and ``ActionGateDecision`` for policy callbacks, and
``ActionGateDeniedError`` for enforce-mode denials.

Quick start::

    from atb import ActionGate, ActionGateInput
    gate = ActionGate()
    gate.run(ActionGateInput("tool", "resource", "read", {}), lambda: "ok")
"""

from __future__ import annotations

import hashlib
import time
import uuid
from collections.abc import Awaitable, Callable, Mapping
from dataclasses import dataclass, field
from datetime import UTC, datetime
from typing import Any, Literal, TypeVar

from atb.bundle import Bundle
from atb.canonicalize import canonicalize
from atb.exceptions import ATBError
from atb.workflow_common import WorkflowEventSink

T = TypeVar("T")


class ActionGateDeniedError(ATBError):
    """Raised when an action is denied in enforce mode.

    Args:
        *args: Positional error message arguments passed to ``Exception``.

    Returns:
        None.

    Raises:
        None.
    """


@dataclass(frozen=True)
class ActionGateInput:
    """Input metadata describing an action before it executes.

    Args:
        action_type: Stable action or tool type.
        target_resource_id: Resource the action intends to affect.
        intended_effect: Human-readable effect statement.
        action_parameters: JSON-like parameters supplied to the action.
        subject_id: Optional subject identifier to hash into policy payloads.
        action_id: Optional caller-provided action identifier.
        policy_context: Optional policy-specific context.

    Returns:
        A frozen dataclass instance.

    Raises:
        None.
    """

    action_type: str
    target_resource_id: str
    intended_effect: str
    action_parameters: Mapping[str, Any]
    subject_id: str | None = None
    action_id: str | None = None
    policy_context: Mapping[str, Any] | None = None
    principal: "ActionPrincipal | None" = None
    effective_scope: str | None = None


@dataclass(frozen=True)
class ActionPrincipal:
    """Acting principal: who initiated the action, and for whom when delegated.

    Args:
        type: One of "human", "agent", or "tool".
        id_hash: Hashed principal identifier.
        on_behalf_of: Optional hashed identifier the action is delegated for.
    """

    type: str
    id_hash: str
    on_behalf_of: str | None = None


def _principal_payload(principal: "ActionPrincipal | None") -> dict[str, Any] | None:
    if principal is None or not principal.type or not principal.id_hash:
        return None
    payload: dict[str, Any] = {"type": principal.type, "id_hash": principal.id_hash}
    if principal.on_behalf_of:
        payload["on_behalf_of"] = principal.on_behalf_of
    return payload


@dataclass(frozen=True)
class ActionGateDecision:
    """Policy decision returned by an action gate callback.

    Args:
        decision: Either ``"allow"`` or ``"deny"``.
        reason_codes: Optional machine-readable decision reason codes.
        policy_id: Policy identifier recorded with the decision.
        policy_version: Policy version recorded with the decision.

    Returns:
        A frozen dataclass instance.

    Raises:
        None.
    """

    decision: Literal["allow", "deny"]
    reason_codes: tuple[str, ...] = field(default_factory=tuple)
    policy_id: str = "local.action_gate"
    policy_version: str = "v1"


def _default_policy(_: ActionGateInput) -> ActionGateDecision:
    return ActionGateDecision(decision="allow")


class ActionGate:
    """Wrap local actions and append ATB action-gate events.

    Args:
        bundle: Optional bundle to append to. A new bundle is created when
            omitted.
        mode: ``"log_only"`` records denials but runs the action; ``"enforce"``
            raises on denials.
        policy: Optional callable that returns an ``ActionGateDecision``.
        auto_save: Save the bundle after each emitted event when true.
        save_path: Optional path used when ``auto_save`` is true.
        actor_id: Optional actor identity metadata.
        org_id: Optional organisation identity metadata.
        workspace_id: Optional workspace identity metadata.

    Returns:
        An ``ActionGate`` instance.

    Raises:
        ValueError: If ``mode`` is not recognised.
    """

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
        event_sink: WorkflowEventSink | None = None,
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
        self.event_sink = event_sink

    def run(self, action: ActionGateInput, fn: Callable[[], T]) -> T:
        """Run a synchronous action under gate policy.

        Args:
            action: Action metadata recorded before evaluation.
            fn: Zero-argument callable to execute after policy evaluation.

        Returns:
            The value returned by ``fn``.

        Raises:
            ActionGateDeniedError: If policy denies and mode is ``"enforce"``.
            Exception: Re-raises any exception produced by ``fn``.
        """
        action_id = self._action_id(action)
        precommit: dict[str, Any] = {
            "action_id": action_id,
            "action_type": action.action_type,
            "action_parameters_digest": _canonical_digest(dict(action.action_parameters)),
            "target_resource_id": action.target_resource_id,
            "intended_effect": action.intended_effect,
        }
        principal = _principal_payload(action.principal)
        if principal is not None:
            precommit["principal"] = principal
        self._emit("ai.action.precommit", precommit)

        decision = self.policy(action)
        self._emit("ai.policy.decision", self._policy_payload(action, action_id, decision))
        if decision.decision == "deny" and self.mode == "enforce":
            raise ActionGateDeniedError(f"action denied by policy for action_id {action_id}")

        started_at = time.perf_counter()
        try:
            result = fn()
        except Exception as exc:
            # A privileged action that raised did not execute successfully:
            # record the forensic ai.action.error event, not a success-shaped
            # executed record, so SDK incidents match capture-side incidents.
            self._emit("ai.action.error", self._error_payload(action, action_id, exc))
            raise

        self._emit("ai.action.executed", self._executed_payload(action, action_id, started_at, result))
        return result

    async def arun(self, action: ActionGateInput, fn: Callable[[], Awaitable[T]]) -> T:
        """Run an asynchronous action under gate policy.

        Args:
            action: Action metadata recorded before evaluation.
            fn: Zero-argument async callable to execute after policy evaluation.

        Returns:
            The awaited value returned by ``fn``.

        Raises:
            ActionGateDeniedError: If policy denies and mode is ``"enforce"``.
            Exception: Re-raises any exception produced by ``fn``.
        """
        action_id = self._action_id(action)
        precommit: dict[str, Any] = {
            "action_id": action_id,
            "action_type": action.action_type,
            "action_parameters_digest": _canonical_digest(dict(action.action_parameters)),
            "target_resource_id": action.target_resource_id,
            "intended_effect": action.intended_effect,
        }
        principal = _principal_payload(action.principal)
        if principal is not None:
            precommit["principal"] = principal
        self._emit("ai.action.precommit", precommit)

        decision = self.policy(action)
        self._emit("ai.policy.decision", self._policy_payload(action, action_id, decision))
        if decision.decision == "deny" and self.mode == "enforce":
            raise ActionGateDeniedError(f"action denied by policy for action_id {action_id}")

        started_at = time.perf_counter()
        try:
            result = await fn()
        except Exception as exc:
            # A privileged action that raised did not execute successfully:
            # record the forensic ai.action.error event, not a success-shaped
            # executed record, so SDK incidents match capture-side incidents.
            self._emit("ai.action.error", self._error_payload(action, action_id, exc))
            raise

        self._emit("ai.action.executed", self._executed_payload(action, action_id, started_at, result))
        return result

    def _action_id(self, action: ActionGateInput) -> str:
        if action.action_id is not None and action.action_id.strip() != "":
            return action.action_id.strip()
        return f"act_{uuid.uuid4().hex}"

    def _emit(self, event_type: str, payload: dict[str, Any]) -> None:
        if self.event_sink is not None:
            self.event_sink.append(event_type, payload)
            return
        self.bundle.append(
            event_type,
            payload,
            actor_id=self.actor_id,
            org_id=self.org_id,
            workspace_id=self.workspace_id,
            timestamp=_now_rfc3339(),
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
    ) -> dict[str, Any]:
        payload: dict[str, Any] = {
            "action_id": action_id,
            "action_type": action.action_type,
            "tool_receipt_digest": _value_digest(receipt),
            "execution_duration_ms": max(0, int((time.perf_counter() - started_at) * 1000)),
            "execution_outcome": "success",
        }
        if action.effective_scope and action.effective_scope.strip():
            payload["effective_scope"] = action.effective_scope.strip()
        return payload

    def _error_payload(
        self,
        action: ActionGateInput,
        action_id: str,
        error: Any,
    ) -> dict[str, Any]:
        return {
            "action_id": action_id,
            "action_type": action.action_type,
            "error_class": "exception",
            "error_detail_digest": _value_digest(error),
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


def _now_rfc3339() -> str:
    return datetime.now(UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z")
