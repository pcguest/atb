# ATB

[![Version](https://img.shields.io/badge/version-v1.1.0-blue)](docs/releases/v1.1.0.md)
[![CI](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml)
[![Security Gate](https://github.com/pcguest/atb/actions/workflows/security.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/security.yml)
[![Security Scan](https://github.com/pcguest/atb/actions/workflows/security-scan.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/security-scan.yml)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Tamper-evident, local-first audit trails for privacy-sensitive AI agents.

ATB creates verifiable trace bundles that teams can inspect locally, verify cryptographically, and export as deterministic evidence for incident review, privacy-sensitive debugging, and compliance workflows. It is designed for teams that need proof of what happened, not another hosted observability control plane.

## Why Teams Use ATB

- Reconstruct agent failures, tool misuse, and high-risk decisions with a verifiable execution trail.
- Keep trace storage, verification, and inspection local by default.
- Produce deterministic SOC 2 and GDPR evidence without rebuilding logs after the fact.
- Share protected bundles through client-side encryption when controlled handoff is required.

## Release Status

- Current release: [`v1.1.0`](docs/releases/v1.1.0.md)
- Gold release status: [`APPROVED FOR GOLD RELEASE`](docs/security/gold-signoff.md)
- Current GitHub checks cover Go, Node, Python, Trivy image and filesystem scans, the consolidated security scan, the golden test, and cross-platform CI on macOS, Ubuntu, and Windows

## 5 Minute Start

```bash
pip install atb-sdk
atb init
atb view
```

Verify integrity at any time:

```bash
atb verify
```

## What ATB Includes

- **Tamper-evident event logs:** SHA-256 hash chains with RFC 8785 canonical JSON catch mutation, reordering, and deletion.
- **Local-first verification:** trace inspection and verification run locally, with no required backend.
- **Client-side encryption:** AES-256-GCM encryption for protected bundle handoff workflows.
- **Deterministic evidence export:** built-in `soc2` and `gdpr` export paths for controls evidence, DSR, and RoPA workflows.
- **Trust dashboard:** `atb view` provides a local dashboard with timeline, graph, inspector, role-based views, and privacy reveal audit logging.
- **Developer integrations:** native tracing middleware for LangChain in Python and Vercel AI SDK in TypeScript.

## Best Fit

ATB is best suited to:

- internal copilots and agent workflows handling sensitive or regulated data
- enterprise AI teams that cannot send raw traces to a hosted vendor by default
- consultancies and delivery teams that need a portable audit artefact for handoff or review

ATB is not intended to be a generic hosted LLM observability platform.

## Installation

- Python CLI package: `pip install atb-sdk`
- TypeScript SDK: `npm install @pcguest/atb-sdk`
- Go CLI: `go install github.com/pcguest/atb/cmd/atb@latest`
- Docker: `docker run --rm -it -v $(pwd):/data ghcr.io/pcguest/atb:latest`

## Documentation

Start at [Docs Home](docs/README.md).

- [Quickstart](docs/quickstart.md)
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
