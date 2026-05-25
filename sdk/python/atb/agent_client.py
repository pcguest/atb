"""Internal HTTP client for the local ATB Agent capture API (v1).

Not part of the public SDK surface.
"""

from __future__ import annotations

import json
import os
import urllib.error
import urllib.request
from collections.abc import Callable, Mapping
from dataclasses import dataclass
from typing import Any

DEFAULT_AGENT_URL = "http://127.0.0.1:6180"


class AgentClientError(Exception):
    """Raised when the Agent HTTP API returns an error response."""

    def __init__(self, message: str, status: int) -> None:
        super().__init__(message)
        self.status = status


@dataclass(frozen=True)
class AgentHttpResponse:
    status: int
    body: str


AgentRequestFn = Callable[..., AgentHttpResponse]


@dataclass(frozen=True)
class AgentOpenResult:
    session_id: str
    bundle_path: str
    actor_id: str | None = None
    profile_id: str | None = None
    purpose_tag: str | None = None


@dataclass(frozen=True)
class AgentCloseResult:
    session_id: str
    bundle_path: str
    head_hash: str
    event_count: int
    opened_at: str
    closed_at: str
    profile_id: str | None = None


def resolve_agent_base_url(env: Mapping[str, str | None] | None = None) -> str | None:
    lookup = os.environ if env is None else env
    disabled = (lookup.get("ATB_AGENT_DISABLE") or "").strip().lower()
    if disabled in {"1", "true"}:
        return None
    explicit = (lookup.get("ATB_AGENT_URL") or "").strip()
    if explicit in {"0", "false"}:
        return None
    if explicit:
        return explicit.rstrip("/")
    auto = (lookup.get("ATB_AGENT_AUTO") or "").strip().lower()
    if auto in {"1", "true"}:
        return DEFAULT_AGENT_URL
    return None


def is_explicit_agent_url(env: Mapping[str, str | None] | None = None) -> bool:
    lookup = os.environ if env is None else env
    explicit = (lookup.get("ATB_AGENT_URL") or "").strip()
    return bool(explicit and explicit not in {"0", "false"})


def probe_agent_health(
    base_url: str,
    request_fn: AgentRequestFn | None = None,
) -> bool:
    fn = request_fn or sync_agent_request
    try:
        response = fn(
            f"{base_url.rstrip('/')}/healthz",
            method="GET",
            headers={"Accept": "application/json"},
            timeout_ms=250,
        )
        if response.status != 200:
            return False
        parsed = json.loads(response.body)
        return isinstance(parsed, dict) and parsed.get("status") == "ok"
    except (OSError, TimeoutError, json.JSONDecodeError, AgentClientError):
        return False


