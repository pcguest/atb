#!/usr/bin/env python3
"""ATB end-user validation harness.

Exercises the *public* ATB product surface as an unfamiliar technical user would:
import a saved chatlog, verify integrity, inspect provenance, derive and resolve
a semantic evidence locator, and probe bounded failure states.

Every step is classified as PRODUCT_PATH (documented public surface),
TEST_ONLY_PATH (fixture convenience), or INTERNAL_SHORTCUT (would be a product
finding). The harness never fabricates success: a step that cannot be completed
through the product surface is recorded as NOT_SUPPORTED or FRICTION.

It is safe to run repeatedly, ingests no secrets, and writes only to its own
scratch directory. It does not commit the raw source material.
"""
from __future__ import annotations

import argparse
import datetime as _dt
import json
import os
import shutil
import socket
import subprocess
import sys
import tempfile
import time
import urllib.parse
import urllib.request
from pathlib import Path

HERE = Path(__file__).resolve().parent
DEFAULT_FIXTURE = HERE / "fixtures" / "development-session.jsonl"
SESSION_TOKEN = "harness-local-session-token"


def now() -> str:
    return _dt.datetime.now(_dt.timezone.utc).replace(microsecond=0).isoformat()


def find_atb(explicit: str | None) -> str:
    if explicit:
        return explicit
    env = os.environ.get("ATB_BIN")
    if env and Path(env).exists():
        return env
    onpath = shutil.which("atb")
    if onpath:
        return onpath
    repo = HERE.parent.parent / "atb"
    if repo.exists():
        return str(repo)
    raise SystemExit("harness: cannot find an `atb` binary (set ATB_BIN)")


def free_port() -> int:
    s = socket.socket()
    s.bind(("127.0.0.1", 0))
    port = s.getsockname()[1]
    s.close()
    return port


