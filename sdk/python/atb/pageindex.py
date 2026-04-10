from __future__ import annotations

# PageIndex integration for ATB Python SDK.
# PageIndex is used under the MIT License.
# Copyright (c) 2025 Vectify AI
# https://github.com/VectifyAI/PageIndex
# Full licence text: see THIRD_PARTY_NOTICES at the repository root.
#
# Step 2 source notes:
# - pageindex.page_index.page_index(doc, model=None, toc_check_page_num=None,
#   max_page_num_each_node=None, max_token_num_each_node=None,
#   if_add_node_id=None, if_add_node_summary=None,
#   if_add_doc_description=None, if_add_node_text=None) is the synchronous
#   tree-build entrypoint in the self-hosted repo.
# - pageindex.retrieve exposes synchronous helpers named get_document(),
#   get_document_structure(), and get_page_content(). In the referenced files,
#   the self-hosted package does not expose a single direct
#   retrieve(query) -> node function.
# - examples/agentic_vectorless_rag_demo.py performs retrieval by reasoning
#   over PageIndexClient.get_document_structure() plus get_page_content().
# - Returned node fields in the generated PageIndex structure are title,
#   node_id, start_index, end_index, summary, and optional nodes.
# - The public build API is synchronous, although page_index() wraps async
#   internals with asyncio.run().
# - PageIndex initialisation requires an OPENAI_API_KEY-compatible model
#   environment for the underlying LLM calls.

import hashlib
import importlib
import json
import re
import subprocess
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Iterator


class ATBAppendError(RuntimeError):
    """Raised when `atb append` exits non-zero."""


class PageIndexRetrievalError(RuntimeError):
    """Raised when PageIndex returns no matching node for the query."""


