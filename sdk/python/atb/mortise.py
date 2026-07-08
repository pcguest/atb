"""Optional client for the separate Mortise custodian-of-record service."""

from __future__ import annotations

import json
from typing import Any, Dict, Mapping
from urllib.error import HTTPError, URLError
from urllib.parse import quote, urlsplit
from urllib.request import HTTPRedirectHandler, Request, build_opener

from atb.exceptions import ATBError

_MAX_RESPONSE_BYTES = 1 << 20
_RECEIPT_VERSION = "custos.receipt.v1"


class MortiseError(ATBError):
    """Raised when Mortise rejects a request or returns an invalid response."""


class _NoRedirects(HTTPRedirectHandler):
    def redirect_request(
        self,
        request: Request,
        fp: Any,
        code: int,
        message: str,
        headers: Any,
        new_url: str,
    ) -> None:
        return None


class MortiseClient:
    """Authenticated client for Mortise custody and verification APIs."""

    def __init__(
        self,
        endpoint: str,
        token: str = "",
        timeout: float = 30.0,
    ) -> None:
        endpoint = endpoint.strip()
        parsed = urlsplit(endpoint)
        if (
            parsed.scheme not in {"http", "https"}
            or not parsed.netloc
            or parsed.username is not None
            or parsed.password is not None
            or parsed.query
            or parsed.fragment
        ):
            raise ValueError("invalid Mortise endpoint")
        if timeout <= 0:
            raise ValueError("timeout must be positive")
        self._endpoint = endpoint.rstrip("/")
        self._token = token.strip()
        self._timeout = timeout
        self._opener = build_opener(_NoRedirects())

    def ingest_bundle(self, bundle: bytes) -> Dict[str, Any]:
        """Lodge a complete ATB bundle and return its signed custody receipt."""
        if not bundle:
            raise ValueError("bundle must not be empty")
        receipt = self._request("POST", "/ingest", bundle, "application/octet-stream")
        if (
            receipt.get("receipt_version") != _RECEIPT_VERSION
            or not receipt.get("receipt_id")
            or not receipt.get("bundle_hash")
        ):
            raise MortiseError("Mortise returned an invalid custody receipt")
        return receipt

    def verify_bundle(self, bundle: bytes) -> Dict[str, Any]:
        """Verify a bundle through Mortise without persisting it."""
        if not bundle:
            raise ValueError("bundle must not be empty")
        return self._request(
            "POST", "/verify/bundle", bundle, "application/octet-stream"
        )

    def verify_receipt(self, receipt: Mapping[str, Any]) -> Dict[str, Any]:
        """Verify a receipt against the deployment's pinned custody key."""
        body = json.dumps(dict(receipt), separators=(",", ":")).encode()
        return self._request("POST", "/verify/receipt", body, "application/json")

    def receipts_by_hash(self, bundle_hash: str) -> Dict[str, Any]:
        """Return this organisation's receipts for an ATB chain-head hash."""
        bundle_hash = bundle_hash.strip()
        if not bundle_hash:
            raise ValueError("bundle_hash must not be empty")
        path = "/receipts/by-hash?bundle_hash=" + quote(bundle_hash, safe="")
        return self._request("GET", path)

    def _request(
        self,
        method: str,
        path: str,
        body: bytes | None = None,
        content_type: str = "",
    ) -> Dict[str, Any]:
        headers = {"Accept": "application/json"}
        if content_type:
            headers["Content-Type"] = content_type
        if self._token:
            headers["Authorization"] = "Bearer " + self._token
        request = Request(
            self._endpoint + path,
            data=body,
            headers=headers,
            method=method,
        )
        try:
            with self._opener.open(request, timeout=self._timeout) as response:
                data = response.read(_MAX_RESPONSE_BYTES + 1)
        except HTTPError as exc:
            detail = exc.read(_MAX_RESPONSE_BYTES).decode("utf-8", errors="replace")
            raise MortiseError(
                f"Mortise returned HTTP {exc.code}: {detail.strip()}"
            ) from exc
        except URLError as exc:
            raise MortiseError(f"Mortise request failed: {exc.reason}") from exc
        if len(data) > _MAX_RESPONSE_BYTES:
            raise MortiseError("Mortise response exceeds 1 MiB")
        try:
            parsed = json.loads(data)
        except (UnicodeDecodeError, json.JSONDecodeError) as exc:
            raise MortiseError("Mortise returned invalid JSON") from exc
        if not isinstance(parsed, dict):
            raise MortiseError("Mortise response must be a JSON object")
        return parsed
