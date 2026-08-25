"""Tests for the internal ATB Agent HTTP client."""

from __future__ import annotations

import json
from io import BytesIO

import pytest

from atb.agent_client import (
    DEFAULT_AGENT_URL,
    MAX_AGENT_RESPONSE_BYTES,
    AgentClient,
    AgentClientError,
    _read_response,
    is_explicit_agent_url,
    probe_agent_health,
    resolve_agent_base_url,
    try_create_agent_client,
)


def _mock_request(handler):
    def request_fn(url, *, method, body=None, headers=None, timeout_ms=30_000):
        return handler(
            url, method=method, body=body, headers=headers, timeout_ms=timeout_ms
        )

    return request_fn


def test_resolve_agent_base_url() -> None:
    assert resolve_agent_base_url({"ATB_AGENT_DISABLE": "1"}) is None
    assert resolve_agent_base_url({"ATB_AGENT_URL": "http://127.0.0.1:9999/"}) == (
        "http://127.0.0.1:9999"
    )
    assert resolve_agent_base_url({"ATB_AGENT_AUTO": "1"}) == DEFAULT_AGENT_URL


def test_agent_client_rejects_non_loopback_url() -> None:
    with pytest.raises(ValueError, match="loopback"):
        AgentClient("https://agent.example.test")


def test_agent_response_is_bounded() -> None:
    with pytest.raises(AgentClientError, match="response exceeds"):
        _read_response(BytesIO(b"x" * (MAX_AGENT_RESPONSE_BYTES + 1)), 200)


def test_agent_client_open_append_close() -> None:
    calls: list[dict[str, str | None]] = []

    def handler(url, *, method, body=None, headers=None, timeout_ms=30_000):
        calls.append({"url": url, "method": method, "body": body})
        if url.endswith("/v1/session/open"):
            return type(
                "R",
                (),
                {
                    "status": 201,
                    "body": json.dumps(
                        {
                            "session_id": "sess_test",
                            "bundle_path": "/tmp/agent/bundle.atb",
                            "actor_id": "actor-1",
                        }
                    ),
                },
            )()
        if "/event" in url:
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
                            "session_id": "sess_test",
                            "bundle_path": "/tmp/agent/bundle.atb",
                            "head_hash": "abc123",
                            "event_count": 2,
                            "opened_at": "2026-05-25T12:00:00Z",
                            "closed_at": "2026-05-25T12:01:00Z",
                        }
                    ),
                },
            )()
        return type(
            "R", (), {"status": 404, "body": json.dumps({"error": "not found"})}
        )()

    client = AgentClient("http://127.0.0.1:6180", request_fn=_mock_request(handler))
    opened = client.open_session(
        actor_id="actor-1",
        purpose_tag="rag_answer",
        bundle_path="/tmp/agent/bundle.atb",
    )
    assert opened.session_id == "sess_test"
    assert opened.bundle_path == "/tmp/agent/bundle.atb"

    client.append_event("ai.model.invoked", {"request_id": "req-1"})
    closed = client.close_session(snapshot_name="review_boundary")
    assert closed.event_count == 2

    assert calls[0]["url"] == "http://127.0.0.1:6180/v1/session/open"
    assert json.loads(calls[0]["body"]) == {
        "actor_id": "actor-1",
        "purpose_tag": "rag_answer",
        "bundle_path": "/tmp/agent/bundle.atb",
    }
    assert calls[1]["url"] == "http://127.0.0.1:6180/v1/session/sess_test/event"
    assert json.loads(calls[1]["body"]) == {
        "event_type": "ai.model.invoked",
        "payload": {"request_id": "req-1"},
    }
    assert calls[2]["url"] == "http://127.0.0.1:6180/v1/session/sess_test/close"
    assert json.loads(calls[2]["body"]) == {"snapshot_name": "review_boundary"}


def test_agent_client_maps_http_errors() -> None:
    def handler(url, *, method, body=None, headers=None, timeout_ms=30_000):
        if url.endswith("/v1/session/open"):
            return type(
                "R",
                (),
                {
                    "status": 201,
                    "body": json.dumps(
                        {
                            "session_id": "sess_test",
                            "bundle_path": "/tmp/agent/bundle.atb",
                        }
                    ),
                },
            )()
        return type(
            "R", (), {"status": 404, "body": json.dumps({"error": "session not found"})}
        )()

    client = AgentClient("http://127.0.0.1:6180", request_fn=_mock_request(handler))
    client.open_session()
    with pytest.raises(AgentClientError) as exc:
        client.append_event("ai.model.invoked", {})
    assert exc.value.status == 404
    assert "session not found" in str(exc.value)


def test_agent_client_escapes_session_id_path_segments() -> None:
    calls: list[str] = []

    def handler(url, *, method, body=None, headers=None, timeout_ms=30_000):
        calls.append(url)
        if url.endswith("/v1/session/open"):
            return type(
                "R",
                (),
                {
                    "status": 201,
                    "body": json.dumps(
                        {
                            "session_id": "sess/with space",
                            "bundle_path": "/tmp/agent/bundle.atb",
                        }
                    ),
                },
            )()
        return type("R", (), {"status": 202, "body": "{}"})()

    client = AgentClient("http://127.0.0.1:6180", request_fn=_mock_request(handler))
    client.open_session()
    client.append_event("ai.model.invoked", {})
    assert calls[1].endswith("/v1/session/sess%2Fwith%20space/event")


def test_agent_client_maps_409_conflict() -> None:
    def handler(url, *, method, body=None, headers=None, timeout_ms=30_000):
        if url.endswith("/v1/session/open"):
            return type(
                "R",
                (),
                {
                    "status": 201,
                    "body": json.dumps(
                        {
                            "session_id": "sess_test",
                            "bundle_path": "/tmp/agent/bundle.atb",
                        }
                    ),
                },
            )()
        return type(
            "R",
            (),
            {"status": 409, "body": json.dumps({"error": "session already closed"})},
        )()

    client = AgentClient("http://127.0.0.1:6180", request_fn=_mock_request(handler))
    client.open_session()
    with pytest.raises(AgentClientError) as exc:
        client.close_session()
    assert exc.value.status == 409
    assert "session already closed" in str(exc.value)


def test_try_create_agent_client_auto_health_failure() -> None:
    request_fn = _mock_request(
        lambda url, **kwargs: type("R", (), {"status": 503, "body": ""})()
    )
    assert try_create_agent_client({"ATB_AGENT_AUTO": "1"}, request_fn) is None


def test_try_create_agent_client_explicit_url_skips_health() -> None:
    request_fn = _mock_request(
        lambda url, **kwargs: type("R", (), {"status": 503, "body": ""})()
    )
    client = try_create_agent_client(
        {"ATB_AGENT_URL": "http://127.0.0.1:6180"}, request_fn
    )
    assert client is not None
    assert is_explicit_agent_url({"ATB_AGENT_URL": "http://127.0.0.1:6180"}) is True


def test_probe_agent_health_and_auto_discovery() -> None:
    def handler(url, *, method, body=None, headers=None, timeout_ms=30_000):
        if url.endswith("/healthz"):
            return type(
                "R", (), {"status": 200, "body": json.dumps({"status": "ok"})}
            )()
        return type("R", (), {"status": 404, "body": ""})()

    request_fn = _mock_request(handler)
    assert probe_agent_health("http://127.0.0.1:6180", request_fn) is True
    assert try_create_agent_client({"ATB_AGENT_AUTO": "1"}, request_fn) is not None
