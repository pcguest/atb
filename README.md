# ATB

[![Version](https://img.shields.io/badge/version-v1.8.1-blue)](CHANGELOG.md)
[![CI](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml)
[![Security Gate](https://github.com/pcguest/atb/actions/workflows/security.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/security.yml)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

ATB is the local-first audit trail for privacy-sensitive AI systems.

It records AI workflow events as tamper-evident bundles you can inspect locally, verify cryptographically, and export as deterministic evidence for incident review, customer handoff, and internal audit or privacy review. It is designed for teams that need a tamper-evident record of what was recorded without default external trace storage.

## Why Teams Pick ATB

| Need | How ATB addresses it |
| --- | --- |
| Keep raw traces local by default | ATB records bundles locally and does not require default external trace storage. |
| Reconstruct failures, tool misuse, and high-risk decisions | Every appended event is hash-chained in a verifiable execution trail you can inspect locally. |
| Hand over portable audit evidence | Bundles and deterministic exports travel as artifacts instead of reconstructed notes or screenshots. |
| Record privacy reveals in the same evidence trail | Privacy reveal events are appended to the same tamper-evident bundle as the rest of the workflow. |

## Release Status

- Current release: [`v1.8.1`](CHANGELOG.md)
- Gold release status: [`APPROVED FOR GOLD RELEASE`](docs/security/gold-signoff.md)
- All CI and security gates pass on `main`. See [Actions](https://github.com/pcguest/atb/actions).

## 5 Minute Start

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb bundle new
atb append agent.run --data='{"workflow":"support-triage","case_id":"case-1042","severity":"sev2"}'
atb append policy.alert --data='{"check":"pii_redaction","outcome":"fail","ticket_id":"case-1042"}'
atb snapshot incident_review --gate fail
atb verify
atb trust-report --format markdown
```

That sequence creates a local incident bundle with a failed review gate but a valid evidence chain. `atb bundle new` is the explicit alias for `atb init`; both are supported. For the full review path, including the local dashboard and evidence export, use the [Incident Review Workflow](docs/guides/incident-review-workflow.md). For sender and recipient handoff, use the [Customer Handoff Workflow](docs/guides/customer-handoff-workflow.md).

## Verification Profiles

ATB includes six built-in obligation profiles. Use `--profile <id>` to evaluate against a specific built-in profile, or pass a local YAML profile path when you are validating a custom definition:

```text
atb.profile.privileged_tool_action
atb.profile.rag_answer
atb.profile.data_export
atb.profile.policy_decision
atb.profile.human_override
atb.profile.background_automation
```

```bash
atb verify --bundle run.atb/bundle.atb --profile atb.profile.privileged_tool_action
atb verify --bundle run.atb/bundle.atb --profile atb.profile.rag_answer --json
# The JSON output includes a "cas" object with "grade" and sub_scores
# such as AC, EC, FC, GC, RC, SC, TC, and XC for the selected profile.
atb verify --bundle run.atb/bundle.atb --profile ./profiles/release-review.yaml
```

## What ATB Includes

| Capability | Detail |
| --- | --- |
| Tamper-evident event logs | SHA-256 hash chains with RFC 8785 canonical JSON catch mutation, reordering, and deletion. |
| Local-first verification | Trace inspection and verification run locally, with no required backend. |
| Client-side encryption | AES-256-GCM encryption for protected bundle handoff workflows. |
| Deterministic evidence export | `compliance`, `soc2`, and `gdpr` export paths for incident review, controls evidence, DSR, and RoPA workflows. |
| Local viewer and dashboard | `atb view` serves the local viewer, and `atb view --ui-experimental` enables the role-based dashboard with timeline, graph, inspector, and privacy reveal audit logging. |
| Developer integrations | Native tracing middleware for LangChain in Python and Vercel AI SDK in TypeScript. |
| Go CLI as the primary distribution path | Python and TypeScript packages are SDKs that write the same bundle format, not the primary CLI install path. |

- [Why ATB: integrity, completeness, and what we claim](docs/why-atb.md)

## Verification model

```text
event_hash = SHA-256(prev_hash || RFC8785(event_json))
genesis:    prev_hash = "0000...0000" (64 zeros)
```

Every event in a bundle is bound to its predecessor; any mutation, reordering, or deletion breaks the chain.

## Best Fit

ATB is best suited to:

- security-minded AI teams running internal copilots or agent workflows with sensitive data
- consultancies and delivery teams that need a portable audit artefact for handoff or review
- enterprise builders that need a reviewable local evidence layer for internal audit or privacy review

ATB is not intended to be a generic hosted LLM observability platform.

## Installation

- Go CLI: `go install github.com/pcguest/atb/cmd/atb@latest` (requires Go 1.21+)
- Python SDK: `pip install atb-sdk`
- TypeScript SDK: `npm install @pcguest/atb-sdk`
- Docker: build locally with `docker build -t atb .`

Python and TypeScript packages are SDKs only. Their installed `atb` command is a compatibility stub that prints Go CLI install guidance and will be removed in a future major release.

## Documentation

Start at [Docs Home](docs/README.md).

- [Quickstart](docs/quickstart.md)
- [Incident Review Workflow](docs/guides/incident-review-workflow.md)
- [Incident Review for Private AI Workflows](docs/use-cases/incident-review.md)
- [Customer Handoff Workflow](docs/guides/customer-handoff-workflow.md)
- [Customer Handoff Without Platform Lock-In](docs/use-cases/customer-handoff.md)
- [Internal Audit and Privacy Review](docs/use-cases/internal-audit-privacy-review.md)
- [ATB vs Hosted AI Observability](docs/comparisons/hosted-observability.md)
- [ATB vs Logs, Screenshots, and Ad Hoc Exports](docs/comparisons/logs-and-screenshots.md)
- [Dashboard Specification](docs/spec-dashboard.md)
- [AI Integrations](docs/guides/README.md)
- [ATB Specification v1.0](docs/spec-v1.0.md)
- [AI Trace Event Specification](docs/spec-ai-traces.md)
- [Compliance Export Overview](docs/compliance/export.md)
- [Contributing Guide](CONTRIBUTING.md)
- [Security Policy](SECURITY.md)
- [Changelog](CHANGELOG.md)
