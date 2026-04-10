from __future__ import annotations

import argparse
import contextlib
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Any, Iterator


REPO_ROOT = Path(__file__).resolve().parent.parent
SDK_ROOT = REPO_ROOT / "sdk" / "python"
if str(SDK_ROOT) not in sys.path:
    sys.path.insert(0, str(SDK_ROOT))

from atb import ATBAppendError, ATBPageIndexRetriever, PageIndexRetrievalError


DEFAULT_DOC = "demo/sample.md"
DEFAULT_MODEL = "gpt-4o-2024-11-20"
DEFAULT_WORKDIR = "./run.atb"


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="ATB x PageIndex end-to-end demo.",
    )
    parser.add_argument(
        "--doc",
        default=DEFAULT_DOC,
        help="Path to a Markdown or PDF file to index.",
    )
    parser.add_argument(
        "--query",
        default="What is ATB?",
        help="Retrieval query string.",
    )
    parser.add_argument(
        "--model",
        default=DEFAULT_MODEL,
        help="LLM model to use for PageIndex tree construction.",
    )
    parser.add_argument(
        "--workdir",
        default=DEFAULT_WORKDIR,
        help="Directory that will contain the ATB bundle.",
    )
    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()

    if not os.environ.get("OPENAI_API_KEY"):
        print("Error: OPENAI_API_KEY environment variable is not set.")
        print("Export your key: export OPENAI_API_KEY=sk-...")
        return 1

    doc_path = Path(args.doc).expanduser().resolve()
    if not doc_path.exists():
        print(f"Error: document not found: {doc_path}")
        return 1

    try:
        bundle_context = prepare_bundle_context(Path(args.workdir))
    except RuntimeError as exc:
        print(f"Error: {exc}")
        return 1

    bundle_file = bundle_context["bundle_dir"] / "bundle.atb"
    doc_display = display_path(doc_path)
    bundle_display = display_path(bundle_file)

    print("ATB x PageIndex - End-to-End Demo")
    print("=================================")
    print(f"Document : {doc_display}")
    print(f"Query    : {args.query}")
    print()

    try:
        print("[1/4] Initialising ATB bundle...")
        run_command(["atb", "init", "--format", "json"], cwd=bundle_context["cwd"])
        print("      \u2713 Bundle ready")
        print()

        print("[2/4] Building PageIndex tree...")
        retriever = ATBPageIndexRetriever(model=args.model, atb_cli="atb")
        with temporary_cwd(bundle_context["cwd"]):
            tree, index_id = retriever.build_index(
                source_path=str(doc_path),
                index_id=None,
            )
        index_hash = hashlib.sha256(
            json.dumps(tree, sort_keys=True).encode("utf-8")
        ).hexdigest()
        index_record = read_last_record(bundle_file)
        index_sequence = event_sequence(index_record)
        print(f"      \u2713 {count_nodes(tree)} nodes indexed (SHA-256: {index_hash[:8]}...)")
        print(f"      \u2713 atb.event.rag_index recorded (sequence {index_sequence})")
        print()

        print("[3/4] Retrieving answer node...")
        with temporary_cwd(bundle_context["cwd"]):
            result = retriever.retrieve(
                query=args.query,
                index=tree,
                index_id=index_id,
                source_uri=str(doc_path),
                retrieval_id=None,
            )
        retrieval_record = read_last_record(bundle_file)
        retrieval_sequence = event_sequence(retrieval_record)
        title = str(result.get("title", "Untitled node"))
        start_page = result.get("start_index", "?")
        end_page = result.get("end_index", start_page)
        print(f'      \u2713 Node: "{title}" (pages {start_page}-{end_page})')
        print(f"      \u2713 atb.event.rag_retrieval recorded (sequence {retrieval_sequence})")
        print()

        print("[4/4] Verifying bundle integrity...")
        verify = run_command(["atb", "verify", "--json"], cwd=bundle_context["cwd"])
        verify_json = json.loads(verify.stdout)
        integrity = verify_json.get("integrity", {})
        if not integrity.get("chain_valid"):
            print("Error: atb verify reported an invalid chain.")
            print(verify.stdout.strip() or "No verification output returned.")
            return 1
        records = read_bundle_records(bundle_file)
        head_hash = str(records[-1].get("hash", "")) if records else ""
        print(f"      \u2713 Chain valid - {len(records)} events, head: {head_hash[:16]}...")
        print()

        answer = str(result.get("summary") or result.get("title") or "No summary available.")
        print("***")
        print("Answer node summary:")
        print(f"  {answer}")
        print()
        print(f"Bundle location: {bundle_display}")
        print(f"Run `atb inspect --bundle {bundle_display}` to see all recorded events.")
        return 0
    except ModuleNotFoundError as exc:
        if exc.name and exc.name.startswith("pageindex"):
            print("Error: PageIndex is not installed.")
            print("Install it with: pip install pageindex")
            return 1
        print(f"Error: missing Python dependency: {exc}")
        return 1
    except PageIndexRetrievalError:
        print("Error: PageIndex returned no matching node for that query.")
        return 1
    except ATBAppendError as exc:
        print(f"Error: {exc}")
        return 1
    except RuntimeError as exc:
        print(f"Error: {exc}")
        return 1
    finally:
        cleanup = bundle_context.get("cleanup_root")
        if cleanup is not None:
            shutil.rmtree(cleanup, ignore_errors=True)


