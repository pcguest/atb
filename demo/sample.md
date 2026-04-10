# ATB Overview

## What ATB Is

ATB, short for Agent Trace Bundle, is a tamper-evident audit trail for privacy-sensitive AI systems. It records the significant events that happen during an AI-assisted workflow and links those events together with hashes so the resulting bundle can be checked later for integrity. The goal is not to record every possible detail forever. The goal is to preserve the right evidence: what request arrived, what model or retrieval step ran, what tools were called, what controls were evaluated, and what output was produced.

## Why It Exists

Privacy-sensitive AI work often happens in regulated or high-trust environments where teams need more than logs. They need evidence that can travel with a review packet, survive handoff between teams, and still be validated months later. Plain application logs are easy to reorder, redact incorrectly, or lose across systems. ATB packages evidence into a portable bundle so that engineering, security, legal, and audit stakeholders can review the same chain of events and verify that it has not been silently altered.

## How The Hash Chain Works

An ATB bundle is stored as newline-delimited JSON records. Each record contains an event plus the cryptographic hash of that event. Every event also includes the previous record hash, which means the bundle forms a hash chain from the manifest at sequence zero to the current head. If any event payload, timestamp, sequence number, or ordering is changed after the fact, the chain no longer verifies. This gives operators a practical tamper-evidence mechanism without requiring a remote service for normal local use.

## What Verification Profiles Do

Integrity alone only proves that the recorded history is internally consistent. Verification profiles add policy meaning to that history. A profile can express expectations such as which event types must appear in a retrieval workflow, which required fields must be present, or which control points should exist before a privileged action is allowed. When `atb verify` runs with a profile, it evaluates the bundle against those obligations and produces structured findings about missing evidence, broken relationships, or residual risk.

## MCP Server Support

ATB also ships with an MCP server so AI agents and local tooling can interact with the same audit trail through a standard interface. The server exposes tools for bundle operations and emits canonical event types for workflow evidence. In the RAG integration, dedicated event types record when an index is created and when a retrieval decision is made. That means an agent can use retrieval as part of its work while leaving behind an auditable record of which corpus version was indexed and which node was selected for the answer path.

## PageIndex Integration

PageIndex adds reasoning-oriented document indexing and retrieval on top of ATB's evidence layer. A document is first converted into a structured tree of nodes with titles, summaries, and page spans. ATB records the resulting index hash, source URI, and node count as an `atb.event.rag_index` event. Later, when a user asks a question, PageIndex selects the best matching node and ATB records that decision as an `atb.event.rag_retrieval` event with the query, node identifier, title, page range, and latency. Together, the two systems provide both useful retrieval behavior and a tamper-evident provenance trail for how that retrieval happened.
