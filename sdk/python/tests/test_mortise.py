from __future__ import annotations

import json
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any, Dict, Iterator, Tuple

import pytest

from atb import MortiseClient, MortiseError


class _Handler(BaseHTTPRequestHandler):
    authorization = ""

    def do_POST(self) -> None:
        type(self).authorization = self.headers.get("Authorization", "")
        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length)
        if self.path == "/ingest":
            self._json(
                201,
                {
                    "receipt_version": "custos.receipt.v1",
                    "receipt_id": "receipt-1",
                    "bundle_hash": "bundle-hash",
                    "attestation": {"algorithm": "ed25519"},
                    "received_bytes": len(body),
                },
            )
        elif self.path == "/verify/bundle":
            self._json(200, {"verified": True, "bundle_hash": "bundle-hash"})
        elif self.path == "/verify/receipt":
            parsed = json.loads(body)
            self._json(
                200,
                {
                    "verified": parsed["receipt_id"] == "receipt-1",
                    "receipt_id": parsed["receipt_id"],
                },
            )
        else:
            self._json(404, {"error": "not found"})

    def do_GET(self) -> None:
        type(self).authorization = self.headers.get("Authorization", "")
        if self.path == "/receipts/by-hash?bundle_hash=bundle-hash":
            self._json(200, {"bundle_hash": "bundle-hash", "count": 1, "receipts": []})
        else:
            self._json(404, {"error": "not found"})

    def log_message(self, _format: str, *_args: Any) -> None:
        return

    def _json(self, status: int, body: Dict[str, Any]) -> None:
        data = json.dumps(body).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)


@pytest.fixture
def mortise_server() -> Iterator[Tuple[str, type[_Handler]]]:
    server = ThreadingHTTPServer(("127.0.0.1", 0), _Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        host, port = server.server_address
        yield f"http://{host}:{port}", _Handler
    finally:
        server.shutdown()
        thread.join()
        server.server_close()


def test_mortise_client_flows(mortise_server: Tuple[str, type[_Handler]]) -> None:
    base_url, handler = mortise_server
    client = MortiseClient(base_url + "/", token="secret")
    receipt = client.ingest_bundle(b"bundle")
    assert receipt["receipt_id"] == "receipt-1"
    assert receipt["attestation"]["algorithm"] == "ed25519"
    assert handler.authorization == "Bearer secret"

    verified_bundle = client.verify_bundle(b"bundle")
    assert verified_bundle["verified"] is True
    verified_receipt = client.verify_receipt(receipt)
    assert verified_receipt == {"verified": True, "receipt_id": "receipt-1"}
    lookup = client.receipts_by_hash("bundle-hash")
    assert lookup["count"] == 1


@pytest.mark.parametrize(
    "endpoint",
    [
        "",
        "mortise.example",
        "ftp://mortise.example",
        "https://user:pass@mortise.example",
        "https://mortise.example?token=secret",
        "https://mortise.example/#fragment",
    ],
)
def test_mortise_client_rejects_unsafe_endpoints(endpoint: str) -> None:
    with pytest.raises(ValueError):
        MortiseClient(endpoint)


def test_mortise_client_surfaces_http_errors() -> None:
    class RejectHandler(_Handler):
        def do_POST(self) -> None:
            self._json(422, {"error": "invalid bundle"})

    server = ThreadingHTTPServer(("127.0.0.1", 0), RejectHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        host, port = server.server_address
        with pytest.raises(MortiseError, match="422"):
            MortiseClient(f"http://{host}:{port}").ingest_bundle(b"bad")
    finally:
        server.shutdown()
        thread.join()
        server.server_close()
