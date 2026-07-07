#!/usr/bin/env python3
"""Create and review an offline ATB agent-incident bundle."""

from __future__ import annotations

import hashlib
import os
import shutil
import subprocess
from collections.abc import Mapping
from datetime import UTC, datetime, timedelta
from pathlib import Path
from typing import Any

from atb import (
    ActionGate,
    ActionGateDecision,
    ActionGateInput,
    Bundle,
    IdentityEvidence,
)
from atb.oversight import HumanOverrideEmitter, ToolCallEmitter

SESSION_ID = "sess-laptop-demo"
BUNDLE_PATH = Path("run.atb/agent-incident-demo.atb")


class SessionSink:
    """Attach one session ID to SDK-emitted oversight and action events."""

    def __init__(self, bundle: Bundle, session_id: str) -> None:
        self.bundle = bundle
        self.session_id = session_id
        self._next_time = datetime.now(UTC)

    def append(self, event_type: str, payload: Mapping[str, Any]) -> None:
        data = dict(payload)
        data.setdefault("session_id", self.session_id)
        timestamp = self._next_time.isoformat().replace("+00:00", "Z")
        self._next_time += timedelta(milliseconds=1)
        self.bundle.append(event_type, data, timestamp=timestamp)


def run_cli(atb: str, *args: str) -> None:
    print(f"\n$ {atb} {' '.join(args)}")
    subprocess.run([atb, *args], check=True)


def main() -> None:
    bundle = Bundle(name="agent incident laptop demo")
    sink = SessionSink(bundle, SESSION_ID)
    sink.append(
        "ai.request.received",
        {
            "session_id": SESSION_ID,
            "request_id": "req-laptop-demo",
            "actor_id_hash": "sha256:operator-demo",
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
        + hashlib.sha256(b"demo-jwt-header-and-payload").hexdigest(),
    )
    gate = ActionGate(
        mode="log_only",
        policy=lambda _: ActionGateDecision(
            decision="deny",
            reason_codes=("change_window_closed",),
            policy_id="release-policy",
            policy_version="2026-06",
        ),
        event_sink=sink,
    )
    try:
        gate.run(
            ActionGateInput(
                action_type="deploy_service",
                target_resource_id="production/api",
                intended_effect="deploy release candidate",
                action_parameters={"version": "2026.06.15"},
                subject_id="agent:release-bot",
                action_id="act-laptop-demo",
                effective_scope="production-deployer",
                identity_evidence=evidence,
            ),
            lambda: (_ for _ in ()).throw(RuntimeError("simulated sink rejection")),
        )
    except RuntimeError:
        pass

    HumanOverrideEmitter(sink).emit(
        "Escalated denied deployment for incident review",
        actor_id="reviewer@example.test",
        overridden_action_id="act-laptop-demo",
        identity_evidence=evidence,
    )
    sink.append(
        "atb.session.close",
        {
            "session_id": SESSION_ID,
            "actor_id": "agent:release-bot",
            "model": "",
            # The gated action was denied before any LLM exchange happened.
            "exchange_count": 0,
            "total_tokens": 0,
            "closed_at": datetime.now(UTC).isoformat().replace("+00:00", "Z"),
        },
    )
    bundle.save(BUNDLE_PATH)

    atb = os.environ.get("ATB_BIN", "atb")
    if shutil.which(atb) is None and not Path(atb).exists():
        raise SystemExit("ATB CLI not found; set ATB_BIN=/path/to/atb")

    run_cli(
        atb,
        "verify",
        "--bundle",
        str(BUNDLE_PATH),
        "--profile",
        "atb.profile.policy_decision",
        "--format",
        "json",
    )
    run_cli(
        atb,
        "trust-report",
        str(BUNDLE_PATH),
        "--profile",
        "atb.profile.policy_decision",
        "--format",
        "text",
    )
    run_cli(atb, "incident", "list", "--bundle", str(BUNDLE_PATH))
    run_cli(
        atb,
        "incident",
        "report",
        "--bundle",
        str(BUNDLE_PATH),
        "--session",
        SESSION_ID,
    )
    print(
        f"\nOpen locally: {atb} view --bundle {BUNDLE_PATH} --profile atb.profile.policy_decision"
    )


if __name__ == "__main__":
    main()
