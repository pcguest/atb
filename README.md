# ATB

[![Version](https://img.shields.io/badge/version-v1.1.0-blue)](docs/releases/v1.1.0.md)
[![CI](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml)
[![Security Gate](https://github.com/pcguest/atb/actions/workflows/security.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/security.yml)
[![Security Scan](https://github.com/pcguest/atb/actions/workflows/security-scan.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/security-scan.yml)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

ATB is the local-first audit trail for privacy-sensitive AI systems.

It records AI workflow events as tamper-evident bundles you can inspect locally, verify cryptographically, and export as deterministic evidence for incident review, customer handoff, and internal audit or privacy review. It is designed for teams that need proof of what happened without default external trace storage.

## Why Teams Pick ATB

- Keep raw traces local by default.
- Reconstruct agent failures, tool misuse, and high-risk decisions with a verifiable execution trail.
- Hand over a portable bundle and deterministic export instead of reconstructed notes.
- Record privacy reveals as part of the same audit trail.

## Release Status

- Current release: [`v1.1.0`](docs/releases/v1.1.0.md)
- Gold release status: [`APPROVED FOR GOLD RELEASE`](docs/security/gold-signoff.md)
- Current GitHub checks cover Go, Node, Python, Trivy image and filesystem scans, the consolidated security scan, the golden test, and cross-platform CI on macOS, Ubuntu, and Windows

## 5 Minute Start

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb init
atb append agent.run --data='{"workflow":"support-triage","case_id":"case-1042","severity":"sev2"}'
atb append policy.alert --data='{"check":"pii_redaction","outcome":"fail","ticket_id":"case-1042"}'
atb snapshot incident_review --gate fail
atb verify
atb trust-report --format markdown
```

That sequence creates a local incident bundle with a failed review gate but a valid evidence chain. For the full review path, including the local dashboard and evidence export, use the [Incident Review Workflow](docs/guides/incident-review-workflow.md). For sender and recipient handoff, use the [Customer Handoff Workflow](docs/guides/customer-handoff-workflow.md).

## What ATB Includes

- **Tamper-evident event logs:** SHA-256 hash chains with RFC 8785 canonical JSON catch mutation, reordering, and deletion.
- **Local-first verification:** trace inspection and verification run locally, with no required backend.
- **Client-side encryption:** AES-256-GCM encryption for protected bundle handoff workflows.
- **Deterministic evidence export:** `compliance`, `soc2`, and `gdpr` export paths for incident review, controls evidence, DSR, and RoPA workflows.
- **Local viewer and dashboard:** `atb view` serves the local viewer, and `atb view --ui-experimental` enables the role-based dashboard with timeline, graph, inspector, and privacy reveal audit logging.
- **Developer integrations:** native tracing middleware for LangChain in Python and Vercel AI SDK in TypeScript.
- **Go CLI as the primary distribution path:** Python and TypeScript packages are SDKs that write the same bundle format, not the primary CLI install path.

## Best Fit

ATB is best suited to:

- security-minded AI teams running internal copilots or agent workflows with sensitive data
- consultancies and delivery teams that need a portable audit artefact for handoff or review
- enterprise builders that need a reviewable local evidence layer for internal audit or privacy review

ATB is not intended to be a generic hosted LLM observability platform.

## Installation

- Go CLI: `go install github.com/pcguest/atb/cmd/atb@latest`
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

## Licence

MIT. See [LICENSE](LICENSE).
