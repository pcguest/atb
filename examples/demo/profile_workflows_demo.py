#!/usr/bin/env python3
"""Profile workflow SDK demo — data export ops on a fresh bundle.

Run from repository root:
    python examples/demo/profile_workflows_demo.py

Requires `atb` on PATH (or set ATB_BIN).
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

from atb import Bundle
from atb.background_job_tracker import BackgroundJobScheduleInput, BackgroundJobTracker
from atb.data_export_gate import DataExportGate, DataExportInput
from atb.human_override_gate import HumanOverrideActionInput, HumanOverrideApprovalInput, HumanOverrideGate
from atb.policy_decision_recorder import PolicyDecisionActionInput, PolicyDecisionRecorder


def atb_bin() -> str:
    return os.environ.get("ATB_BIN", shutil.which("atb") or "atb")


def verify_profile(bundle_path: Path, profile_id: str, *, required: bool = True) -> dict:
    proc = subprocess.run(
        [
            atb_bin(),
            "verify",
            "--bundle",
            str(bundle_path),
            "--profile",
            profile_id,
            "--format",
            "json",
        ],
        check=False,
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0 and required:
        raise subprocess.CalledProcessError(proc.returncode, proc.args, proc.stdout, proc.stderr)
    if proc.stdout.strip():
        return json.loads(proc.stdout)
    raise subprocess.CalledProcessError(proc.returncode, proc.args, proc.stdout, proc.stderr)


def main() -> int:
    work = Path(tempfile.mkdtemp(prefix="atb-profile-demo-"))
    bundle_path = work / "bundle.atb"
    export_checkpoint = work / "export-phase.atb"

    bundle = Bundle()
    job_tracker = BackgroundJobTracker(bundle=bundle, actor_id="scheduler-ops")
    export_gate = DataExportGate(bundle=bundle, actor_id="export-runner", record_approval=True)

    def run_scheduled_export() -> dict[str, int]:
        return export_gate.run(
            DataExportInput(
                action_type="export_analytics",
                target_resource_id="warehouse.analytics",
                intended_effect="export weekly rollup",
                action_parameters={"format": "parquet", "rows": 1200},
                action_id="act-scheduled-export",
            ),
            lambda: {"rows": 1200, "bytes": 450_000},
        )

    job_tracker.run_job(
        BackgroundJobScheduleInput(
            job_type="weekly_analytics_export",
            trigger_source="cron",
            scheduled_by_id_hash="sha256:scheduler-nightly",
            job_id="job-export-demo",
        ),
        worker_id_hash="sha256:worker-export-01",
        fn=run_scheduled_export,
    )

    bundle.save(str(export_checkpoint))
    export_report = verify_profile(export_checkpoint, "atb.profile.data_export")

    policy_recorder = PolicyDecisionRecorder(bundle=bundle, actor_id="policy-engine")
    policy_recorder.record(
        PolicyDecisionActionInput(
            action_type="approve_retention_exception",
            action_parameters={"ticket": "LEGAL-88", "dataset": "warehouse.pii_requests"},
            action_id="act-retention-policy",
        ),
        {"decision": "allow", "reason_codes": ["legal_hold_active"]},
    )

    override_gate = HumanOverrideGate(bundle=bundle, actor_id="ops-lead")
    override_gate.run(
        HumanOverrideActionInput(
            action_type="export_pii_subset",
            target_resource_id="warehouse.pii_requests",
            intended_effect="legal hold export",
            action_parameters={"case_id": "LEGAL-88"},
            action_id="act-override-export",
        ),
        HumanOverrideApprovalInput(
            approver_id_hash="sha256:legal-lead",
            approval_outcome="approved",
            justification_digest="sha256:legal-hold-justification",
        ),
        lambda: {"rows": 14},
    )

    bundle.save(str(bundle_path))
    policy_report = verify_profile(bundle_path, "atb.profile.policy_decision", required=False)
    override_report = verify_profile(bundle_path, "atb.profile.human_override", required=False)

    print(
        json.dumps(
            {
                "data_export": {
                    "pass": export_report["pass"],
                    "cas_grade": export_report["cas_grade"],
                    "profile_id": export_report["profile_id"],
                },
                "policy_decision": {
                    "pass": policy_report["pass"],
                    "cas_grade": policy_report["cas_grade"],
                },
                "human_override": {
                    "pass": override_report["pass"],
                    "cas_grade": override_report["cas_grade"],
                },
            },
            indent=2,
        )
    )

    if not export_report.get("pass"):
        print("data_export verify failed", file=sys.stderr)
        return 1

    bundle.verify()
    print(f"bundle: {bundle_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