class AgentClient:
    """HTTP client for Agent session open / event / close."""

    def __init__(
        self,
        base_url: str,
        *,
        request_fn: AgentRequestFn | None = None,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self._request_fn = request_fn or sync_agent_request
        self._session_id: str | None = None
        self.bundle_path: str | None = None

    @property
    def active_session_id(self) -> str | None:
        return self._session_id

    def open_session(
        self,
        *,
        actor_id: str | None = None,
        purpose_tag: str | None = None,
        profile_id: str | None = None,
        bundle_path: str | None = None,
    ) -> AgentOpenResult:
        body: dict[str, str] = {}
        if actor_id and actor_id.strip():
            body["actor_id"] = actor_id.strip()
        if purpose_tag and purpose_tag.strip():
            body["purpose_tag"] = purpose_tag.strip()
        if profile_id and profile_id.strip():
            body["profile_id"] = profile_id.strip()
        if bundle_path and str(bundle_path).strip():
            body["bundle_path"] = str(bundle_path).strip()

        response = self._post("/v1/session/open", body)
        parsed = _parse_json(response, "open session")
        session_id = parsed.get("session_id") or parsed.get("sessionId")
        resolved_bundle_path = parsed.get("bundle_path") or parsed.get("bundlePath")
        if not session_id or not resolved_bundle_path:
            raise AgentClientError(
                "agent open session: missing session_id or bundle_path",
                response.status,
            )
        self._session_id = str(session_id)
        self.bundle_path = str(resolved_bundle_path)
        return AgentOpenResult(
            session_id=self._session_id,
            bundle_path=self.bundle_path,
            actor_id=parsed.get("actor_id") or parsed.get("actorId"),
            profile_id=parsed.get("profile_id") or parsed.get("profileId"),
            purpose_tag=parsed.get("purpose_tag") or parsed.get("purposeTag"),
        )

    def append(self, event_type: str, payload: Mapping[str, Any]) -> None:
        self.append_event(event_type, dict(payload))

    def append_event(self, event_type: str, payload: Mapping[str, Any] | None = None) -> None:
        session_id = self._require_session_id()
        response = self._post(
            f"/v1/session/{urllib.request.quote(session_id, safe='')}/event",
            {
                "event_type": event_type,
                "payload": dict(payload or {}),
            },
        )
        if response.status not in {202, 204}:
            raise _agent_error(response, "append event")

    def close_session(self, *, snapshot_name: str | None = None) -> AgentCloseResult:
        session_id = self._require_session_id()
        body: dict[str, str] = {}
        if snapshot_name and snapshot_name.strip():
            body["snapshot_name"] = snapshot_name.strip()
        response = self._post(
            f"/v1/session/{urllib.request.quote(session_id, safe='')}/close",
            body,
        )
        parsed = _parse_json(response, "close session")
        result = AgentCloseResult(
            session_id=str(parsed.get("session_id") or parsed.get("sessionId") or session_id),
            bundle_path=str(
                parsed.get("bundle_path")
                or parsed.get("bundlePath")
                or self.bundle_path
                or ""
            ),
            profile_id=parsed.get("profile_id") or parsed.get("profileId"),
            head_hash=str(parsed.get("head_hash") or parsed.get("headHash") or ""),
            event_count=int(parsed.get("event_count") or parsed.get("eventCount") or 0),
            opened_at=str(parsed.get("opened_at") or parsed.get("openedAt") or ""),
            closed_at=str(parsed.get("closed_at") or parsed.get("closedAt") or ""),
        )
        self._session_id = None
        return result

    def _require_session_id(self) -> str:
        if not self._session_id:
            raise RuntimeError("agent session is not open")
        return self._session_id

    def _post(self, path: str, body: Mapping[str, Any]) -> AgentHttpResponse:
        return self._request_fn(
            f"{self.base_url}{path}",
            method="POST",
            body=json.dumps(body, separators=(",", ":")),
            headers={
                "Accept": "application/json",
                "Content-Type": "application/json",
            },
            timeout_ms=30_000,
        )


def try_create_agent_client(
    env: Mapping[str, str | None] | None = None,
    request_fn: AgentRequestFn | None = None,
) -> AgentClient | None:
    base_url = resolve_agent_base_url(env)
    if not base_url:
        return None
    fn = request_fn or sync_agent_request
    if not is_explicit_agent_url(env) and not probe_agent_health(base_url, fn):
        return None
    return AgentClient(base_url, request_fn=fn)


def sync_agent_request(
    url: str,
    *,
    method: str,
    body: str | None = None,
    headers: Mapping[str, str] | None = None,
    timeout_ms: int = 30_000,
) -> AgentHttpResponse:
    request = urllib.request.Request(
        url,
        data=body.encode("utf-8") if body is not None else None,
        headers=dict(headers or {}),
        method=method,
    )
    timeout = max(timeout_ms, 1) / 1000.0
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            payload = response.read().decode("utf-8")
            return AgentHttpResponse(status=response.status, body=payload)
    except urllib.error.HTTPError as exc:
        payload = exc.read().decode("utf-8", errors="replace")
        return AgentHttpResponse(status=exc.code, body=payload)
    except TimeoutError as exc:
        raise TimeoutError(f"agent request timed out: {url}") from exc


def _parse_json(response: AgentHttpResponse, action: str) -> dict[str, Any]:
    if response.status < 200 or response.status >= 300:
        raise _agent_error(response, action)
    try:
        parsed = json.loads(response.body)
    except json.JSONDecodeError as exc:
        raise AgentClientError(f"agent {action}: invalid JSON response", response.status) from exc
    if not isinstance(parsed, dict):
        raise AgentClientError(f"agent {action}: expected JSON object", response.status)
    return parsed


def _agent_error(response: AgentHttpResponse, action: str) -> AgentClientError:
    message = f"agent {action} failed ({response.status})"
    try:
        parsed = json.loads(response.body)
        if isinstance(parsed, dict) and parsed.get("error"):
            message = str(parsed["error"])
    except json.JSONDecodeError:
        pass
    return AgentClientError(message, response.status)


__all__ = [
    "AgentClient",
    "AgentClientError",
    "AgentCloseResult",
    "AgentHttpResponse",
    "AgentOpenResult",
    "DEFAULT_AGENT_URL",
    "is_explicit_agent_url",
    "probe_agent_health",
    "resolve_agent_base_url",
    "sync_agent_request",
    "try_create_agent_client",
]
