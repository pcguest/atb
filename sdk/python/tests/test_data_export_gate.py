from __future__ import annotations

import pytest

from atb import Bundle, IdentityEvidence
from atb.data_export_gate import DataExportApproval, DataExportGate, DataExportInput


def _user_event_types(bundle: Bundle) -> list[str]:
    return [
        record.event["type"]
        for record in bundle.records
        if record.event["type"] != "atb.bundle.manifest"
    ]


def test_data_export_gate_records_export_error_on_failure() -> None:
    bundle = Bundle()
    gate = DataExportGate(bundle=bundle, actor_id="actor-1", record_approval=False)
    action = DataExportInput(
        action_type="export_dataset",
        target_resource_id="dataset-1",
        intended_effect="export to s3",
        action_parameters={"rows": 100},
    )

    def boom() -> str:
        raise RuntimeError("sink refused")

    with pytest.raises(RuntimeError, match="sink refused"):
        gate.run(action, boom)

    types = _user_event_types(bundle)
    assert "data.export.error" in types
    assert "data.export.executed" not in types

    errors = [
        record.event["data"]
        for record in bundle.records
        if record.event["type"] == "data.export.error"
    ]
    assert len(errors) == 1
    assert errors[0]["error_class"] == "exception"
    assert errors[0]["error_detail_digest"].startswith("sha256:")


def test_data_export_gate_records_reviewer_identity_evidence() -> None:
    bundle = Bundle()
    gate = DataExportGate(
        bundle=bundle,
        actor_id="actor-1",
        record_approval=lambda _: DataExportApproval(
            approver_id_hash="sha256:reviewer",
            identity_evidence=IdentityEvidence(
                identity_provider="https://idp.example",
                subject="reviewer-1",
                assertion_type="x509",
                assertion_digest="sha256:certificate",
            ),
        ),
    )
    action = DataExportInput(
        action_type="export_dataset",
        target_resource_id="dataset-1",
        intended_effect="export to s3",
        action_parameters={"rows": 100},
    )

    gate.run(action, lambda: "ok")

    approval = next(
        record.event["data"]
        for record in bundle.records
        if record.event["type"] == "ai.human.approval"
    )
    assert approval["identity_evidence"]["assertion_type"] == "x509"
