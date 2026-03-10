# Cryptographically Verifiable Audit Trails for AI Agents

[![Build Status](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml)
[![Security (Trivy)](https://github.com/pcguest/atb/actions/workflows/security.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/security.yml)
[![Version](https://img.shields.io/badge/version-v1.0.3-blue.svg)](https://github.com/pcguest/atb/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

Tamper-evident, local-first observability for LangChain, Vercel AI, and custom agents. Built for engineers who need to prove what happened.

**Don't just trust your AI. Verify it.**

## Why ATB Exists

AI systems fail in ways that are difficult to reconstruct after the fact:

- Hallucinations and tool misuse without a reliable execution trail.
- Compliance requests (SOC2/GDPR) without deterministic evidence.
- Post-incident analysis blocked by mutable logs and missing context.

ATB gives teams verifiable trace evidence they can audit, reproduce, and export.

## How ATB Solves It

- Hash-chained event logs using SHA-256 and RFC 8785 canonical JSON.
- Local-first storage and verification (`run.atb/bundle.atb`) with no required backend.
- Optional client-side encryption for protected sharing workflows.
- Deterministic SOC2/GDPR export outputs for audit and legal workflows.

## 5-Minute Win

```bash
pip install atb
atb init
atb view
```

Verify integrity anytime:

```bash
atb verify
```

What you get immediately in `atb view`:

```text
+--------------------------------------------------------------+
| ATB Dashboard                                                 |
|  ✓ Chain Verified    Events: 12,482    Head: 6f1a...         |
|---------------------------------------------------------------|
| Timeline | Trace Graph | Inspector | Privacy Reveal Audit     |
+--------------------------------------------------------------+
```

## Core Features

- **Tamper-Evidence:** SHA-256 hash chains and canonical JSON verification catch mutation, reordering, and deletion.
- **Privacy & Encryption:** Local-first by default with client-side AES-256-GCM encryption and zero-knowledge key handling.
- **Compliance Exports:** Deterministic `soc2` and `gdpr` export paths for controls evidence, DSR, and RoPA workflows.
- **AI Integrations:** Native tracing middleware for LangChain (Python) and Vercel AI SDK (TypeScript).
- **Visual Dashboard:** `atb view` delivers a local dashboard with timeline, graph, inspector, tamper blocking state, and reveal auditing.

## Trust & Security

ATB is built for high-trust engineering and audit environments:

- **Local-First Security Model:** verification and trace inspection run locally.
- **Distroless Runtime Image:** Docker runtime is based on `distroless/static-debian12:nonroot`.
- **Security Scanning in CI:** gosec, Bandit, npm audit, Trivy filesystem/image scanning, and secret scanning gates.
- **Responsible Disclosure:** see [SECURITY.md](SECURITY.md).

## Installation Options

- Python package: `pip install atb`
- TypeScript SDK: `npm install @pcguest/atb-sdk`
- Go CLI: `go install github.com/pcguest/atb/cmd/atb@latest`
- Docker: `docker run --rm -it -v $(pwd):/data ghcr.io/pcguest/atb:latest`

## Documentation

Start at [Docs Home](docs/README.md).

### Getting Started

- [Quickstart](docs/quickstart.md)
- [Dashboard Specification](docs/spec-dashboard.md)
- [Integrations Overview](docs/guides/README.md)

### Deep Dives

- [ATB Specification v1.0](docs/spec-v1.0.md)
- [AI Trace Event Specification](docs/spec-ai-traces.md)
- [Compliance Export Overview](docs/compliance/export.md)
- [Launch Specification](docs/spec-launch.md)

### Community

- [Contributing Guide](CONTRIBUTING.md)
- [Security Policy](SECURITY.md)
- [Versioning Policy](VERSIONING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Changelog](CHANGELOG.md)

## License

MIT. See [LICENSE](LICENSE).
