# ATB

Tamper-evident audit trails for privacy-sensitive AI systems.

[![Go Reference](https://pkg.go.dev/badge/github.com/pcguest/atb.svg)](https://pkg.go.dev/github.com/pcguest/atb) [![CI](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml) [![Security](https://github.com/pcguest/atb/actions/workflows/security.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/security.yml) [![Licence](https://img.shields.io/badge/licence-MIT-green.svg)](LICENSE)

## What It Is

ATB records AI workflow events as tamper-evident, hash-chained bundles
you can inspect locally, verify cryptographically, and export as
deterministic evidence for incident review, audit, and customer
handoff. It does not require a backend and does not send trace data to
external storage by default.

Current release: [`v1.0.0.1`](CHANGELOG.md).

## Install

```bash
go install github.com/pcguest/atb/cmd/atb@latest
```

Requires Go 1.25.0+. Python and TypeScript SDKs are available for
in-process instrumentation. See [SDK docs](docs/guides/README.md).

## Quickstart

```bash
atb init
atb append ai.request.received \
  '{"workflow":"support-triage","case_id":"case-1042","severity":"sev2"}'
atb append ai.action.precommit \
  '{"action":"escalate","target":"tier-2","approved_by":"ops-lead"}'
atb snapshot incident_review
atb verify
```

`atb verify` checks the SHA-256 hash chain and any recorded bundle
signature material. Add `--with-anchor` to verify RFC 3161 timestamp
token material when an anchor is present. Mutation, reordering, or
deletion breaks the chain.

`atb bundle new` is the explicit alias for `atb init`.

## How It Works

```text
event_hash = SHA-256(prev_hash || RFC8785(event_json))
genesis:    prev_hash = 0000...0000 (64 zeros)
```

Events are appended sequentially. Each event seals the previous one.
The resulting bundle is a portable artefact that can be verified later
without a server.

## Verification Profiles

ATB ships six built-in obligation profiles. Use `--profile` to evaluate
whether a bundle contains the expected event sequence, field coverage,
and causal relations for a workflow.

```bash
atb verify --profile atb.profile.rag_answer
atb verify --profile atb.profile.privileged_tool_action --format json
atb verify --profile ./profiles/custom.yaml
```

| Profile | Use case |
| --- | --- |
| `atb.profile.rag_answer` | RAG answer provenance |
| `atb.profile.privileged_tool_action` | Gated high-impact tool execution |
| `atb.profile.data_export` | Data export evidence trail |
| `atb.profile.policy_decision` | Policy engine allow or deny record |
| `atb.profile.human_override` | Human-in-the-loop override chain |
| `atb.profile.background_automation` | Scheduled job audit trail |

CAS is emitted in JSON output for CAS-enabled profiles. It is a
profile-scoped completeness signal for recorded evidence, not an
external attestation.

## Integrations

ATB includes a native MCP server and SDKs for Python and TypeScript.
Run `atb events` to inspect the canonical event registry and built-in
profile membership.

### MCP Server

```bash
atb mcp serve
```

| Tool | Description |
| --- | --- |
| `atb_init` | Initialise a new bundle |
| `status` | Report version, bundle presence, chain length, and head hash |
| `verify` | Verify bundle integrity and profile results |
| `rag_index_record` | Record a PageIndex index build as `atb.event.rag_index` |
| `rag_retrieval_record` | Record a PageIndex retrieval as `atb.event.rag_retrieval` |

### PageIndex

The Python SDK includes `ATBPageIndexRetriever` for PageIndex-backed
document indexing and retrieval. `build_index()` records
`atb.event.rag_index`. `retrieve()` records `atb.event.rag_retrieval`.

```python
from atb import ATBPageIndexRetriever

retriever = ATBPageIndexRetriever(model="gpt-4o-2024-11-20")
tree, index_id = retriever.build_index("report.pdf")
node = retriever.retrieve(
    "What were net interest margins?",
    tree,
    index_id,
    "report.pdf",
)
```

Those records join the same bundle as your request, model, and response
events when you emit them, so retrieval evidence can be reviewed
alongside the rest of the workflow.

## Docs Index

- [Quickstart](docs/quickstart.md)
- [ATB Specification v1.0](docs/spec-v1.0.md)
- [AI Trace Event Specification](docs/spec-ai-traces.md)
- [Verification Profiles](docs/profiles.md)
- [MCP Integration](docs/integrations/mcp.md)
- [Incident Review Workflow](docs/guides/incident-review-workflow.md)
- [Customer Handoff Workflow](docs/guides/customer-handoff-workflow.md)
- [Key Management](docs/key-management.md)
- [Compliance Export](docs/compliance/export.md)
- [SIEM and GRC Integration](docs/integrations/siem-grc.md)
- [Changelog](CHANGELOG.md)

## Attribution

ATB uses [PageIndex](https://github.com/VectifyAI/PageIndex) by
Vectify AI under the MIT Licence in the Python SDK. See
[THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES) for the full licence text.

## Licence

MIT. See [LICENSE](LICENSE).
Copyright (c) 2026 Patrick Guest.
