"""Tests for AutomationSession disk-backed workflow harness."""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from atb import Bundle
from atb.action_gate import ActionGateInput
from atb.agent_client import AgentClient
from atb.automation_session import AutomationSession, is_capture_environment
from atb.policy_decision_recorder import PolicyDecisionActionInput


@pytest.fixture(autouse=True)
def _disable_agent_by_default(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("ATB_AGENT_DISABLE", "1")
    monkeypatch.delenv("ATB_AGENT_URL", raising=False)
    monkeypatch.delenv("ATB_AGENT_AUTO", raising=False)


def _user_types(bundle: Bundle) -> list[str]:
    return [
        record.event["type"]
        for record in bundle.records
        if record.event["type"] != "atb.bundle.manifest"
    ]


def test_chained_rag_workflow_on_one_request() -> None:
    session = AutomationSession(
        bundle=Bundle(),
        save_path="run.atb/bundle.atb",
        actor_id="actor-rag",
        purpose_tag="rag_answer",
        request_id="req-chain-1",
    )

    session.log_retrieval(
        "what is ATB?",
        "docs-v1",
        "2026-05",
        3,
        [{"id": "doc-1"}],
    )
    session.log_model_invocation(
        "openai",
        "gpt-4o",
        "Answer using context",
        parameters={"temperature": 0},
    )
    session.log_model_output("ATB is an audit substrate.")
    session.log_response_sent("ATB is an audit substrate.")

    assert _user_types(session.bundle) == [
        "ai.request.received",
        "ai.retrieval.executed",
        "ai.model.invoked",
        "ai.model.output",
        "ai.response.sent",
    ]

    request = next(
        record.event["data"]
        for record in session.bundle.records
        if record.event["type"] == "ai.request.received"
    )
    assert request["request_id"] == "req-chain-1"
    assert request["purpose_tag"] == "rag_answer"


def test_privileged_tool_action_with_commit() -> None:
    session = AutomationSession(
        bundle=Bundle(),
        save_path="run.atb/bundle.atb",
        actor_id="actor-tool",
    )

    session.run_tool_action(
        ActionGateInput(
            action_type="restart_service",
            target_resource_id="svc-api",
            intended_effect="roll restart",
            action_parameters={"service": "api"},
            action_id="act-tool-1",
        ),
        lambda: {"status": "restarted"},
    )

    assert _user_types(session.bundle) == [
        "ai.request.received",
        "ai.action.precommit",
        "ai.policy.decision",
        "ai.action.executed",
        "ai.action.committed",
    ]


def test_from_capture_environment(tmp_path: Path) -> None:
    bundle_path = tmp_path / "run.atb" / "bundle.atb"
    env = {
        "ATB_BUNDLE_PATH": str(bundle_path),
        "ATB_CAPTURE_RUN_ID": "cap-deadbeef",
        "ATB_CAPTURE_MODE": "run",
    }
    assert is_capture_environment(env) is True

    session = AutomationSession.from_capture_environment(env)
    assert session is not None
    assert session.capture_run_id == "cap-deadbeef"
    assert session.default_purpose_tag == "run"
    assert session.bundle is not None


def test_from_capture_environment_returns_none_outside_capture() -> None:
    assert AutomationSession.from_capture_environment({}) is None


def test_requires_bundle_path_when_not_in_capture_env(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.delenv("ATB_BUNDLE_PATH", raising=False)
    with pytest.raises(ValueError, match="bundle_path or ATB_BUNDLE_PATH"):
        AutomationSession()


def test_snapshot_on_close() -> None:
    session = AutomationSession(bundle=Bundle(), save_path="run.atb/bundle.atb")
    session.begin_request("policy_decision")
    session.log_policy_decision(
        PolicyDecisionActionInput(
            action_type="approve",
            action_parameters={"ticket": "INC-9"},
        ),
        {"decision": "allow", "reason_codes": ["ok"]},
    )
    session.close(snapshot_name="review_boundary")

    assert "atb.snapshot" in _user_types(session.bundle)


def test_open_with_direct_bundle_path(tmp_path: Path) -> None:
    bundle_path = tmp_path / "capture.atb"
    session = AutomationSession.open(
        bundle_path=bundle_path,
        actor_id="actor-disk",
        purpose_tag="rag_answer",
        request_id="req-disk-1",
    )

    session.log_retrieval(
        "hello",
        "docs-v1",
        "2026-05",
        1,
        [{"id": "doc-1"}],
    )
    session.log_model_invocation("openai", "gpt-4o", "Answer")
    session.close()

    assert bundle_path.exists()
    loaded = Bundle.load(bundle_path)
    assert _user_types(loaded) == [
        "ai.request.received",
        "ai.retrieval.executed",
        "ai.model.invoked",
    ]
    loaded.verify()


def test_open_from_capture_env_vars(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    bundle_path = tmp_path / "run.atb" / "bundle.atb"
    monkeypatch.setenv("ATB_BUNDLE_PATH", str(bundle_path))
    monkeypatch.setenv("ATB_CAPTURE_RUN_ID", "cap-env-1")
    monkeypatch.setenv("ATB_CAPTURE_MODE", "run")

    session = AutomationSession.open(actor_id="actor-env", request_id="req-env-1")
    session.log_model_invocation("openai", "gpt-4o", "hi")
    session.close()

    assert bundle_path.exists()
    loaded = Bundle.load(bundle_path)
    assert "ai.model.invoked" in _user_types(loaded)
    loaded.verify()


def test_open_resumes_existing_bundle(tmp_path: Path) -> None:
    bundle_path = tmp_path / "resume.atb"
    first = AutomationSession.open(bundle_path=bundle_path, request_id="req-resume")
    first.log_model_invocation("openai", "gpt-4o", "first hop")
    first.close()

    second = AutomationSession.open(bundle_path=bundle_path, request_id="req-resume-2")
    second.log_model_output("continued")
    second.close()

    loaded = Bundle.load(bundle_path)
    assert _user_types(loaded) == [
        "ai.request.received",
        "ai.model.invoked",
        "ai.model.output",
    ]
    loaded.verify()


def test_agent_disable_forces_disk_mode(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    bundle_path = tmp_path / "forced-disk.atb"
    monkeypatch.setenv("ATB_AGENT_URL", "http://127.0.0.1:6180")
    monkeypatch.setenv("ATB_AGENT_DISABLE", "1")

    session = AutomationSession.open(
        bundle_path=bundle_path,
        actor_id="actor-disk",
        request_id="req-forced-disk",
    )
    session.log_model_invocation("openai", "gpt-4o", "disk only")
    session.close()

    assert session.is_using_agent() is False
    assert bundle_path.exists()
    loaded = Bundle.load(bundle_path)
    assert "ai.model.invoked" in [record.event["type"] for record in loaded.records]


def test_routes_workflow_events_through_agent(tmp_path: Path) -> None:
    events: list[dict[str, object]] = []
    calls: list[str] = []

    def request_fn(url, *, method, body=None, headers=None, timeout_ms=30_000):
        calls.append(url)
        if url.endswith("/v1/session/open"):
            return type(
                "R",
                (),
                {
                    "status": 201,
                    "body": json.dumps(
                        {
                            "session_id": "sess_agent",
                            "bundle_path": str(tmp_path / "agent" / "bundle.atb"),
                        }
                    ),
                },
            )()
        if "/event" in url:
            parsed = json.loads(body or "{}")
            events.append(parsed)
            return type(
                "R", (), {"status": 202, "body": json.dumps({"status": "queued"})}
            )()
        if "/close" in url:
            return type(
                "R",
                (),
                {
                    "status": 200,
                    "body": json.dumps(
                        {
                            "session_id": "sess_agent",
                            "bundle_path": str(tmp_path / "agent" / "bundle.atb"),
                            "head_hash": "deadbeef",
                            "event_count": len(events),
                            "opened_at": "2026-05-25T12:00:00Z",
                            "closed_at": "2026-05-25T12:01:00Z",
                        }
                    ),
                },
            )()
        return type(
            "R", (), {"status": 404, "body": json.dumps({"error": "not found"})}
        )()

    agent_client = AgentClient("http://127.0.0.1:6180", request_fn=request_fn)
    hinted_path = tmp_path / "hint.atb"
    session = AutomationSession(
        bundle_path=hinted_path,
        agent_client=agent_client,
        actor_id="actor-agent",
        purpose_tag="rag_answer",
        request_id="req-agent-1",
    )

    assert session.is_using_agent() is True
    session.log_retrieval(
        "what is ATB?",
        "docs-v1",
        "2026-05",
        3,
        [{"id": "doc-1"}],
    )
    session.log_model_invocation("openai", "gpt-4o", "Answer using context")
    session.close()

    assert [event["event_type"] for event in events] == [
        "ai.request.received",
        "ai.retrieval.executed",
        "ai.model.invoked",
    ]
    assert events[0]["payload"]["purpose_tag"] == "rag_answer"
    assert _user_types(session.bundle) == []
    assert not hinted_path.exists()
    assert any("/close" in url for url in calls)
