# ATB — tamper-evident audit trails for AI systems

[![CI](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml) ![Go version](https://img.shields.io/badge/go-1.26.4-blue) [![Licence](https://img.shields.io/badge/licence-MIT-green.svg)](LICENSE) ![EU AI Act Article 12 logging](https://img.shields.io/badge/EU%20AI%20Act-Article%2012%20logging-blue)

**Record what your AI agents actually did — into append-only, hash-chained
bundles that anyone can verify offline, with no vendor account and no trust
in the operator.**

Current release: [`v1.14.2`](CHANGELOG.md)

- **Tamper-evident by construction** — every event is RFC 8785-canonicalised
  and SHA-256 chained; one flipped byte breaks verification.
- **Verify with zero trust** — `atb verify` runs locally against the bundle
  alone; Go, Python, and TypeScript verifiers agree byte-for-byte
  (cross-language golden vectors in CI).
- **Built for incidents, not dashboards** — capture an agent session, then
  prove after the fact which privileged action fired, what was approved, and
  what failed — even when the agent's own logs can't be trusted.

## Quickstart

Install the CLI:

```bash
go install github.com/pcguest/atb/cmd/atb@latest
```

A `go install` build serves a minimal install-guidance page for `atb view`;
for the full embedded review UI, build from a checkout with `make build`.

Initialise a bundle, capture a workflow, and verify against a profile:

```bash
atb bundle new
atb capture run -- your-ai-script.sh
atb verify --profile atb.profile.policy_decision <bundle-file>
atb view <bundle-file> --profile atb.profile.policy_decision
```

## Agent incident forensics

ATB doubles as a local-first flight recorder for AI agents. `atb intercept`
records an agent's provider API traffic, tool calls, and failures into a
tamper-evident bundle — request/response bodies are digested (not stored) by
default, and credential headers are stripped. Afterwards:

```bash
atb incident list   --bundle <bundle-file>                  # discover sessions + anomalies
atb incident report --bundle <bundle-file> --session <id>   # session-scoped forensic report
```

The report shows integrity (hash chain), bundle signature provenance, anomaly
flags such as `tool_without_approval`, and every event hash-addressed to the
authoritative bundle — so a failed or unapproved privileged action is provable
even when the agent's own logs can't be trusted. See the
[agent incident forensics walkthrough](./docs/guides/agent-incident-forensics.md).

## How it works

ATB records AI workflow events into append-only `.atb` bundles (NDJSON). Each
event is canonicalised with RFC 8785 before hashing; each record hash is
SHA-256 over the previous hash and the canonical event JSON, with a zero-hash
genesis sentinel. Bundles can be signed with Ed25519 and anchored through an
RFC 3161 timestamp authority; sensitive bundles can be encrypted with
AES-256-GCM before storage or transfer. Verification recomputes the chain and
checks signatures, anchors, profile obligations, and CAS output — entirely
locally.

```
agent / app ──► capture (SDK · intercept proxy · OTel import)
                    │
                    ▼
              .atb bundle  (append-only · hash-chained · signed)
                    │
        ┌───────────┼───────────────┐
        ▼           ▼               ▼
   atb verify   atb incident    atb view
   (offline)    (forensics)     (local viewer)
                    │
                    ▼ optional push
          Custos (custody · WORM · receipts · transparency log)
```

Long-term custody — WORM storage, signed receipts, and an RFC 6962
transparency log with witness cosignatures — lives in the companion product
[Custos](https://github.com/pcguest/custos-product); the full story is in its
[end-to-end guide](https://github.com/pcguest/custos-product/blob/main/docs/e2e-atb-custos.md).

## Obligation profiles

| Profile ID | Purpose | CAS scored |
| --- | --- | --- |
| `atb.profile.privileged_tool_action` | Privileged tool execution and policy gating | Yes |
| `atb.profile.rag_answer` | Retrieval-augmented answer evidence | Yes |
| `atb.profile.data_export` | Data export authorisation and movement | Yes |
| `atb.profile.policy_decision` | Policy decision rationale and subject linkage | Yes |
| `atb.profile.human_override` | Human override approval and justification | Yes |
| `atb.profile.background_automation` | Background job scheduling and execution | Yes |

## SDKs

| Language | Path |
| --- | --- |
| Go | [`pkg/api/v1`](./pkg/api/v1) |
| Python | [`sdk/python`](./sdk/python) |
| TypeScript | [`sdk/typescript`](./sdk/typescript) |

All three agree byte-for-byte with the Go implementation via shared golden
vectors (`make test-golden`).

## EU AI Act

ATB's automatic logging model aligns with EU AI Act Article 12 traceability
requirements for recorded high-risk AI workflows. The retention guard prevents
configuration below the EU AI Act minimum unless the operator explicitly
allows it. CAS scoring exposes residual capture risk.

**Honest limits:** ATB proves integrity of what was recorded. It does not
prove universal capture, model correctness, actor identity, or legal
compliance by itself — and it never certifies compliance.

## Documentation

| Read | For |
| --- | --- |
| [Documentation hub](./docs/README.md) | Start here and choose the developer, auditor, or operator path |
| [Five-minute quickstart](./docs/quickstart.md) | Install ATB, create a bundle, and verify it locally |
| [Developer capture guide](./docs/guides/capture-quickstart.md) | Instrument a workflow and understand capture boundaries |
| [Auditor acceptance guide](./docs/ciso-acceptance-guide.md) | Review integrity, profile results, CAS, and residual risk |
| [Operator WORM guide](./docs/integrations/worm-s3.md) | Store bundles under operator-controlled immutable retention |
| [Bundle specification](./docs/spec-v1.0.md) | Frozen format, hashing, and canonicalisation contract |
| [Security model](./docs/security.md) | Threat model, guarantees, and explicit limitations |
| [Submission / evaluation](https://github.com/pcguest/custos-product/blob/main/docs/SUBMISSION.md) | Versions, evaluator commands, never-claims, and shipped boundary |

## Contributing & security

- [CONTRIBUTING.md](./CONTRIBUTING.md) — local release gates (golden vectors,
  full test suite, version checks) are the gates of record.
- [SECURITY.md](./SECURITY.md) — reporting vulnerabilities.
- [VERSIONING.md](./VERSIONING.md) — SemVer for releases; the canonical hash
  input is frozen and changes only via a deliberate manifest-version migration.

## Licence

[MIT](./LICENSE)
