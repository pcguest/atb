from __future__ import annotations

import asyncio

import pytest

from atb import Bundle
from atb.action_gate import ActionGate, ActionGateDecision, ActionGateDeniedError, ActionGateInput


def _user_records(bundle: Bundle) -> list:
    return [
        record
        for record in bundle.records
        if record.event["type"] != "atb.bundle.manifest"
    ]


def test_action_gate_run_records_precommit_before_execution() -> None:
    bundle = Bundle()
    gate = ActionGate(bundle=bundle, actor_id="actor-1")
    action = ActionGateInput(
        action_type="deploy_change",
        target_resource_id="svc-prod",
        intended_effect="roll out release",
        action_parameters={"version": "0.9.2-beta"},
    )
    seen_before_execution: list[str] = []

    def fn() -> str:
        seen_before_execution.extend(record.event["type"] for record in _user_records(bundle))
        return "ok"

    gate.run(action, fn)

    assert seen_before_execution == ["ai.action.precommit", "ai.policy.decision"]


def test_action_gate_run_records_policy_and_executed_after_success() -> None:
    bundle = Bundle()
    gate = ActionGate(
        bundle=bundle,
        actor_id="actor-1",
        policy=lambda _: ActionGateDecision(
            decision="allow",
            reason_codes=("policy_pass",),
            policy_id="policy.allow",
            policy_version="2026-04-01",
        ),
    )
    action = ActionGateInput(
        action_type="deploy_change",
        target_resource_id="svc-prod",
        intended_effect="roll out release",
        action_parameters={"version": "0.9.2-beta"},
    )

    result = gate.run(action, lambda: {"status": "done"})

    assert result == {"status": "done"}
    events = [record.event for record in _user_records(bundle)]
    assert [event["type"] for event in events] == [
        "ai.action.precommit",
        "ai.policy.decision",
        "ai.action.executed",
    ]

    precommit = events[0]["data"]
    assert precommit["action_id"].startswith("act_")
    assert precommit["action_type"] == "deploy_change"
    assert precommit["action_parameters_digest"].startswith("sha256:")
    assert precommit["target_resource_id"] == "svc-prod"
    assert precommit["intended_effect"] == "roll out release"

    policy = events[1]["data"]
    assert policy["decision_id"] == precommit["action_id"]
    assert policy["action_id"] == precommit["action_id"]
    assert policy["policy_id"] == "policy.allow"
    assert policy["policy_version"] == "2026-04-01"
    assert policy["decision"] == "allow"
    assert policy["decision_reason_codes"] == ["policy_pass"]
    assert policy["subject_id_hash"].startswith("sha256:")

    executed = events[2]["data"]
    assert executed["action_id"] == precommit["action_id"]
    assert executed["action_type"] == "deploy_change"
    assert executed["tool_receipt_digest"].startswith("sha256:")
    assert executed["execution_duration_ms"] >= 0
    assert executed["execution_outcome"] == "success"


def test_action_gate_run_enforce_mode_blocks_when_policy_denies() -> None:
    bundle = Bundle()
    gate = ActionGate(
        bundle=bundle,
        mode="enforce",
        actor_id="actor-1",
        policy=lambda _: ActionGateDecision(decision="deny", reason_codes=("blocked",)),
    )
    action = ActionGateInput(
        action_type="delete_records",
        target_resource_id="customer-42",
        intended_effect="remove account",
        action_parameters={"record_count": 12},
    )
    called = False

    def fn() -> str:
        nonlocal called
        called = True
        return "should-not-run"

    with pytest.raises(ActionGateDeniedError):
        gate.run(action, fn)

    assert called is False
    events = [record.event for record in _user_records(bundle)]
    assert [event["type"] for event in events] == ["ai.action.precommit", "ai.policy.decision"]
    assert events[1]["data"]["decision"] == "deny"


def test_action_gate_run_log_only_never_blocks_on_deny() -> None:
    bundle = Bundle()
    gate = ActionGate(
        bundle=bundle,
        mode="log_only",
        actor_id="actor-1",
        policy=lambda _: ActionGateDecision(decision="deny", reason_codes=("observe_only",)),
    )
    action = ActionGateInput(
        action_type="deploy_change",
        target_resource_id="svc-prod",
        intended_effect="roll out release",
        action_parameters={"version": "0.9.2-beta"},
    )
    called = False

    def fn() -> str:
        nonlocal called
        called = True
        return "ran"

    result = gate.run(action, fn)

    assert result == "ran"
    assert called is True
    events = [record.event for record in _user_records(bundle)]
    assert [event["type"] for event in events] == [
        "ai.action.precommit",
        "ai.policy.decision",
        "ai.action.executed",
    ]
    assert events[1]["data"]["decision"] == "deny"
    assert events[2]["data"]["execution_outcome"] == "success"


def test_action_gate_arun_matches_sync_semantics() -> None:
    bundle = Bundle()
    gate = ActionGate(bundle=bundle, actor_id="actor-1")
    action = ActionGateInput(
        action_type="deploy_change",
        target_resource_id="svc-prod",
        intended_effect="roll out release",
        action_parameters={"version": "0.9.2-beta"},
    )
    seen_before_execution: list[str] = []

    async def fn() -> dict[str, str]:
        seen_before_execution.extend(record.event["type"] for record in _user_records(bundle))
        return {"status": "async-ok"}

    result = asyncio.run(gate.arun(action, fn))

    assert result == {"status": "async-ok"}
    assert seen_before_execution == ["ai.action.precommit", "ai.policy.decision"]
    events = [record.event for record in _user_records(bundle)]
    assert [event["type"] for event in events] == [
        "ai.action.precommit",
        "ai.policy.decision",
        "ai.action.executed",
    ]


def test_action_gate_run_records_action_error_on_failure() -> None:
    bundle = Bundle()
    gate = ActionGate(bundle=bundle, actor_id="actor-1")
    action = ActionGateInput(
        action_type="delete_records",
        target_resource_id="svc-prod",
        intended_effect="purge accounts",
        action_parameters={"scope": "all"},
    )

    def boom() -> str:
        raise RuntimeError("sink refused")

    with pytest.raises(RuntimeError, match="sink refused"):
        gate.run(action, boom)

    events = [record.event for record in _user_records(bundle)]
    assert [event["type"] for event in events] == [
        "ai.action.precommit",
        "ai.policy.decision",
        "ai.action.error",
    ]
    precommit = events[0]["data"]
    error = events[2]["data"]
    assert error["action_id"] == precommit["action_id"]
    assert error["error_class"] == "exception"
    assert error["error_detail_digest"].startswith("sha256:")


def test_action_gate_arun_records_action_error_on_failure() -> None:
    bundle = Bundle()
    gate = ActionGate(bundle=bundle, actor_id="actor-1")
    action = ActionGateInput(
        action_type="delete_records",
        target_resource_id="svc-prod",
        intended_effect="purge accounts",
        action_parameters={"scope": "all"},
    )

    async def boom() -> str:
        raise RuntimeError("async sink refused")

    with pytest.raises(RuntimeError, match="async sink refused"):
        asyncio.run(gate.arun(action, boom))

    events = [record.event for record in _user_records(bundle)]
    assert [event["type"] for event in events] == [
        "ai.action.precommit",
        "ai.policy.decision",
        "ai.action.error",
    ]
    assert events[2]["data"]["error_class"] == "exception"
