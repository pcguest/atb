<!--
README above-the-fold design notes:

- Today it emphasises: local-first tamper-evident bundles, SHA-256/RFC 8785/RFC 3161 primitives,
  six obligation profiles with CAS, and multi-surface SDK support.
- Sections/phrases strong enough to keep: the one-sentence description, the integrity primitive
  note, the obligation profiles table, the trust model caveat, the Why ATB contrast with
  observability tools, and the Quickstart.
- Sections/phrases to replace or move down: the bare "Surfaces:" line (replace with a bullet in
  Why ATB that names all surfaces including atb view and atb push); the Why ATB prose paragraph
  (convert to bullets to match the rest of the README style). Also add a "What ATB does not do"
  block and a Demo hooks stub near the top.
- For launch it should emphasise:
  - Local-first, tamper-evident bundles; no backend required by default.
  - Conservative cryptography: SHA-256, RFC 8785, RFC 3161; explicit trust model.
  - Six schema-locked profiles and CAS as a local completeness signal, not compliance.
  - Multi-surface support: CLI, Python/TypeScript SDKs, MCP bridge, LangChain, PageIndex,
    atb view (with profile/CAS summary panel), atb push (S3/WORM export).
-->

# ATB

Records workflow events as tamper-evident, SHA-256 hash-chained bundles on local disk. No backend required.

