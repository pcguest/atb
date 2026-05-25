"""Tests for profile workflow SDK helpers."""

from __future__ import annotations

from atb import Bundle
from atb.background_job_tracker import BackgroundJobScheduleInput, BackgroundJobTracker
from atb.data_export_gate import DataExportDeniedError, DataExportGate, DataExportInput
from atb.human_override_gate import (
    HumanOverrideActionInput,
    HumanOverrideApprovalInput,
    HumanOverrideDeniedError,
    HumanOverrideGate,
)
from atb.policy_decision_recorder import PolicyDecisionActionInput, PolicyDecisionRecorder


def _user_types(bundle: Bundle) -> list[str]:
    return [
        record.event["type"]
        for record in bundle.records
        if record.event["type"] != "atb.bundle.manifest"
    ]


def test_data_export_gate_records_profile_sequence() -> None:
    bundle = Bundle()
    gate = DataExportGate(bundle=bundle, actor_id="actor-export")

    gate.run(
        DataExportInput(
            action_type="export_data",
            target_resource_id="dataset-1",
            intended_effect="export approved dataset",
            action_parameters={"format": "csv"},
            action_id="act-export-1",
        ),
        lambda: {"rows": 42},
    )

    assert _user_types(bundle) == [
        "ai.request.received",
        "data.export.precommit",
        "ai.policy.decision",
        "data.export.executed",
        "ai.human.approval",
    ]


def test_data_export_gate_enforce_mode() -> None:
    gate = DataExportGate(mode="enforce", policy=lambda _: {"decision": "deny"})

    try:
        gate.run(
            DataExportInput(
                action_type="export_data",
                target_resource_id="dataset-1",
                intended_effect="export",
                action_parameters={},
            ),
            lambda: "ok",
        )
    except DataExportDeniedError:
        return
    raise AssertionError("expected DataExportDeniedError")


def test_policy_decision_recorder() -> None:
    bundle = Bundle()
    recorder = PolicyDecisionRecorder(bundle=bundle, actor_id="actor-pol")

    action_id = recorder.record(
        PolicyDecisionActionInput(
            action_type="approve_change",
            action_parameters={"ticket": "INC-1"},
            action_id="act-pol-1",
        ),
        {"decision": "allow", "reason_codes": ["approved"]},
    )

    assert action_id == "act-pol-1"
    assert _user_types(bundle) == [
        "ai.request.received",
        "ai.action.precommit",
        "ai.policy.decision",
    ]


def test_human_override_gate() -> None:
    bundle = Bundle()
    gate = HumanOverrideGate(bundle=bundle, actor_id="actor-override")

    gate.run(
        HumanOverrideActionInput(
            action_type="override_action",
            target_resource_id="svc-1",
            intended_effect="run override",
            action_parameters={"mode": "manual"},
            action_id="act-override-1",
        ),
        HumanOverrideApprovalInput(
            approver_id_hash="sha256:approver-1",
            approval_outcome="approved",
        ),
        lambda: "done",
    )

    types = _user_types(bundle)
    assert types.index("ai.human.approval") < types.index("ai.action.precommit")
    assert "ai.action.executed" in types


def test_human_override_gate_enforce_denied() -> None:
    gate = HumanOverrideGate(mode="enforce")

    try:
        gate.run(
            HumanOverrideActionInput(
                action_type="override_action",
                target_resource_id="svc-1",
                intended_effect="run override",
                action_parameters={},
            ),
            HumanOverrideApprovalInput(
                approver_id_hash="sha256:approver-1",
                approval_outcome="denied",
            ),
            lambda: "done",
        )
    except HumanOverrideDeniedError:
        return
    raise AssertionError("expected HumanOverrideDeniedError")


def test_background_job_tracker() -> None:
    bundle = Bundle()
    tracker = BackgroundJobTracker(bundle=bundle)

    tracker.run_job(
        BackgroundJobScheduleInput(
            job_type="nightly_sync",
            trigger_source="cron",
            scheduled_by_id_hash="sha256:scheduler",
            job_id="job-1",
        ),
        "sha256:worker",
        lambda: "ok",
    )

    assert _user_types(bundle) == [
        "ai.job.scheduled",
        "ai.job.started",
        "ai.job.completed",
    ]