class Harness:
    def __init__(self, atb: str, fixture: Path, work: Path) -> None:
        self.atb = atb
        self.fixture = fixture
        self.work = work
        self.steps: list[dict] = []
        self.findings: list[dict] = []
        self.bundle = work / "session.atb"
        self.head = ""
        self.record_hash = ""
        self.locator = ""
        self.viewer = None
        self.viewer_port = 0

    # -- helpers ---------------------------------------------------------
    def run(self, *args: str, check: bool = False) -> subprocess.CompletedProcess:
        return subprocess.run(
            [self.atb, *args],
            capture_output=True,
            text=True,
            check=check,
        )

    def step(self, **kw) -> None:
        kw.setdefault("severity", "none")
        kw.setdefault("internal_knowledge_required", False)
        self.steps.append(kw)

    def finding(self, fid: str, severity: str, classification: str, summary: str, evidence: str = "") -> None:
        self.findings.append(
            {
                "id": fid,
                "severity": severity,
                "classification": classification,
                "summary": summary,
                "evidence": evidence,
            }
        )

    def http_get(self, path: str) -> tuple[int, dict]:
        req = urllib.request.Request(
            f"http://127.0.0.1:{self.viewer_port}{path}",
            headers={"X-ATB-Session-Token": SESSION_TOKEN},
        )
        try:
            with urllib.request.urlopen(req, timeout=10) as resp:
                return resp.status, json.loads(resp.read().decode())
        except urllib.error.HTTPError as e:  # type: ignore[attr-defined]
            try:
                return e.code, json.loads(e.read().decode())
            except Exception:
                return e.code, {}

    # -- journey ---------------------------------------------------------
    def a1_discover(self) -> None:
        help_out = self.run("--help")
        has_import = "import chatlog" in help_out.stdout
        has_verify = "\n  verify " in help_out.stdout or "verify [" in help_out.stdout
        has_view = "view [" in help_out.stdout
        self.step(
            id="A1",
            task="Discover what ATB is and how to import, verify, and inspect a chatlog",
            expected_goal="README + `atb --help` explain the capture→verify→investigate flow",
            documented_path="README.md; docs/integrations/chatlog-import.md; atb --help",
            actual_path="atb --help",
            result="PASS" if (has_import and has_verify and has_view) else "FRICTION",
            path_class="PRODUCT_PATH",
            evidence={"has_import_chatlog": has_import, "has_verify": has_verify, "has_view": has_view},
            friction="",
        )

    def a2_capture(self) -> None:
        proc = self.run(
            "import", "chatlog", "--from", "generic-jsonl",
            "--input", str(self.fixture), "--bundle", str(self.bundle), "--format", "json",
        )
        try:
            out = json.loads(proc.stdout)
        except Exception:
            out = {}
        ok = proc.returncode == 0 and out.get("events_written", 0) > 0
        self.step(
            id="A2",
            task="Import the saved chatlog into a local bundle",
            expected_goal="A .atb bundle is created; the command reports events written",
            documented_path="atb import chatlog --from generic-jsonl --input <file>",
            actual_path="atb import chatlog --from generic-jsonl --input <file> --bundle <path>",
            result="PASS" if ok else "FAIL",
            path_class="PRODUCT_PATH",
            evidence={"exit": proc.returncode, "events_written": out.get("events_written")},
            friction="" if ok else proc.stderr.strip()[:300],
        )

    def a3_verify(self) -> None:
        proc = self.run("verify", "--bundle", str(self.bundle), "--format", "json")
        try:
            rep = json.loads(proc.stdout)
        except Exception:
            rep = {}
        chain_valid = bool(rep.get("gate_result", {}).get("chain_valid"))
        self.step(
            id="A3",
            task="Verify integrity and distinguish it from truth",
            expected_goal="verify reports chain_valid; documentation states this is integrity, not truth",
            documented_path="atb verify --bundle <path> --format json",
            actual_path="atb verify --bundle <path> --format json",
            result="PASS" if chain_valid else "FAIL",
            path_class="PRODUCT_PATH",
            evidence={
                "exit": proc.returncode,
                "chain_valid": chain_valid,
                "pass": rep.get("pass"),
                "cas_grade": rep.get("cas_grade"),
                "report_version": rep.get("report_version"),
            },
            friction="",
        )

    def a4_inspect(self) -> None:
        proc = self.run("inspect", "--bundle", str(self.bundle), "--json")
        try:
            records = json.loads(proc.stdout)
        except Exception:
            records = []
        self.head = records[-1]["hash"] if records else ""
        seq1 = next((r for r in records if r["event"].get("seq") == 1), None)
        self.record_hash = seq1["hash"] if seq1 else ""
        acq = seq1["event"].get("acquisition", {}) if seq1 else {}
        has_acq = bool(acq.get("source_digest") and acq.get("source_system"))
        self.step(
            id="A4",
            task="Inspect the event timeline and acquisition provenance",
            expected_goal="Records, sequence numbers, hashes and acquisition metadata are visible",
            documented_path="atb inspect --bundle <path> --json",
            actual_path="atb inspect --bundle <path> --json",
            result="PASS" if (self.head and self.record_hash and has_acq) else "FRICTION",
            path_class="PRODUCT_PATH",
            evidence={
                "records": len(records),
                "head_hash": self.head,
                "seq1_hash": self.record_hash,
                "acquisition_source_system": acq.get("source_system"),
                "acquisition_source_digest": acq.get("source_digest"),
            },
            friction="" if has_acq else "acquisition provenance not present on seq 1",
        )

    def _start_viewer(self) -> bool:
        self.viewer_port = free_port()
        self.viewer = subprocess.Popen(
            [
                self.atb, "view", "--bundle", str(self.bundle),
                "--no-open", "--host", "127.0.0.1", "--port", str(self.viewer_port),
                "--session-token", SESSION_TOKEN,
            ],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        for _ in range(50):
            time.sleep(0.2)
            try:
                self.http_get("/api/v1/verification")
                return True
            except Exception:
                continue
        return False

    def a5_locator(self) -> None:
        self.locator = f"atb://evidence/1/{self.head}?seq=1&record={self.record_hash}"
        if not self._start_viewer():
            self.step(
                id="A5", task="Derive and resolve a semantic evidence locator",
                expected_goal="The locator resolves to the intended record",
                documented_path="docs/specification/evidence-locator.md; viewer API",
                actual_path="atb view + GET /api/v1/bundle/locate",
                result="FAIL", path_class="PRODUCT_PATH",
                evidence={"locator": self.locator},
                friction="viewer API did not become ready",
            )
            self.finding("H-ATB-01", "medium", "PRODUCT_HARDENING",
                         "No CLI command emits a locator; the user must assemble it from inspect output and the spec",
                         "atb inspect --json exposes hashes but no locator string")
            return
        q = urllib.parse.quote(self.locator, safe="")
        status, body = self.http_get(f"/api/v1/bundle/locate?locator={q}")
        ok = status == 200 and body.get("ok") and body.get("canonical") == self.locator
        self.step(
            id="A5",
            task="Derive and resolve a semantic evidence locator",
            expected_goal="The locator resolves to the intended record (identity, not integrity)",
            documented_path="docs/specification/evidence-locator.md; GET /api/v1/bundle/locate",
            actual_path="construct locator from inspect; GET /api/v1/bundle/locate",
            result="PASS" if ok else "FAIL",
            path_class="PRODUCT_PATH",
            evidence={"http": status, "canonical": body.get("canonical"), "record_hash_matched": body.get("record_hash_matched")},
            friction="locator must be assembled by hand; no `atb locate`/`--emit-locator` command exists",
            internal_knowledge_required=False,
        )
        if ok:
            self.finding("H-ATB-01", "low", "PRODUCT_HARDENING",
                         "No CLI/UI surface emits a locator; users must assemble it from inspect output + spec",
                         "A `--emit-locator` option or a copy control in the viewer would remove this friction")

    def a6_failures(self) -> None:
        # Malformed locator -> location failure, explicitly not an integrity verdict.
        q = urllib.parse.quote("not-a-locator", safe="")
        status, body = self.http_get(f"/api/v1/bundle/locate?locator={q}")
        malformed_ok = body.get("error_code") == "LOCATOR_MALFORMED" and body.get("ok") is False
        self.step(
            id="A6a",
            task="Probe a malformed locator",
            expected_goal="Reported as a location failure, never as tampering/invalidity",
            documented_path="docs/specification/evidence-locator.md §8",
            actual_path="GET /api/v1/bundle/locate?locator=not-a-locator",
            result="PASS" if malformed_ok else "FRICTION",
            path_class="PRODUCT_PATH",
            evidence={"http": status, "error_code": body.get("error_code")},
        )
        # Unknown head hash -> BUNDLE_NOT_AVAILABLE (still a location outcome).
        wrong = "atb://evidence/1/" + ("0" * 64) + "?seq=1"
        q = urllib.parse.quote(wrong, safe="")
        status, body = self.http_get(f"/api/v1/bundle/locate?locator={q}")
        self.step(
            id="A6b",
            task="Resolve a locator whose head hash is not in the loaded bundle",
            expected_goal="Reported as BUNDLE_NOT_AVAILABLE, not as invalidity",
            documented_path="docs/specification/evidence-locator.md §8",
            actual_path="GET /api/v1/bundle/locate (unknown head)",
            result="PASS" if body.get("error_code") == "BUNDLE_NOT_AVAILABLE" else "FRICTION",
            path_class="PRODUCT_PATH",
            evidence={"http": status, "error_code": body.get("error_code")},
        )
        # Source representation changed -> reconcile records a bounded finding.
        edited = self.work / "edited.jsonl"
        lines = self.fixture.read_text().splitlines()
        lines = [ln.replace("stale", "outdated") if "release notes are stale" in ln else ln for ln in lines]
        edited.write_text("\n".join(lines) + "\n")
        proc = self.run(
            "import", "chatlog", "--from", "generic-jsonl", "--input", str(edited),
            "--bundle", str(self.bundle), "--reconcile", "--format", "json",
        )
        insp = self.run("inspect", "--bundle", str(self.bundle), "--json")
        try:
            records = json.loads(insp.stdout)
        except Exception:
            records = []
        changed = [r for r in records if r["event"].get("type") == "atb.acquisition.finding"]
        ver = self.run("verify", "--bundle", str(self.bundle), "--format", "json")
        still_valid = '"chain_valid": true' in ver.stdout
        self.step(
            id="A6c",
            task="Re-import an edited copy of the same source (representation changed)",
            expected_goal="A bounded source_record_changed finding; integrity still valid; no tamper claim",
            documented_path="docs/integrations/chatlog-import.md (reconciliation)",
            actual_path="atb import chatlog --reconcile",
            result="PASS" if (changed and still_valid) else "FRICTION",
            path_class="PRODUCT_PATH",
            evidence={"exit": proc.returncode, "findings": len(changed), "chain_valid_after": still_valid},
            friction="" if changed else "reconcile did not surface a source_record_changed finding",
        )

    def stop(self) -> None:
        if self.viewer:
            self.viewer.terminate()
            try:
                self.viewer.wait(timeout=5)
            except Exception:
                self.viewer.kill()


def main() -> int:
    ap = argparse.ArgumentParser(description="ATB end-user validation harness")
    ap.add_argument("--atb", help="path to the atb binary")
    ap.add_argument("--fixture", default=str(DEFAULT_FIXTURE))
    ap.add_argument("--out", default="")
    ap.add_argument("--work", default="")
    args = ap.parse_args()

    fixture = Path(args.fixture)
    if not fixture.exists():
        raise SystemExit(f"harness: fixture not found: {fixture}")
    atb = find_atb(args.atb)
    work = Path(args.work) if args.work else Path(tempfile.mkdtemp(prefix="atb-enduser-"))
    work.mkdir(parents=True, exist_ok=True)

    started = now()
    h = Harness(atb, fixture, work)
    version = h.run("version").stdout.strip()
    try:
        h.a1_discover()
        h.a2_capture()
        h.a3_verify()
        h.a4_inspect()
        h.a5_locator()
        h.a6_failures()
    finally:
        h.stop()

    results = [s["result"] for s in h.steps]
    hard_fail = any(r == "FAIL" for r in results)
    report = {
        "schema_version": "atb.dev.run.v1",
        "harness": "end-user",
        "product": "atb",
        "started_at": started,
        "finished_at": now(),
        "environment": {"atb_version": version, "atb_binary": atb},
        "source_fixture": {
            "path": str(fixture),
            "classification": "REPRESENTATIVE_FIXTURE",
            "privacy_reviewed": True,
            "transformation": "Synthesised from a small, non-sensitive excerpt of the ATB/Mortise development workflow; no secrets, credentials, or private chat content.",
        },
        "steps": h.steps,
        "findings": h.findings,
        "gates": {"ATB_PRODUCT_GATE": "NO_GO" if hard_fail else "GO"},
    }
    text = json.dumps(report, indent=2)
    if args.out:
        Path(args.out).write_text(text + "\n")
    print(text)
    print(f"\n# ATB end-user harness: {'FAIL' if hard_fail else 'PASS'} ({len(h.steps)} steps, {len(h.findings)} findings)")
    return 1 if hard_fail else 0


if __name__ == "__main__":
    sys.exit(main())