def prepare_bundle_context(workdir: Path) -> dict[str, Path | None]:
    bundle_dir = workdir.expanduser().resolve()
    if bundle_dir.exists() and not bundle_dir.is_dir():
        raise RuntimeError(f"workdir is not a directory: {bundle_dir}")

    bundle_dir.mkdir(parents=True, exist_ok=True)
    if bundle_dir.name == "run.atb":
        return {"bundle_dir": bundle_dir, "cwd": bundle_dir.parent, "cleanup_root": None}

    helper_root = Path(tempfile.mkdtemp(prefix="atb-demo-", dir=str(bundle_dir.parent)))
    link_path = helper_root / "run.atb"
    try:
        link_path.symlink_to(bundle_dir, target_is_directory=True)
    except OSError as exc:
        raise RuntimeError(
            "unable to prepare a run.atb alias for the selected workdir"
        ) from exc
    return {"bundle_dir": bundle_dir, "cwd": helper_root, "cleanup_root": helper_root}


@contextlib.contextmanager
def temporary_cwd(path: Path) -> Iterator[None]:
    previous = Path.cwd()
    os.chdir(path)
    try:
        yield
    finally:
        os.chdir(previous)


def run_command(cmd: list[str], cwd: Path) -> subprocess.CompletedProcess[str]:
    try:
        result = subprocess.run(
            cmd,
            cwd=str(cwd),
            capture_output=True,
            text=True,
            check=False,
        )
    except FileNotFoundError as exc:
        raise RuntimeError(f"command failed: {' '.join(cmd)} ({exc})") from exc

    if result.returncode != 0:
        details = result.stderr.strip() or result.stdout.strip() or f"exit {result.returncode}"
        raise RuntimeError(f"command failed: {' '.join(cmd)}\n{details}")
    return result


def read_bundle_records(bundle_file: Path) -> list[dict[str, Any]]:
    if not bundle_file.exists():
        raise RuntimeError(f"bundle file not found: {bundle_file}")

    records: list[dict[str, Any]] = []
    with bundle_file.open("r", encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if not line:
                continue
            records.append(json.loads(line))
    return records


def read_last_record(bundle_file: Path) -> dict[str, Any]:
    records = read_bundle_records(bundle_file)
    if not records:
        raise RuntimeError(f"bundle file is empty: {bundle_file}")
    return records[-1]


def event_sequence(record: dict[str, Any]) -> int:
    event = record.get("event", {})
    sequence = event.get("seq")
    if not isinstance(sequence, int):
        raise RuntimeError("bundle record did not include an integer event sequence")
    return sequence


def count_nodes(tree: dict[str, Any]) -> int:
    if "structure" in tree and isinstance(tree["structure"], list):
        return sum(count_nodes(child) for child in tree["structure"] if isinstance(child, dict))

    count = 1 if "node_id" in tree else 0
    for child in tree.get("nodes", []):
        if isinstance(child, dict):
            count += count_nodes(child)
    return count


def display_path(path: Path) -> str:
    try:
        relative = path.relative_to(Path.cwd())
        rendered = relative.as_posix()
        if not rendered.startswith("."):
            return f"./{rendered}"
        return rendered
    except ValueError:
        return str(path)


if __name__ == "__main__":
    raise SystemExit(main())