class ATBPageIndexRetriever:
    """
    Wraps PageIndex tree indexing and reasoning-based retrieval with
    automatic ATB audit event recording.

    Every call to build_index() appends an atb.event.rag_index record
    to the current ATB bundle.

    Every call to retrieve() appends an atb.event.rag_retrieval record
    to the current ATB bundle.

    Both events are appended via the ATB CLI binary (`atb append`).
    The caller is responsible for ensuring `atb init` has been run
    before using this class.

    PageIndex used under MIT License (Copyright 2025 Vectify AI).
    """

    def __init__(
        self,
        model: str = "gpt-4o-2024-11-20",
        atb_cli: str = "atb",
        actor_id: str | None = None,
        workspace_id: str | None = None,
    ) -> None:
        self.model = model
        self.atb_cli = atb_cli
        self.actor_id = actor_id
        self.workspace_id = workspace_id

    def build_index(
        self,
        source_path: str,
        index_id: str | None = None,
    ) -> tuple[dict[str, Any], str]:
        """
        Build a PageIndex tree for the document at source_path.

        Returns (tree, index_id) where tree is the full PageIndex tree
        dict and index_id is the identifier used in the ATB event
        (auto-generated UUID4 if not provided by caller).

        Appends atb.event.rag_index to the current bundle before
        returning.

        Raises:
            ATBAppendError: if `atb append` exits non-zero.
        """

        resolved_index_id = index_id or str(uuid.uuid4())
        tree = _build_pageindex_tree(source_path, self.model)
        index_hash = hashlib.sha256(
            json.dumps(tree, sort_keys=True).encode()
        ).hexdigest()
        payload = {
            "index_id": resolved_index_id,
            "source_uri": _to_source_uri(source_path),
            "index_hash": index_hash,
            "node_count": self._count_nodes(tree),
            "indexed_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        }
        self._atb_append("atb.event.rag_index", payload)
        return tree, resolved_index_id

    def retrieve(
        self,
        query: str,
        index: dict[str, Any],
        index_id: str,
        source_uri: str,
        retrieval_id: str | None = None,
    ) -> dict[str, Any]:
        """
        Perform reasoning-based tree retrieval over the given index.

        Returns the matched PageIndex node dict. Appends
        atb.event.rag_retrieval to the current bundle before returning.

        Raises:
            PageIndexRetrievalError: if PageIndex returns no result.
            ATBAppendError: if `atb append` exits non-zero.
        """

        resolved_retrieval_id = retrieval_id or str(uuid.uuid4())
        start = time.perf_counter()
        node = _retrieve_pageindex_node(query, index)
        latency_ms = int((time.perf_counter() - start) * 1000)
        if not node:
            raise PageIndexRetrievalError(
                f"PageIndex returned no matching node for query: {query}"
            )

        payload = {
            "retrieval_id": resolved_retrieval_id,
            "index_id": index_id,
            "source_uri": source_uri,
            "query": query,
            "node_id": node["node_id"],
            "node_title": node["title"],
            "page_start": node["start_index"],
            "page_end": node["end_index"],
            "latency_ms": latency_ms,
        }
        if node.get("summary"):
            payload["node_summary"] = node["summary"]
        self._atb_append("atb.event.rag_retrieval", payload)
        return node

    def _atb_append(self, event_type: str, data: dict[str, Any]) -> None:
        cmd = [self.atb_cli, "append", event_type, json.dumps(data), "--format", "json"]
        if self.actor_id:
            cmd += ["--actor-id", self.actor_id]
        if self.workspace_id:
            cmd += ["--workspace-id", self.workspace_id]
        result = subprocess.run(cmd, capture_output=True, text=True)
        if result.returncode != 0:
            raise ATBAppendError(
                f"atb append {event_type} failed (exit {result.returncode}): "
                f"{result.stderr.strip()}"
            )

    def _count_nodes(self, tree: dict[str, Any]) -> int:
        if "structure" in tree and isinstance(tree["structure"], list):
            return sum(
                self._count_nodes(child)
                for child in tree["structure"]
                if isinstance(child, dict)
            )

        count = 1 if "node_id" in tree else 0
        for child in tree.get("nodes", []):
            if isinstance(child, dict):
                count += self._count_nodes(child)
        return count


def _build_pageindex_tree(source_path: str, model: str) -> dict[str, Any]:
    page_index_module = importlib.import_module("pageindex.page_index")
    page_index = getattr(page_index_module, "page_index")
    return page_index(
        doc=source_path,
        model=model,
        if_add_node_id="yes",
        if_add_node_summary="yes",
        if_add_doc_description="yes",
    )


def _retrieve_pageindex_node(query: str, index: dict[str, Any]) -> dict[str, Any] | None:
    # The referenced self-hosted repo exposes tree-building plus agent tool
    # primitives, but not a direct retrieve(query) -> node helper. Until
    # upstream exposes one, perform deterministic in-process tree search over
    # official PageIndex node metadata.
    query_terms = _tokenise(query)
    best_node: dict[str, Any] | None = None
    best_score: tuple[int, int, int] | None = None

    for node in _iter_nodes(index):
        score = _score_node(query, query_terms, node)
        if score is None:
            continue
        if best_score is None or score > best_score:
            best_score = score
            best_node = node

    return best_node


def _iter_nodes(tree: dict[str, Any]) -> Iterator[dict[str, Any]]:
    if isinstance(tree.get("structure"), list):
        for child in tree["structure"]:
            if isinstance(child, dict):
                yield from _iter_nodes(child)
        return

    if "node_id" in tree:
        yield tree

    for child in tree.get("nodes", []):
        if isinstance(child, dict):
            yield from _iter_nodes(child)


def _score_node(
    query: str,
    query_terms: set[str],
    node: dict[str, Any],
) -> tuple[int, int, int] | None:
    title = str(node.get("title", ""))
    summary = str(node.get("summary", ""))
    haystack = f"{title} {summary}".strip().lower()
    if not haystack:
        return None

    exact_phrase = 1 if query.strip().lower() in haystack else 0
    title_terms = _tokenise(title)
    summary_terms = _tokenise(summary)
    title_overlap = len(query_terms & title_terms)
    summary_overlap = len(query_terms & summary_terms)
    score = (exact_phrase * 100) + (title_overlap * 10) + (summary_overlap * 4)
    if score == 0:
        return None

    start_index = int(node.get("start_index", 0) or 0)
    end_index = int(node.get("end_index", start_index) or start_index)
    span = max(end_index - start_index, 0)
    return (score, -span, -start_index)


def _tokenise(text: str) -> set[str]:
    return {
        token
        for token in re.findall(r"[a-z0-9]+", text.lower())
        if token
    }


def _to_source_uri(source_path: str) -> str:
    if "://" in source_path:
        return source_path
    return Path(source_path).expanduser().resolve().as_uri()