[![Release](https://img.shields.io/badge/release-v1.7.2-blue.svg)](CHANGELOG.md) [![Go Reference](https://pkg.go.dev/badge/github.com/pcguest/atb.svg)](https://pkg.go.dev/github.com/pcguest/atb) [![CI](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml) [![Security](https://github.com/pcguest/atb/actions/workflows/security.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/security.yml) [![Licence](https://img.shields.io/badge/licence-MIT-green.svg)](LICENSE)

Current release: [`v1.7.2`](CHANGELOG.md)

Integrity primitive: SHA-256 hash chaining over RFC 8785 canonical JSON. Optional RFC 3161 TSA
anchoring adds a third-party timestamp commitment. ATB proves integrity of what was recorded; it
does not prove recording completeness, model correctness, or that risk controls were applied.

## Why ATB

- **Local-first, no hosted service.** Bundles are portable `.atb` files on disk. The core
  `init → append → snapshot → verify` workflow requires no network access and routes no payload
  data to third-party infrastructure.
- **Conservative, explicit cryptographic primitives.** SHA-256 over RFC 8785 canonical JSON for
  hash chaining; optional RFC 3161 TSA anchoring for third-party timestamp commitments.
- **Six schema-locked obligation profiles with CAS.** Profiles define required event sets for
  concrete workflows (RAG answers, privileged tools, policy decisions, human overrides, data
  exports, background jobs). CAS is a profile-scoped completeness signal for recorded evidence,
  not a compliance determination.
- **Multi-surface instrumentation.** Go CLI · Python SDK · TypeScript SDK · MCP stdio bridge ·
  LangChain callbacks · PageIndex retriever · `atb view` local dashboard · `atb push` WORM export.
- **Opt-in WORM export.** `atb push` uploads bundles to S3-compatible stores with optional Object
  Lock headers. This is a complement to the local guarantee, not a replacement.

### What ATB does not do

- Does not stream events to a hosted backend or observability pipeline.
- Does not evaluate model correctness, risk controls, or causal completeness beyond what is
  recorded in the bundle.
- Does not produce an external audit opinion; CAS is a local score over recorded evidence only.
- Does not replace filesystem integrity monitoring or WORM storage for regulated deployments where
  bundle-replacement attacks are in scope.

## Demo

See `docs/launch/assets/atb-verify-demo.gif` for a short terminal walkthrough (bundle init,
events appended, `atb verify` success, manual tamper + failing verify, and `atb view` surfacing
the profile/CAS summary panel). See `docs/launch/assets/atb-verify-report.png` for an example
verify report summary showing profile ID, pass/fail, CAS score/grade, chain/anchor status, and
critical obligation failures.

## Obligation profiles

Six schema-locked profiles define the required event sets for concrete workflows:

| Profile | Records |
|---------|---------|
| `atb.profile.privileged_tool_action` | A privileged tool invocation requiring explicit pre-commit authorisation |
| `atb.profile.rag_answer` | A retrieval-augmented generation answer with cited sources |
| `atb.profile.policy_decision` | A policy evaluation and its outcome |
| `atb.profile.human_override` | A human operator overriding an automated decision |
| `atb.profile.background_automation` | An unattended automated task running without human-in-the-loop |
| `atb.profile.data_export` | A data export with recipient, data set, and recorded legal basis |

For compliance control mappings see [`docs/compliance/`](docs/compliance/).

## Quickstart

```bash
atb bundle new
atb append ai.request.received \
  --data '{"workflow":"support-triage","case_id":"case-1042","severity":"sev2"}'
atb append ai.action.precommit \
  --data '{"action_id":"act-1042","action":"escalate","target":"tier-2","approved_by":"ops-lead"}'
atb snapshot incident_review
atb verify
```

`atb verify` checks the SHA-256 hash chain and any recorded bundle
signature material. Add `--with-anchor` to verify RFC 3161 timestamp
token material when an anchor is present. Mutation, reordering, or
deletion breaks the chain.

`atb bundle new` is the explicit alias for `atb init`.

## How it works

```text
event_hash = SHA-256(prev_hash || RFC8785(event_json))
genesis:    prev_hash = 0000...0000 (64 zeros)
```

Events are appended sequentially. Each event seals the previous one.
The resulting bundle is a portable artefact that can be verified later
without a server.

## Trust model

ATB proves that a bundle was not altered after recording. It does not
prove that recording was complete, that every relevant event was
captured, or that the bundle file itself has not been replaced
wholesale by an attacker with write access before export. For regulated
deployments, pair ATB with filesystem integrity monitoring or export to
a WORM-capable store before relying on bundles as primary evidence.

## Install

```bash
go install github.com/pcguest/atb/cmd/atb@latest
```

> Note: `go install` from the module proxy builds without the embedded web UI. For `atb view` and `atb view --ui-experimental`, build from source: `go build ./cmd/atb`
>
> Security note: `atb view` binds to `127.0.0.1` by default and has no authentication layer. It is intended for single-user local inspection only.

Requires Go 1.25.0+. Python and TypeScript SDKs are available for
in-process instrumentation. See the [Python SDK](sdk/python/README.md)
and [TypeScript SDK](sdk/typescript/README.md).

## Verification profiles

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

CAS is emitted in JSON output for all six built-in profiles. It is a
profile-scoped completeness signal for recorded evidence within the
declared profile and trust boundary, not an external attestation or a
universal completeness score.

Example `atb verify --profile atb.profile.rag_answer --format json`
output for a bundle with a broken hash chain. `pass: false` combined
with `residual_risk: "Critical"` indicates chain integrity failure;
`critical_failures` lists obligation gaps the profile found as a result.

```json
{
  "bundle_path": "run.atb/2026-04-14T09-12-03Z.atb",
  "profile_id": "atb.profile.rag_answer",
  "pass": false,
  "cas_score": 0.18,
  "cas_grade": "Insufficient",
  "sub_scores": {
    "AC": 0.0,
    "EC": 0.5,
    "FC": 0.25,
    "GC": 0.0,
    "RC": 0.0,
    "SC": 0.0,
    "TC": 0.5,
    "XC": 0.0
  },
  "critical_failures": [
    {
      "kind": "missing_event",
      "detail": "required event type not present: ai.model.invoked"
    }
  ],
  "required_warnings": [],
  "informational_notes": [],
  "residual_risk": "Critical"
}
```

## Integrations

ATB includes SDKs for Python and TypeScript, and it ships a local MCP
stdio bridge via `atb mcp serve`. The bridge exposes status,
verification, bundle initialisation, and PageIndex RAG recording tools.
It does not auto-instrument third-party MCP servers, and the current SDKs
do not include MCP-specific helper APIs. Run `atb events` to inspect the
canonical event registry and built-in profile membership.

### LangChain

The Python SDK ships `ATBCallbackHandler`, a LangChain callback handler that emits `ai.llm.call`, `ai.tool.exec`, and `ai.chain.run` events into the active bundle automatically.

```python
from atb import Bundle
from atb.langchain_callback import ATBCallbackHandler

bundle = Bundle()
handler = ATBCallbackHandler(bundle=bundle)

llm = ChatOpenAI(model="gpt-4o-mini", callbacks=[handler])
chain = prompt | llm
chain.invoke({"question": "What is tamper-evident logging?"})
```

See [`docs/integrations/langchain.md`](docs/integrations/langchain.md) for the full integration guide.

### MCP bridge (beta)

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

## Docs index

- [Architecture](docs/architecture.md)
- [Quickstart](docs/quickstart.md)
- [ATB Specification v1.0](docs/spec-v1.0.md)
- [AI Trace Event Specification](docs/spec-ai-traces.md)
- [Verification Profiles](docs/profiles.md)
- [Security Model](docs/security.md)
- [MCP Integration](docs/integrations/mcp.md)
- [WORM/S3 Export](docs/integrations/worm-s3.md)
- [Python SDK](sdk/python/README.md)
- [TypeScript SDK](sdk/typescript/README.md)
- [Incident Review Workflow](docs/guides/incident-review-workflow.md)
- [Customer Handoff Workflow](docs/guides/customer-handoff-workflow.md)
- [Key Management](docs/key-management.md)
- [Compliance Export](docs/compliance/export.md)
- [EU AI Act Mapping](docs/compliance/eu-ai-act.md)
- [SIEM and GRC Integration](docs/integrations/siem-grc.md)
- [Changelog](CHANGELOG.md)

## Attribution

ATB uses [PageIndex](https://github.com/VectifyAI/PageIndex) by
Vectify AI under the MIT Licence in the Python SDK. See
[THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES) for the full licence text.

## Licence

MIT. See [LICENSE](LICENSE).
Copyright (c) 2026 Patrick Guest.
