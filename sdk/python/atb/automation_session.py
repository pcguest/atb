"""AutomationSession — workflow chaining harness for multi-hop AI capture.

Composes :class:`~atb.workflow_common.WorkflowContext` and
:class:`~atb.action_gate.ActionGate`,
and :class:`~atb.policy_decision_recorder.PolicyDecisionRecorder` so callers open one
bundle connection, emit events across successive model/tool hops, and flush without
per-call boilerplate.

When the local ATB Agent is configured (``ATB_AGENT_URL`` or ``ATB_AGENT_AUTO``),
events route through the Agent HTTP API; otherwise events append to the bundle file.
"""

from __future__ import annotations

import os
from collections.abc import Awaitable, Callable, Mapping
from os import PathLike
from pathlib import Path
from typing import Any, TypeVar

from atb.action_gate import ActionGate, ActionGateInput
from atb.agent_client import AgentClient, try_create_agent_client
from atb.bundle import Bundle
from atb.event_types import (
    AI_ACTION_COMMITTED_EVENT_TYPE,
    AI_MODEL_INVOKED_EVENT_TYPE,
    AI_MODEL_OUTPUT_EVENT_TYPE,
    AI_RESPONSE_SENT_EVENT_TYPE,
    AI_RETRIEVAL_EXECUTED_EVENT_TYPE,
    SNAPSHOT_EVENT_TYPE,
)
from atb.policy_decision_recorder import (
    PolicyDecisionActionInput,
    PolicyDecisionRecorder,
)
from atb.workflow_common import (
    WorkflowContext,
    canonical_digest,
    new_action_id,
    value_digest,
)

T = TypeVar("T")
_AGENT_CLIENT_UNSET: Any = object()


def is_capture_environment(env: Mapping[str, str | None] | None = None) -> bool:
    """Return True when live capture environment variables are present."""
    lookup = os.environ if env is None else env
    return bool((lookup.get("ATB_BUNDLE_PATH") or "").strip())


