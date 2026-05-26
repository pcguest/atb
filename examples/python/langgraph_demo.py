#!/usr/bin/env python3
"""Minimal LangGraph + ATB reference integration.

Run from repository root (optional extra):
    pip install -e 'sdk/python[langgraph]'
    python examples/python/langgraph_demo.py

Requires `atb` on PATH for profile verification.
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import TypedDict

from atb import Bundle


class GraphState(TypedDict):
    query: str
    context: str
    answer: str


def atb_bin() -> str:
    return os.environ.get("ATB_BIN", shutil.which("atb") or "atb")


def build_graph(bundle: Bundle):
    try:
        from langgraph.graph import END, START, StateGraph
    except ModuleNotFoundError as exc:
        raise SystemExit(
            "LangGraph is not installed. Run: pip install -e 'sdk/python[langgraph]'"
        ) from exc

    def retrieve(state: GraphState) -> GraphState:
        bundle.append(
            "ai.request.received",
            {
                "request_id": "req-langgraph-001",
                "actor_id_hash": "sha256:demo-user",
                "purpose_tag": "rag_answer",
            },
        )
        bundle.append(
            "ai.retrieval.executed",
            {"retrieval_id": "ret-001", "source": "stub_kb", "hit_count": 1},
        )
        return {**state, "context": "ATB records tamper-evident AI workflow evidence."}

    def generate(state: GraphState) -> GraphState:
        bundle.append(
            "ai.model.invoked",
            {
                "model_provider": "stub",
                "model_id": "demo-graph",
                "model_parameters_digest": "sha256:params-demo",
                "prompt_digest": "sha256:prompt-demo",
            },
        )
        answer = f"Stub answer for: {state['query']} (context bytes={len(state['context'])})"
        bundle.append(
            "ai.model.output",
            {"output_digest": "sha256:output-demo", "output_format": "text/plain"},
        )
        return {**state, "answer": answer}

    def tool_exec(state: GraphState) -> GraphState:
        bundle.append(
            "ai.tool.exec",
            {
                "request_id": "req-langgraph-001",
                "tool_name": "format_response",
                "context": {"tool_name": "format_response"},
            },
        )
        bundle.append(
            "ai.response.sent",
            {
                "request_id": "req-langgraph-001",
                "output_digest": "sha256:response-demo",
                "output_format": "text/plain",
            },
        )
        return state

    graph = StateGraph(GraphState)
    graph.add_node("retrieve", retrieve)
    graph.add_node("generate", generate)
    graph.add_node("tool_exec", tool_exec)
    graph.add_edge(START, "retrieve")
    graph.add_edge("retrieve", "generate")
    graph.add_edge("generate", "tool_exec")
    graph.add_edge("tool_exec", END)
    return graph.compile()


def main() -> int:
    work = Path(tempfile.mkdtemp(prefix="atb-langgraph-demo-"))
    bundle_path = work / "bundle.atb"

    bundle = Bundle()
    graph = build_graph(bundle)
    result = graph.invoke({"query": "What does ATB prove?", "context": "", "answer": ""})

    bundle.save(str(bundle_path))
    print("Graph result:", result["answer"])

    proc = subprocess.run(
        [
            atb_bin(),
            "verify",
            "--bundle",
            str(bundle_path),
            "--profile",
            "atb.profile.rag_answer",
            "--format",
            "json",
        ],
        check=True,
        capture_output=True,
        text=True,
    )
    report = json.loads(proc.stdout)
    print(
        json.dumps(
            {
                "pass": report["pass"],
                "cas_grade": report["cas_grade"],
                "profile_id": report["profile_id"],
            },
            indent=2,
        )
    )
    print(f"bundle: {bundle_path}")
    return 0 if report.get("pass") else 1


if __name__ == "__main__":
    raise SystemExit(main())
