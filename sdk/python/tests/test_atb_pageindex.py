from __future__ import annotations

import json
import uuid
from types import SimpleNamespace
from unittest.mock import patch

import pytest

from atb import ATBAppendError, ATBPageIndexRetriever, PageIndexRetrievalError


def _successful_run() -> SimpleNamespace:
    return SimpleNamespace(returncode=0, stderr="", stdout="")


def _payload_from_call(mock_run) -> dict:
    cmd = mock_run.call_args.args[0]
    return json.loads(cmd[3])


def test_build_index_appends_rag_index_event() -> None:
    tree = {
        "node_id": "0001",
        "title": "Root",
        "start_index": 0,
        "end_index": 10,
        "nodes": [],
    }
    retriever = ATBPageIndexRetriever()

    with patch("atb.pageindex._build_pageindex_tree", return_value=tree), patch(
        "atb.pageindex.subprocess.run",
        return_value=_successful_run(),
    ) as mock_run:
        built_tree, index_id = retriever.build_index("report.pdf")

    assert built_tree == tree
    assert index_id
    cmd = mock_run.call_args.args[0]
    assert "atb.event.rag_index" in cmd
    payload = json.loads(cmd[3])
    assert payload["index_hash"]
    assert payload["node_count"] == 1
    assert "report.pdf" in payload["source_uri"]


def test_retrieve_appends_rag_retrieval_event() -> None:
    node = {
        "node_id": "0007",
        "title": "Financial Stability",
        "start_index": 22,
        "end_index": 28,
        "summary": "The Fed...",
    }
    tree = {"structure": [node]}
    retriever = ATBPageIndexRetriever()

    with patch("atb.pageindex._retrieve_pageindex_node", return_value=node), patch(
        "atb.pageindex.subprocess.run",
        return_value=_successful_run(),
    ) as mock_run:
        result = retriever.retrieve("margins query", tree, "idx-001", "report.pdf")

    assert result == node
    cmd = mock_run.call_args.args[0]
    assert "atb.event.rag_retrieval" in cmd
    payload = json.loads(cmd[3])
    assert payload["node_id"] == "0007"
    assert payload["page_start"] == 22
    assert payload["page_end"] == 28
    assert payload["index_id"] == "idx-001"
    assert payload["latency_ms"] >= 0


def test_build_index_atb_failure_raises() -> None:
    tree = {
        "node_id": "0001",
        "title": "Root",
        "start_index": 0,
        "end_index": 10,
        "nodes": [],
    }
    retriever = ATBPageIndexRetriever()

    with patch("atb.pageindex._build_pageindex_tree", return_value=tree), patch(
        "atb.pageindex.subprocess.run",
        return_value=SimpleNamespace(returncode=1, stderr="bundle not found", stdout=""),
    ):
        with pytest.raises(ATBAppendError, match="bundle not found"):
            retriever.build_index("report.pdf")


def test_retrieve_no_result_raises() -> None:
    retriever = ATBPageIndexRetriever()

    with patch("atb.pageindex._retrieve_pageindex_node", return_value=None):
        with pytest.raises(PageIndexRetrievalError):
            retriever.retrieve("margins query", {"structure": []}, "idx-001", "report.pdf")


def test_auto_generated_ids_are_valid_uuid4() -> None:
    tree = {
        "node_id": "0001",
        "title": "Root",
        "start_index": 0,
        "end_index": 10,
        "nodes": [],
    }
    node = {
        "node_id": "0007",
        "title": "Financial Stability",
        "start_index": 22,
        "end_index": 28,
        "summary": "The Fed...",
    }
    retriever = ATBPageIndexRetriever()

    with patch("atb.pageindex._build_pageindex_tree", return_value=tree), patch(
        "atb.pageindex._retrieve_pageindex_node",
        return_value=node,
    ), patch(
        "atb.pageindex.subprocess.run",
        return_value=_successful_run(),
    ) as mock_run:
        built_tree, index_id = retriever.build_index("report.pdf")
        retriever.retrieve("margins query", built_tree, index_id, "report.pdf")

    first_payload = json.loads(mock_run.call_args_list[0].args[0][3])
    second_payload = json.loads(mock_run.call_args_list[1].args[0][3])

    assert str(uuid.UUID(first_payload["index_id"])) == first_payload["index_id"]
    assert uuid.UUID(first_payload["index_id"]).version == 4
    assert str(uuid.UUID(second_payload["retrieval_id"])) == second_payload["retrieval_id"]
    assert uuid.UUID(second_payload["retrieval_id"]).version == 4
