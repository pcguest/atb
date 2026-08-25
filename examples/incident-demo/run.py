"""Run the deterministic, offline ATB incident-forensics demo."""

from __future__ import annotations

import hashlib
import json
import os
import shutil
import subprocess
from collections.abc import Mapping, Sequence
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any

from atb import (
    ActionGate,
    ActionGateDecision,
    ActionGateInput,
    Bundle,
    Event,
    IdentityEvidence,
    compute_hash,
)
from atb.bundle import MANIFEST_EVENT_TYPE, Record
from atb.hash import GENESIS_HASH
from atb.oversight import HumanOverrideEmitter, ToolCallEmitter

SESSION_ID = "sess-incident-demo"
REQUEST_ID = "req-incident-demo"
ACTION_ID = "act-incident-demo"
BASE_TIME = datetime(2026, 8, 1, 9, 0, tzinfo=timezone.utc)
BUNDLE_ID = "0123456789abcdef0123456789abcdef"
OUTPUT_DIR = Path("run.atb/incident-demo")
BUNDLE_PATH = OUTPUT_DIR / "incident.atb"
CONTENT_TAMPERED_PATH = OUTPUT_DIR / "incident-content-tampered.atb"
ORDER_TAMPERED_PATH = OUTPUT_DIR / "incident-order-tampered.atb"
RECORD_TAMPERED_PATH = OUTPUT_DIR / "incident-record-removed.atb"
APP_LOG_PATH = Path("examples/incident-demo/application.log")


class SessionSink:
    """Attach a fixed session and monotonically increasing timestamps."""

    def __init__(self, bundle: Bundle) -> None:
        self.bundle = bundle
        self._next_time = BASE_TIME + timedelta(seconds=1)

    def append(self, event_type: str, payload: Mapping[str, Any]) -> None:
        data = dict(payload)
        data.setdefault("session_id", SESSION_ID)
        timestamp = _timestamp(self._next_time)
        self._next_time += timedelta(seconds=1)
        self.bundle.append(event_type, data, timestamp=timestamp)


def _timestamp(value: datetime) -> str:
    return value.isoformat().replace("+00:00", "Z")


def _new_deterministic_bundle() -> Bundle:
    created_at = _timestamp(BASE_TIME)
    manifest_data = json.dumps(
        {
            "version": "1",
            "created_at": created_at,
            "bundle_id": BUNDLE_ID,
        },
        separators=(",", ":"),
    )
    event = Event(
        seq=0,
        prev_hash=GENESIS_HASH,
        type=MANIFEST_EVENT_TYPE,
        data=manifest_data,
        hash_algo="sha256",
        timestamp=created_at,
    ).to_dict()
    return Bundle(
        name="agent incident demo",
        records=[Record(event=event, hash=compute_hash(event, GENESIS_HASH))],
    )


