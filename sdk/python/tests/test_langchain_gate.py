from __future__ import annotations

import pytest

from atb import Bundle
from atb.action_gate import ActionGate, ActionGateDecision, ActionGateDeniedError
from atb.langchain_gate import gate_langchain_tool


class StubTool:
    name = "deploy_change"
    description = "roll out release"
    args_schema = {"type": "object"}

    def _run(self, payload: dict[str, str]) -> dict[str, str]:
        return {"result": payload["version"]}

    async def _arun(self, payload: dict[str, str]) -> dict[str, str]:
        return {"result": payload["version"]}


def test_gate_langchain_tool_records_events() -> None:
    bundle = Bundle()
    gate = ActionGate(bundle=bundle, actor_id="actor-1")

    wrapped = gate_langchain_tool(StubTool(), gate)
    result = wrapped._run({"version": "1.5.0"})

    assert result == {"result": "1.5.0"}
    assert [record.event["type"] for record in bundle.records] == [
        "ai.action.precommit",
        "ai.policy.decision",
        "ai.action.executed",
    ]


def test_gate_langchain_tool_enforce_blocks() -> None:
    bundle = Bundle()
    gate = ActionGate(
        bundle=bundle,
        mode="enforce",
        actor_id="actor-1",
        policy=lambda _: ActionGateDecision(decision="deny", reason_codes=("blocked",)),
    )

    wrapped = gate_langchain_tool(StubTool(), gate)

    with pytest.raises(ActionGateDeniedError):
        wrapped._run({"version": "1.5.0"})

    assert [record.event["type"] for record in bundle.records] == [
        "ai.action.precommit",
        "ai.policy.decision",
    ]
