from __future__ import annotations

from typing import Any

from atb.action_gate import ActionGate, ActionGateInput

try:
    from langchain_core.tools import BaseTool
except Exception:  # pragma: no cover - fallback when optional dependency is absent.
    class BaseTool:  # type: ignore[no-redef]
        pass


class _GatedLangChainTool:
    def __init__(self, tool: Any, gate: ActionGate) -> None:
        self._tool = tool
        self._gate = gate
        self.name = getattr(tool, "name", "")
        self.description = getattr(tool, "description", "")
        self.args_schema = getattr(tool, "args_schema", None)

    def __getattr__(self, name: str) -> Any:
        return getattr(self._tool, name)

    def _run(self, *args: Any, **kwargs: Any) -> Any:
        action = _action_input(self._tool, args, kwargs)
        return self._gate.run(action, lambda: self._tool._run(*args, **kwargs))

    async def _arun(self, *args: Any, **kwargs: Any) -> Any:
        action = _action_input(self._tool, args, kwargs)
        return await self._gate.arun(action, lambda: self._tool._arun(*args, **kwargs))


def gate_langchain_tool(tool: BaseTool, gate: ActionGate) -> BaseTool:
    return _GatedLangChainTool(tool, gate)  # type: ignore[return-value]


def _action_input(tool: Any, args: tuple[Any, ...], kwargs: dict[str, Any]) -> ActionGateInput:
    if kwargs:
        action_parameters: dict[str, Any] = dict(kwargs)
    elif len(args) == 1 and isinstance(args[0], dict):
        action_parameters = dict(args[0])
    else:
        action_parameters = {"args": list(args)}

    tool_name = getattr(tool, "name", "")
    description = getattr(tool, "description", "") or ""
    return ActionGateInput(
        action_type=tool_name,
        target_resource_id=tool_name,
        intended_effect=description,
        action_parameters=action_parameters,
        subject_id=None,
    )