def _write_bundle() -> None:
    bundle = _new_deterministic_bundle()
    sink = SessionSink(bundle)
    sink.append(
        "ai.request.received",
        {
            "request_id": REQUEST_ID,
            "actor_id_hash": "sha256:release-agent",
            "purpose_tag": "policy_decision",
        },
    )

    ToolCallEmitter(sink).emit(
        "deploy_service",
        actor_id="agent:release-bot",
        tool_input_digest="sha256:deploy-parameters",
    )

    evidence = IdentityEvidence(
        identity_provider="https://idp.example.test",
        subject="reviewer@example.test",
        auth_context="mfa",
        assertion_type="jwt",
        assertion_digest="sha256:"
        + hashlib.sha256(b"demo-reviewer-assertion").hexdigest(),
    )
    gate = ActionGate(
        mode="log_only",
        policy=lambda _: ActionGateDecision(
            decision="deny",
            reason_codes=("change_window_closed",),
            policy_id="release-policy",
            policy_version="2026-08",
        ),
        event_sink=sink,
    )
    try:
        gate.run(
            ActionGateInput(
                action_type="deploy_service",
                target_resource_id="production/api",
                intended_effect="deploy release candidate",
                action_parameters={"version": "2026.08.01"},
                subject_id="agent:release-bot",
                action_id=ACTION_ID,
                effective_scope="production-deployer",
                identity_evidence=evidence,
            ),
            lambda: (_ for _ in ()).throw(RuntimeError("deployment rejected")),
        )
    except RuntimeError:
        pass

    HumanOverrideEmitter(sink).emit(
        "Escalated denied deployment for incident review",
        actor_id="reviewer@example.test",
        overridden_action_id=ACTION_ID,
        identity_evidence=evidence,
    )
    sink.append(
        "atb.session.close",
        {
            "actor_id": "agent:release-bot",
            "model": "",
            "exchange_count": 0,
            "total_tokens": 0,
            "closed_at": _timestamp(BASE_TIME + timedelta(seconds=8)),
        },
    )
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    bundle.save(BUNDLE_PATH)

    raw = BUNDLE_PATH.read_bytes()
    content_tampered = raw.replace(
        b"change_window_closed", b"change_window_open", 1
    )
    if content_tampered == raw:
        raise RuntimeError("demo tamper marker not found")
    CONTENT_TAMPERED_PATH.write_bytes(content_tampered)

    records = raw.splitlines(keepends=True)
    if len(records) < 5:
        raise RuntimeError("demo bundle has too few records for tamper vectors")
    reordered = list(records)
    reordered[2], reordered[3] = reordered[3], reordered[2]
    ORDER_TAMPERED_PATH.write_bytes(b"".join(reordered))
    RECORD_TAMPERED_PATH.write_bytes(b"".join(records[:2] + records[3:]))


def _run_cli(atb: str, args: Sequence[str], expected: int = 0) -> str:
    print(f"\n$ {atb} {' '.join(args)}")
    result = subprocess.run(
        [atb, *args],
        check=False,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    print(result.stdout, end="")
    if result.returncode != expected:
        raise RuntimeError(
            f"command exited {result.returncode}; expected {expected}: {' '.join(args)}"
        )
    return result.stdout


def main() -> None:
    atb = os.environ.get("ATB_BIN", "atb")
    if shutil.which(atb) is None and not Path(atb).exists():
        raise SystemExit("ATB CLI not found; set ATB_BIN=/path/to/atb")

    _write_bundle()
    first_digest = hashlib.sha256(BUNDLE_PATH.read_bytes()).hexdigest()
    _write_bundle()
    second_digest = hashlib.sha256(BUNDLE_PATH.read_bytes()).hexdigest()
    if first_digest != second_digest:
        raise RuntimeError("demo bundle is not byte-for-byte deterministic")

    application_log = APP_LOG_PATH.read_text(encoding="utf-8")
    if "deployment failed" not in application_log:
        raise RuntimeError("application log does not contain the generic failure")
    for omitted_fact in (
        "deploy_service",
        "change_window_closed",
        "reviewer@example.test",
        "tool_without_approval",
    ):
        if omitted_fact in application_log:
            raise RuntimeError(
                f"application log unexpectedly reveals omitted fact: {omitted_fact}"
            )
    print("Ordinary application log (insufficient to reconstruct the incident):")
    print(application_log, end="")

    _run_cli(atb, ["verify", "--bundle", str(BUNDLE_PATH)])
    sessions = _run_cli(atb, ["incident", "list", "--bundle", str(BUNDLE_PATH)])
    report = _run_cli(
        atb,
        [
            "incident",
            "report",
            "--bundle",
            str(BUNDLE_PATH),
            "--session",
            SESSION_ID,
        ],
    )
    if "tool_without_approval" not in sessions + report:
        raise RuntimeError("expected tool_without_approval finding was not reported")

    for tampered_path in (
        CONTENT_TAMPERED_PATH,
        ORDER_TAMPERED_PATH,
        RECORD_TAMPERED_PATH,
    ):
        _run_cli(atb, ["verify", "--bundle", str(tampered_path)], expected=2)
    print(f"\nDeterministic bundle SHA-256: {first_digest}")
    print(f"Viewer: {atb} view --bundle {BUNDLE_PATH}")


if __name__ == "__main__":
    main()
