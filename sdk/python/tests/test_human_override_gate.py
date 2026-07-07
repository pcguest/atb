from __future__ import annotations

import pytest

from atb import Bundle, IdentityEvidence
from atb.human_override_gate import (
    HumanOverrideActionInput,
    HumanOverrideApprovalInput,
    HumanOverrideGate,
)


def _user_event_types(bundle: Bundle) -> list[str]:
    return [
        record.event["type"]
        for record in bundle.records
        if record.event["type"] != "atb.bundle.manifest"
    ]


def test_human_override_gate_records_action_error_on_failure() -> None:
    bundle = Bundle()
    gate = HumanOverrideGate(bundle=bundle, actor_id="actor-1")
    action = HumanOverrideActionInput(
        action_type="delete_records",
        target_resource_id="svc-prod",
        intended_effect="purge accounts",
        action_parameters={"scope": "all"},
    )
    approval = HumanOverrideApprovalInput(approver_id_hash="sha256:approver")

    def boom() -> str:
        raise RuntimeError("sink refused")

    with pytest.raises(RuntimeError, match="sink refused"):
        gate.run(action, approval, boom)

    types = _user_event_types(bundle)
    assert types[-1] == "ai.action.error"
    assert "ai.action.executed" not in types
    error = bundle.records[-1].event["data"]
    assert error["error_class"] == "exception"
    assert error["error_detail_digest"].startswith("sha256:")


def test_human_override_gate_records_principal_on_precommit() -> None:
    from atb.action_gate import ActionPrincipal

    bundle = Bundle()
    gate = HumanOverrideGate(bundle=bundle, actor_id="actor-1")
    action = HumanOverrideActionInput(
        action_type="override_action",
        target_resource_id="svc-1",
        intended_effect="run override",
        action_parameters={"x": 1},
        principal=ActionPrincipal(type="human", id_hash="sha256:op-1"),
    )
    approval = HumanOverrideApprovalInput(approver_id_hash="sha256:approver")

    gate.run(action, approval, lambda: "ok")

    precommit = next(
        record.event["data"]
        for record in bundle.records
        if record.event["type"] == "ai.action.precommit"
    )
    assert precommit["principal"] == {"type": "human", "id_hash": "sha256:op-1"}


def test_human_override_gate_records_reviewer_identity_evidence() -> None:
    bundle = Bundle()
    gate = HumanOverrideGate(bundle=bundle, actor_id="actor-1")
    action = HumanOverrideActionInput(
        action_type="override_action",
        target_resource_id="svc-1",
        intended_effect="run override",
        action_parameters={"x": 1},
    )
    approval = HumanOverrideApprovalInput(
        approver_id_hash="sha256:approver",
        identity_evidence=IdentityEvidence(
            identity_provider="https://idp.example",
            subject="reviewer-1",
            assertion_type="jwt",
            assertion_digest="sha256:assertion",
        ),
    )

    gate.run(action, approval, lambda: "ok")

    approval_event = next(
        record.event["data"]
        for record in bundle.records
        if record.event["type"] == "ai.human.approval"
    )
    assert approval_event["identity_evidence"] == {
        "identity_provider": "https://idp.example",
        "subject": "reviewer-1",
        "assertion_type": "jwt",
        "assertion_digest": "sha256:assertion",
    }
