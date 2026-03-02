"""
ATB LangChain integration.

Provides an ``ATBCallbackHandler`` that automatically records LangChain
chain/agent/LLM events into an ATB bundle.

Usage::

    from atb import Bundle
    from atb.integrations.langchain import ATBCallbackHandler

    bundle = Bundle()
    handler = ATBCallbackHandler(bundle)

    llm = ChatOpenAI(callbacks=[handler])
    chain = LLMChain(llm=llm, prompt=..., callbacks=[handler])
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

from atb.bundle import Bundle

if TYPE_CHECKING:
    pass


class ATBCallbackHandler:
    """A LangChain callback handler that records events into an ATB bundle.

    .. note::
        Install LangChain before using this handler:
        ``pip install atb-sdk[langchain]``

    Args:
        bundle: The :class:`~atb.Bundle` to append events to.
        auto_save: If ``True``, save the bundle after every event.
        save_path: Path to save the bundle. Defaults to ``run.atb/bundle.atb``.
    """

    def __init__(
        self,
        bundle: Bundle | None = None,
        auto_save: bool = False,
        save_path: str | None = None,
    ) -> None:
        self.bundle = bundle or Bundle()
        self.auto_save = auto_save
        self.save_path = save_path

    def on_llm_start(self, serialized: dict[str, Any], prompts: list[str], **kwargs: Any) -> None:
        self.bundle.append("langchain.llm.start", {
            "model": serialized.get("name", "unknown"),
            "prompt_count": len(prompts),
        })
        self._maybe_save()

    def on_llm_end(self, response: Any, **kwargs: Any) -> None:
        self.bundle.append("langchain.llm.end", {
            "generations": len(getattr(response, "generations", [])),
        })
        self._maybe_save()

    def on_chain_start(self, serialized: dict[str, Any], inputs: dict[str, Any], **kwargs: Any) -> None:
        self.bundle.append("langchain.chain.start", {
            "chain": serialized.get("name", "unknown"),
            "input_keys": list(inputs.keys()),
        })
        self._maybe_save()

    def on_chain_end(self, outputs: dict[str, Any], **kwargs: Any) -> None:
        self.bundle.append("langchain.chain.end", {
            "output_keys": list(outputs.keys()),
        })
        self._maybe_save()

    def on_tool_start(self, serialized: dict[str, Any], input_str: str, **kwargs: Any) -> None:
        self.bundle.append("langchain.tool.start", {
            "tool": serialized.get("name", "unknown"),
            "input": input_str[:500],  # Truncate for safety.
        })
        self._maybe_save()

    def on_tool_end(self, output: str, **kwargs: Any) -> None:
        self.bundle.append("langchain.tool.end", {
            "output": output[:500],
        })
        self._maybe_save()

    def on_agent_action(self, action: Any, **kwargs: Any) -> None:
        self.bundle.append("langchain.agent.action", {
            "tool": getattr(action, "tool", "unknown"),
            "tool_input": str(getattr(action, "tool_input", ""))[:500],
        })
        self._maybe_save()

    def on_agent_finish(self, finish: Any, **kwargs: Any) -> None:
        self.bundle.append("langchain.agent.finish", {
            "return_values": {k: str(v)[:200] for k, v in getattr(finish, "return_values", {}).items()},
        })
        self._maybe_save()

    def _maybe_save(self) -> None:
        if self.auto_save:
            self.bundle.save(self.save_path)