class AutomationSession:
    """Session-scoped harness for chained ATB event emission across AI workflow hops."""

    def __init__(
        self,
        *,
        bundle_path: str | PathLike[str] | None = None,
        bundle: Bundle | None = None,
        actor_id: str | None = None,
        purpose_tag: str | None = None,
        request_id: str | None = None,
        capture_run_id: str | None = None,
        auto_save: bool = False,
        save_path: str | PathLike[str] | None = None,
        org_id: str | None = None,
        workspace_id: str | None = None,
        env: Mapping[str, str | None] | None = None,
        agent_client: AgentClient | None | Any = _AGENT_CLIENT_UNSET,
    ) -> None:
        lookup = os.environ if env is None else env
        resolved_path = _resolve_bundle_path(bundle_path, save_path, lookup)
        if resolved_path is None:
            raise ValueError(
                "AutomationSession requires bundle_path or ATB_BUNDLE_PATH "
                "environment variable"
            )

        self.save_path = str(resolved_path)
        self.capture_run_id = _optional_str(capture_run_id) or _optional_str(
            lookup.get("ATB_CAPTURE_RUN_ID")
        )
        self.default_purpose_tag = (
            _optional_str(purpose_tag)
            or _optional_str(lookup.get("ATB_CAPTURE_MODE"))
            or "ai_workflow"
        )
        self.active_request_id = _optional_str(request_id)

        resolved_agent = _resolve_agent_client(agent_client, lookup)
        self._agent_client = resolved_agent
        self._use_agent = resolved_agent is not None
        event_sink = resolved_agent

        if bundle is None and not self._use_agent and Path(self.save_path).exists():
            bundle = Bundle.load(self.save_path)

        if self._use_agent and resolved_agent is not None:
            opened = resolved_agent.open_session(
                actor_id=actor_id,
                purpose_tag=self.default_purpose_tag,
                bundle_path=self.save_path,
            )
            if opened.bundle_path:
                self.save_path = opened.bundle_path

        self.ctx = WorkflowContext(
            bundle,
            auto_save=auto_save,
            save_path=self.save_path,
            actor_id=actor_id,
            org_id=org_id,
            workspace_id=workspace_id,
            event_sink=event_sink,
        )
        self.tool_gate = ActionGate(
            bundle=self.ctx.bundle,
            auto_save=auto_save,
            save_path=self.save_path,
            actor_id=actor_id,
            org_id=org_id,
            workspace_id=workspace_id,
            event_sink=event_sink,
        )
        self.policy_recorder = PolicyDecisionRecorder(
            bundle=self.ctx.bundle,
            auto_save=auto_save,
            save_path=self.save_path,
            actor_id=actor_id,
            org_id=org_id,
            workspace_id=workspace_id,
            event_sink=event_sink,
        )

    @classmethod
    def open(
        cls,
        *,
        bundle_path: str | PathLike[str] | None = None,
        bundle: Bundle | None = None,
        actor_id: str | None = None,
        purpose_tag: str | None = None,
        request_id: str | None = None,
        capture_run_id: str | None = None,
        save_path: str | PathLike[str] | None = None,
        org_id: str | None = None,
        workspace_id: str | None = None,
        env: Mapping[str, str | None] | None = None,
        agent_client: AgentClient | None | Any = _AGENT_CLIENT_UNSET,
    ) -> AutomationSession:
        """Open a session with auto-save enabled (typical live capture default)."""
        return cls(
            bundle_path=bundle_path,
            bundle=bundle,
            actor_id=actor_id,
            purpose_tag=purpose_tag,
            request_id=request_id,
            capture_run_id=capture_run_id,
            auto_save=True,
            save_path=save_path,
            org_id=org_id,
            workspace_id=workspace_id,
            env=env,
            agent_client=agent_client,
        )

    @classmethod
    def from_capture_environment(
        cls,
        env: Mapping[str, str | None] | None = None,
    ) -> AutomationSession | None:
        """Create or resume a session from ``atb capture run`` environment variables.

        Returns ``None`` when capture env vars are absent.
        """
        lookup = os.environ if env is None else env
        if not is_capture_environment(lookup):
            return None
        save_path = str(lookup.get("ATB_BUNDLE_PATH", "")).strip()
        bundle = None
        if not _resolve_agent_client(_AGENT_CLIENT_UNSET, lookup):
            bundle = Bundle.load(save_path) if Path(save_path).exists() else None
        return cls.open(
            bundle=bundle,
            save_path=save_path,
            capture_run_id=lookup.get("ATB_CAPTURE_RUN_ID"),
            purpose_tag=lookup.get("ATB_CAPTURE_MODE") or "capture_run",
            env=lookup,
        )

    def is_using_agent(self) -> bool:
        """True when this session emits through the local ATB Agent HTTP API."""
        return self._use_agent

    @property
    def bundle(self) -> Bundle:
        return self.ctx.bundle

    def begin_request(
        self,
        purpose_tag: str | None = None,
        request_id: str | None = None,
    ) -> str:
        """Begin or continue the active request chain."""
        tag = _optional_str(purpose_tag) or self.default_purpose_tag
        rid = self.ctx.bootstrap_request(tag, request_id or self.active_request_id)
        self.active_request_id = rid
        return rid

    def log_model_invocation(
        self,
        provider: str,
        model: str,
        prompt: Any,
        *,
        parameters: Mapping[str, Any] | None = None,
        request_id: str | None = None,
    ) -> None:
        """Emit ``ai.model.invoked`` for the active request."""
        rid = self.begin_request(self.default_purpose_tag, request_id)
        self.ctx.emit(
            AI_MODEL_INVOKED_EVENT_TYPE,
            {
                "request_id": rid,
                "model_provider": provider,
                "model_id": model,
                "model_parameters_digest": canonical_digest(parameters or {}),
                "prompt_digest": canonical_digest(prompt),
            },
        )

    def log_model_output(
        self,
        output: Any,
        *,
        output_format: str = "text/plain",
        request_id: str | None = None,
    ) -> None:
        """Emit ``ai.model.output`` for the active request."""
        rid = self._request_id(request_id)
        payload: dict[str, Any] = {
            "output_digest": value_digest(output),
            "output_format": output_format,
        }
        if rid is not None:
            payload["request_id"] = rid
        self.ctx.emit(AI_MODEL_OUTPUT_EVENT_TYPE, payload)

    def log_retrieval(
        self,
        query: Any,
        corpus_id: str,
        corpus_version: str,
        top_k: int,
        result_set: Any,
        *,
        request_id: str | None = None,
    ) -> None:
        """Emit ``ai.retrieval.executed`` for the active request."""
        rid = self.begin_request(self.default_purpose_tag, request_id)
        self.ctx.emit(
            AI_RETRIEVAL_EXECUTED_EVENT_TYPE,
            {
                "request_id": rid,
                "retrieval_query_hash": canonical_digest(query),
                "retrieval_corpus_id": corpus_id,
                "retrieval_corpus_version": corpus_version,
                "top_k": top_k,
                "result_set_digest": canonical_digest(result_set),
            },
        )

    def log_response_sent(
        self,
        output: Any,
        *,
        output_format: str = "text/plain",
        request_id: str | None = None,
    ) -> None:
        """Emit ``ai.response.sent`` to close the application response boundary."""
        rid = self.begin_request(self.default_purpose_tag, request_id)
        self.ctx.emit(
            AI_RESPONSE_SENT_EVENT_TYPE,
            {
                "request_id": rid,
                "output_digest": value_digest(output),
                "output_format": output_format,
            },
        )

    def run_tool_action(self, action: ActionGateInput, fn: Callable[[], T]) -> T:
        """Run a tool action with request bootstrap, gate events, and commit."""
        self.begin_request("privileged_tool_action")
        action_id = new_action_id(action.action_id)
        result = self.tool_gate.run(
            ActionGateInput(
                action_type=action.action_type,
                target_resource_id=action.target_resource_id,
                intended_effect=action.intended_effect,
                action_parameters=action.action_parameters,
                subject_id=action.subject_id,
                action_id=action_id,
                policy_context=action.policy_context,
            ),
            fn,
        )
        self.ctx.emit(
            AI_ACTION_COMMITTED_EVENT_TYPE,
            {
                "action_id": action_id,
                "commit_outcome": "success",
                "sink_receipt_digest": value_digest(result),
            },
        )
        return result

    async def arun_tool_action(
        self,
        action: ActionGateInput,
        fn: Callable[[], Awaitable[T]],
    ) -> T:
        """Async variant of :meth:`run_tool_action`."""
        self.begin_request("privileged_tool_action")
        action_id = new_action_id(action.action_id)
        result = await self.tool_gate.arun(
            ActionGateInput(
                action_type=action.action_type,
                target_resource_id=action.target_resource_id,
                intended_effect=action.intended_effect,
                action_parameters=action.action_parameters,
                subject_id=action.subject_id,
                action_id=action_id,
                policy_context=action.policy_context,
            ),
            fn,
        )
        self.ctx.emit(
            AI_ACTION_COMMITTED_EVENT_TYPE,
            {
                "action_id": action_id,
                "commit_outcome": "success",
                "sink_receipt_digest": value_digest(result),
            },
        )
        return result

    def log_policy_decision(
        self,
        action: PolicyDecisionActionInput,
        decision: Mapping[str, Any],
    ) -> str:
        """Record a policy decision without running an action."""
        return self.policy_recorder.record(action, decision)

    def save(self, save_path: str | PathLike[str] | None = None) -> Path:
        """Persist the bundle to disk."""
        if self._use_agent:
            return Path(self.save_path)
        target = Path(save_path) if save_path is not None else Path(self.save_path)
        return self.ctx.bundle.save(target)

    def snapshot(self, name: str) -> None:
        """Append a named snapshot boundary event."""
        trimmed = name.strip()
        if trimmed == "":
            raise ValueError("snapshot name must be non-empty")
        self.ctx.emit(SNAPSHOT_EVENT_TYPE, {"name": trimmed})

    def close(
        self,
        *,
        snapshot_name: str | None = None,
        save_path: str | PathLike[str] | None = None,
    ) -> None:
        """Flush the bundle and optionally append a snapshot."""
        if snapshot_name is not None:
            self.snapshot(snapshot_name)
        if self._use_agent and self._agent_client is not None:
            self._agent_client.close_session(snapshot_name=snapshot_name)
            return
        self.save(save_path)

    def _request_id(self, explicit: str | None = None) -> str | None:
        candidate = _optional_str(explicit) or self.active_request_id
        return candidate or None


def _resolve_agent_client(
    agent_client: AgentClient | None | Any,
    env: Mapping[str, str | None],
) -> AgentClient | None:
    if agent_client is _AGENT_CLIENT_UNSET:
        return try_create_agent_client(env)
    return agent_client


def _resolve_bundle_path(
    bundle_path: str | PathLike[str] | None,
    save_path: str | PathLike[str] | None,
    env: Mapping[str, str | None],
) -> str | None:
    for candidate in (bundle_path, save_path, env.get("ATB_BUNDLE_PATH")):
        resolved = _optional_str(candidate)
        if resolved is not None:
            return resolved
    return None


def _optional_str(value: str | PathLike[str] | None) -> str | None:
    if value is None:
        return None
    trimmed = str(value).strip()
    return trimmed or None


__all__ = ["AutomationSession", "is_capture_environment"]
