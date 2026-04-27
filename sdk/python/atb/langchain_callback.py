"""LangChain callback middleware for ATB AI trace events.

Quick start::

    from atb.langchain_callback import ATBCallbackHandler
    handler = ATBCallbackHandler()
    handler.on_llm_start({"name": "model"}, ["hello"])
"""

from __future__ import annotations

import hashlib
import json
import uuid
from dataclasses import dataclass, field
from datetime import UTC, datetime
from typing import Any, Literal

from atb.bundle import Bundle

try:
    from langchain_core.callbacks.base import BaseCallbackHandler
except Exception:  # pragma: no cover - fallback when optional dependency is absent.
    class BaseCallbackHandler:  # type: ignore[no-redef]
        """Fallback base class when langchain-core is not installed."""


PrivacyMode = Literal["off", "hash", "redact"]


@dataclass
class _SpanState:
    trace_id: str
    span_id: str
    parent_span_id: str | None
    event_type: str
    started_at: datetime
    run_id: str
    chunks: list[str] = field(default_factory=list)
    context: dict[str, Any] = field(default_factory=dict)


class ATBCallbackHandler(BaseCallbackHandler):
    """LangChain callback handler that emits AI-spec ATB events.

    This handler is opt-in and no-op when ``enabled=False``.

    Args:
        bundle: Optional bundle to append callback events to.
        enabled: Disable all event emission when false.
        auto_save: Save the bundle after each emitted event when true.
        save_path: Optional save path used when ``auto_save`` is true.
        privacy_mode: ``"off"``, ``"hash"``, or ``"redact"`` text handling.
        actor_id: Optional actor identity metadata.
        org_id: Optional organisation identity metadata.
        workspace_id: Optional workspace identity metadata.
        framework: Framework label recorded in emitted payloads.
        framework_version: Framework version recorded in emitted payloads.

    Returns:
        An ``ATBCallbackHandler`` instance.

    Raises:
        ValueError: If ``privacy_mode`` is not recognised.
    """

    def __init__(
        self,
        bundle: Bundle | None = None,
        *,
        enabled: bool = True,
        auto_save: bool = False,
        save_path: str | None = None,
        privacy_mode: PrivacyMode = "off",
        actor_id: str | None = None,
        org_id: str | None = None,
        workspace_id: str | None = None,
        framework: str = "langchain",
        framework_version: str = "unknown",
    ) -> None:
        if privacy_mode not in {"off", "hash", "redact"}:
            raise ValueError("privacy_mode must be one of: off, hash, redact")

        self.bundle = bundle if bundle is not None else Bundle()
        self.enabled = enabled
        self.auto_save = auto_save
        self.save_path = save_path
        self.privacy_mode = privacy_mode
        self.actor_id = actor_id
        self.org_id = org_id
        self.workspace_id = workspace_id
        self.framework = framework
        self.framework_version = framework_version
        self._runs: dict[str, _SpanState] = {}

    # ------------------------------------------------------------------
    # LangChain callbacks
    # ------------------------------------------------------------------

    def on_chain_start(
        self,
        serialized: dict[str, Any],
        inputs: dict[str, Any],
        *,
        run_id: Any | None = None,
        parent_run_id: Any | None = None,
        **_: Any,
    ) -> None:
        """Record the start of a LangChain chain run.

        Args:
            serialized: LangChain serialised chain metadata.
            inputs: Chain input mapping.
            run_id: Optional LangChain run identifier.
            parent_run_id: Optional parent run identifier.
            **_: Ignored LangChain callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        if not self.enabled:
            return
        state = self._start_span(
            run_id=run_id,
            parent_run_id=parent_run_id,
            event_type="ai.chain.run",
        )
        context = {
            "chain_name": serialized.get("name") or serialized.get("id") or "unknown",
            "input_keys": sorted(list(inputs.keys())),
        }
        state.context = context
        self._emit(
            "ai.chain.run",
            phase="start",
            state=state,
            context=context,
            status_ok=True,
            status_error=None,
            ended_at=None,
        )

    def on_chain_end(
        self,
        outputs: dict[str, Any],
        *,
        run_id: Any | None = None,
        **_: Any,
    ) -> None:
        """Record the end of a LangChain chain run.

        Args:
            outputs: Chain output mapping.
            run_id: Optional LangChain run identifier.
            **_: Ignored LangChain callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        if not self.enabled:
            return
        state = self._get_or_create_span(run_id=run_id, event_type="ai.chain.run")
        context = {
            **state.context,
            "output_keys": sorted(list(outputs.keys())),
        }
        self._emit(
            "ai.chain.run",
            phase="end",
            state=state,
            context=context,
            status_ok=True,
            status_error=None,
            ended_at=self._now(),
        )
        self._finish_span(state.run_id)

    def on_chain_error(
        self,
        error: BaseException,
        *,
        run_id: Any | None = None,
        **_: Any,
    ) -> None:
        """Record a LangChain chain error.

        Args:
            error: Exception or error raised by the chain.
            run_id: Optional LangChain run identifier.
            **_: Ignored LangChain callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        if not self.enabled:
            return
        state = self._get_or_create_span(run_id=run_id, event_type="ai.chain.run")
        self._emit(
            "ai.chain.run",
            phase="error",
            state=state,
            context=state.context,
            status_ok=False,
            status_error={"type": type(error).__name__, "message": str(error)},
            ended_at=self._now(),
        )
        self._finish_span(state.run_id)

    def on_llm_start(
        self,
        serialized: dict[str, Any],
        prompts: list[str],
        *,
        run_id: Any | None = None,
        parent_run_id: Any | None = None,
        **kwargs: Any,
    ) -> None:
        """Record the start of an LLM call.

        Args:
            serialized: LangChain serialised LLM metadata.
            prompts: Prompt strings supplied to the LLM.
            run_id: Optional LangChain run identifier.
            parent_run_id: Optional parent run identifier.
            **kwargs: Additional LangChain invocation metadata.

        Returns:
            None.

        Raises:
            None.
        """
        if not self.enabled:
            return
        state = self._start_span(
            run_id=run_id,
            parent_run_id=parent_run_id,
            event_type="ai.llm.call",
        )
        provider, model = self._provider_and_model(serialized, kwargs)
        prompt_text = "\n".join(prompts)
        context = {
            "provider": provider,
            "model": model,
            "prompt": self._privacy_text(prompt_text),
        }
        state.context = context
        self._emit(
            "ai.llm.call",
            phase="start",
            state=state,
            context=context,
            status_ok=True,
            status_error=None,
            ended_at=None,
        )

    def on_llm_new_token(
        self,
        token: str,
        *,
        run_id: Any | None = None,
        **_: Any,
    ) -> None:
        """Record a streamed LLM token.

        Args:
            token: Generated token text.
            run_id: Optional LangChain run identifier.
            **_: Ignored LangChain callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        if not self.enabled:
            return
        state = self._get_or_create_span(run_id=run_id, event_type="ai.llm.call")
        state.chunks.append(token)
        chunk_idx = len(state.chunks) - 1
        context = {
            **state.context,
            "stream": {
                "chunk_index": chunk_idx,
                "chunk": self._privacy_text(token),
            },
        }
        self._emit(
            "ai.llm.call",
            phase="delta",
            state=state,
            context=context,
            status_ok=True,
            status_error=None,
            ended_at=None,
        )

    def on_llm_end(
        self,
        response: Any,
        *,
        run_id: Any | None = None,
        **_: Any,
    ) -> None:
        """Record the end of an LLM call.

        Args:
            response: LangChain LLM response object.
            run_id: Optional LangChain run identifier.
            **_: Ignored LangChain callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        if not self.enabled:
            return
        state = self._get_or_create_span(run_id=run_id, event_type="ai.llm.call")
        completion_text = "".join(state.chunks)
        if completion_text == "":
            completion_text = self._extract_response_text(response)

        context = {
            **state.context,
            "completion": self._privacy_text(completion_text),
            "token_usage": self._extract_token_usage(response),
            "finish_reason": self._extract_finish_reason(response),
        }
        self._emit(
            "ai.llm.call",
            phase="end",
            state=state,
            context=context,
            status_ok=True,
            status_error=None,
            ended_at=self._now(),
        )
        self._finish_span(state.run_id)

    def on_llm_error(
        self,
        error: BaseException,
        *,
        run_id: Any | None = None,
        **_: Any,
    ) -> None:
        """Record an LLM error.

        Args:
            error: Exception or error raised by the LLM.
            run_id: Optional LangChain run identifier.
            **_: Ignored LangChain callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        if not self.enabled:
            return
        state = self._get_or_create_span(run_id=run_id, event_type="ai.llm.call")
        self._emit(
            "ai.llm.call",
            phase="error",
            state=state,
            context=state.context,
            status_ok=False,
            status_error={"type": type(error).__name__, "message": str(error)},
            ended_at=self._now(),
        )
        self._finish_span(state.run_id)

    def on_tool_start(
        self,
        serialized: dict[str, Any],
        input_str: str,
        *,
        run_id: Any | None = None,
        parent_run_id: Any | None = None,
        **_: Any,
    ) -> None:
        """Record the start of a tool call.

        Args:
            serialized: LangChain serialised tool metadata.
            input_str: Tool input text.
            run_id: Optional LangChain run identifier.
            parent_run_id: Optional parent run identifier.
            **_: Ignored LangChain callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        if not self.enabled:
            return
        state = self._start_span(
            run_id=run_id,
            parent_run_id=parent_run_id,
            event_type="ai.tool.exec",
        )
        context = {
            "tool_name": serialized.get("name") or serialized.get("id") or "unknown",
            "input": self._privacy_text(input_str),
        }
        state.context = context
        self._emit(
            "ai.tool.exec",
            phase="start",
            state=state,
            context=context,
            status_ok=True,
            status_error=None,
            ended_at=None,
        )

    def on_tool_end(
        self,
        output: Any,
        *,
        run_id: Any | None = None,
        **_: Any,
    ) -> None:
        """Record the end of a tool call.

        Args:
            output: Tool output value.
            run_id: Optional LangChain run identifier.
            **_: Ignored LangChain callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        if not self.enabled:
            return
        state = self._get_or_create_span(run_id=run_id, event_type="ai.tool.exec")
        output_text = self._safe_json(output)
        context = {
            **state.context,
            "output": self._privacy_text(output_text),
        }
        self._emit(
            "ai.tool.exec",
            phase="end",
            state=state,
            context=context,
            status_ok=True,
            status_error=None,
            ended_at=self._now(),
        )
        self._finish_span(state.run_id)

    def on_tool_error(
        self,
        error: BaseException,
        *,
        run_id: Any | None = None,
        **_: Any,
    ) -> None:
        """Record a tool error.

        Args:
            error: Exception or error raised by the tool.
            run_id: Optional LangChain run identifier.
            **_: Ignored LangChain callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        if not self.enabled:
            return
        state = self._get_or_create_span(run_id=run_id, event_type="ai.tool.exec")
        self._emit(
            "ai.tool.exec",
            phase="error",
            state=state,
            context=state.context,
            status_ok=False,
            status_error={"type": type(error).__name__, "message": str(error)},
            ended_at=self._now(),
        )
        self._finish_span(state.run_id)

    # ------------------------------------------------------------------
    # Async wrappers
    # ------------------------------------------------------------------

    async def aon_chain_start(self, serialized: dict[str, Any], inputs: dict[str, Any], **kwargs: Any) -> None:
        """Async wrapper for ``on_chain_start``.

        Args:
            serialized: LangChain serialised chain metadata.
            inputs: Chain input mapping.
            **kwargs: Additional callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        self.on_chain_start(serialized, inputs, **kwargs)

    async def aon_chain_end(self, outputs: dict[str, Any], **kwargs: Any) -> None:
        """Async wrapper for ``on_chain_end``.

        Args:
            outputs: Chain output mapping.
            **kwargs: Additional callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        self.on_chain_end(outputs, **kwargs)

    async def aon_chain_error(self, error: BaseException, **kwargs: Any) -> None:
        """Async wrapper for ``on_chain_error``.

        Args:
            error: Exception or error raised by the chain.
            **kwargs: Additional callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        self.on_chain_error(error, **kwargs)

    async def aon_llm_start(self, serialized: dict[str, Any], prompts: list[str], **kwargs: Any) -> None:
        """Async wrapper for ``on_llm_start``.

        Args:
            serialized: LangChain serialised LLM metadata.
            prompts: Prompt strings supplied to the LLM.
            **kwargs: Additional callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        self.on_llm_start(serialized, prompts, **kwargs)

    async def aon_llm_new_token(self, token: str, **kwargs: Any) -> None:
        """Async wrapper for ``on_llm_new_token``.

        Args:
            token: Generated token text.
            **kwargs: Additional callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        self.on_llm_new_token(token, **kwargs)

    async def aon_llm_end(self, response: Any, **kwargs: Any) -> None:
        """Async wrapper for ``on_llm_end``.

        Args:
            response: LangChain LLM response object.
            **kwargs: Additional callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        self.on_llm_end(response, **kwargs)

    async def aon_llm_error(self, error: BaseException, **kwargs: Any) -> None:
        """Async wrapper for ``on_llm_error``.

        Args:
            error: Exception or error raised by the LLM.
            **kwargs: Additional callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        self.on_llm_error(error, **kwargs)

    async def aon_tool_start(self, serialized: dict[str, Any], input_str: str, **kwargs: Any) -> None:
        """Async wrapper for ``on_tool_start``.

        Args:
            serialized: LangChain serialised tool metadata.
            input_str: Tool input text.
            **kwargs: Additional callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        self.on_tool_start(serialized, input_str, **kwargs)

    async def aon_tool_end(self, output: Any, **kwargs: Any) -> None:
        """Async wrapper for ``on_tool_end``.

        Args:
            output: Tool output value.
            **kwargs: Additional callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        self.on_tool_end(output, **kwargs)

    async def aon_tool_error(self, error: BaseException, **kwargs: Any) -> None:
        """Async wrapper for ``on_tool_error``.

        Args:
            error: Exception or error raised by the tool.
            **kwargs: Additional callback metadata.

        Returns:
            None.

        Raises:
            None.
        """
        self.on_tool_error(error, **kwargs)

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _emit(
        self,
        event_type: str,
        *,
        phase: str,
        state: _SpanState,
        context: dict[str, Any],
        status_ok: bool,
        status_error: dict[str, Any] | None,
        ended_at: datetime | None,
    ) -> None:
        now = self._now()
        latency = None
        ended_at_str: str | None = None
        if ended_at is not None:
            latency = int((ended_at - state.started_at).total_seconds() * 1000)
            ended_at_str = self._iso(ended_at)

        payload = {
            "trace_id": state.trace_id,
            "span_id": state.span_id,
            "parent_span_id": state.parent_span_id,
            "framework": self.framework,
            "framework_version": self.framework_version,
            "phase": phase,
            "run_id": state.run_id,
            "context": context,
            "privacy": {
                "mode": self.privacy_mode,
                "redaction_enabled": self.privacy_mode != "off",
                "pii_ruleset": "phase4-gdpr-v1",
            },
            "timing": {
                "started_at": self._iso(state.started_at),
                "ended_at": ended_at_str,
                "latency_ms": latency,
            },
            "status": {
                "ok": status_ok,
                "error": status_error,
            },
            "emitted_at": self._iso(now),
        }

        self.bundle.append(
            event_type,
            payload,
            actor_id=self.actor_id,
            org_id=self.org_id,
            workspace_id=self.workspace_id,
            timestamp=self._iso(now),
            trace_id=state.trace_id,
            span_id=state.span_id,
            parent_span_id=state.parent_span_id,
        )
        if self.auto_save:
            self.bundle.save(self.save_path)

    def _start_span(self, *, run_id: Any | None, parent_run_id: Any | None, event_type: str) -> _SpanState:
        run_key = self._run_key(run_id)
        parent_key = self._run_key(parent_run_id) if parent_run_id is not None else None
        parent_state = self._runs.get(parent_key) if parent_key else None

        trace_id = parent_state.trace_id if parent_state else self._new_id("trc")
        parent_span_id = parent_state.span_id if parent_state else None
        state = _SpanState(
            trace_id=trace_id,
            span_id=self._new_id("spn"),
            parent_span_id=parent_span_id,
            event_type=event_type,
            started_at=self._now(),
            run_id=run_key,
        )
        self._runs[run_key] = state
        return state

    def _get_or_create_span(self, *, run_id: Any | None, event_type: str) -> _SpanState:
        run_key = self._run_key(run_id)
        existing = self._runs.get(run_key)
        if existing is not None:
            return existing
        return self._start_span(run_id=run_key, parent_run_id=None, event_type=event_type)

    def _finish_span(self, run_key: str) -> None:
        self._runs.pop(run_key, None)

    def _privacy_text(self, text: str) -> dict[str, str]:
        original = text
        if self.privacy_mode == "off":
            emitted = original
        elif self.privacy_mode == "hash":
            emitted = "sha256:" + hashlib.sha256(original.encode("utf-8")).hexdigest()
        else:
            emitted = "[REDACTED]" if original != "" else ""

        digest = hashlib.sha256(emitted.encode("utf-8")).hexdigest()
        return {
            "text": emitted,
            "sha256": "sha256:" + digest,
        }

    def _provider_and_model(self, serialized: dict[str, Any], kwargs: dict[str, Any]) -> tuple[str, str]:
        provider = "unknown"
        model = "unknown"

        name = serialized.get("name")
        if isinstance(name, str) and name:
            model = name

        identifier = serialized.get("id")
        if isinstance(identifier, list) and identifier:
            first = identifier[0]
            if isinstance(first, str) and first:
                provider = first

        invocation_params = kwargs.get("invocation_params")
        if isinstance(invocation_params, dict):
            raw_model = invocation_params.get("model")
            if isinstance(raw_model, str) and raw_model:
                model = raw_model
            raw_provider = invocation_params.get("provider")
            if isinstance(raw_provider, str) and raw_provider:
                provider = raw_provider

        return provider, model

    def _extract_response_text(self, response: Any) -> str:
        generations = getattr(response, "generations", None)
        if not isinstance(generations, list):
            return ""

        parts: list[str] = []
        for item in generations:
            if isinstance(item, list):
                for generation in item:
                    text = getattr(generation, "text", None)
                    if isinstance(text, str):
                        parts.append(text)
            else:
                text = getattr(item, "text", None)
                if isinstance(text, str):
                    parts.append(text)
        return "".join(parts)

    def _extract_token_usage(self, response: Any) -> dict[str, int]:
        llm_output = getattr(response, "llm_output", None)
        if not isinstance(llm_output, dict):
            return {
                "prompt_tokens": 0,
                "completion_tokens": 0,
                "total_tokens": 0,
            }

        usage = llm_output.get("token_usage")
        if not isinstance(usage, dict):
            return {
                "prompt_tokens": 0,
                "completion_tokens": 0,
                "total_tokens": 0,
            }

        prompt_tokens = int(usage.get("prompt_tokens") or 0)
        completion_tokens = int(usage.get("completion_tokens") or 0)
        total_tokens = int(usage.get("total_tokens") or (prompt_tokens + completion_tokens))
        return {
            "prompt_tokens": prompt_tokens,
            "completion_tokens": completion_tokens,
            "total_tokens": total_tokens,
        }

    def _extract_finish_reason(self, response: Any) -> str | None:
        llm_output = getattr(response, "llm_output", None)
        if isinstance(llm_output, dict):
            reason = llm_output.get("finish_reason")
            if isinstance(reason, str):
                return reason

        generations = getattr(response, "generations", None)
        if isinstance(generations, list) and generations:
            first = generations[0]
            if isinstance(first, list) and first:
                info = getattr(first[0], "generation_info", None)
                if isinstance(info, dict):
                    reason = info.get("finish_reason")
                    if isinstance(reason, str):
                        return reason
        return None

    def _safe_json(self, value: Any) -> str:
        if isinstance(value, str):
            return value
        try:
            return json.dumps(value, sort_keys=True, ensure_ascii=True)
        except TypeError:
            return str(value)

    def _run_key(self, run_id: Any | None) -> str:
        if run_id is None:
            return self._new_id("run")
        return str(run_id)

    def _new_id(self, prefix: str) -> str:
        return f"{prefix}_{uuid.uuid4().hex}"

    def _now(self) -> datetime:
        return datetime.now(UTC)

    def _iso(self, ts: datetime) -> str:
        return ts.astimezone(UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z")
