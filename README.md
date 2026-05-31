ATB — tamper-evident audit trails for AI systems.

[![CI](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml) ![Go version](https://img.shields.io/badge/go-1.26.3-blue) [![Licence](https://img.shields.io/badge/licence-MIT-green.svg)](LICENSE) ![EU AI Act Article 12 logging](https://img.shields.io/badge/EU%20AI%20Act-Article%2012%20logging-blue)

## What ATB does

ATB records AI workflow events into append-only `.atb` bundles whose records are linked by a SHA-256 hash chain. It is local-first: bundles are written to disk and can be verified without a hosted service or vendor account. The CLI and Go, Python, and TypeScript SDKs support capture, signing, verification, export, and profile-scored review. Its current logging model aligns with EU AI Act Article 12 by preserving automatic logs with retention guardrails and independently verifiable custody reports.

## Quickstart

Install the CLI:

```bash
go install github.com/pcguest/atb/cmd/atb@latest
```

Capture an AI workflow and verify the bundle path printed by the command:

```bash
atb capture run -- your-ai-script.sh
atb verify <bundle-file>
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

ATB canonicalises each event with RFC 8785 before hashing. Each record hash is SHA-256 over the previous hash and canonical event JSON, with a zero hash genesis sentinel. Bundles can be signed with Ed25519 and anchored through an RFC 3161 timestamp authority. Sensitive bundles can be encrypted with AES-256-GCM before storage or transfer. Verification recomputes the chain and checks signatures, anchors, profile obligations, and CAS output.

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

Go — `pkg/api/v1`
Python — `sdk/python`
TypeScript — `sdk/typescript`

## EU AI Act

ATB's automatic logging model aligns with EU AI Act Article 12 traceability requirements for recorded high-risk AI workflows. The retention guard prevents configuration below the EU AI Act minimum unless the operator explicitly allows it. CAS scoring exposes residual capture risk, but ATB does not prove universal capture, model correctness, actor identity, or legal compliance by itself.

## Roadmap

See [docs/roadmap.md](./docs/roadmap.md).

## Licence

MIT
