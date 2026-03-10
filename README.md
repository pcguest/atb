# Cryptographically Verifiable Audit Trails for AI Agents

[![Build Status](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml)
[![Security](https://github.com/pcguest/atb/actions/workflows/security.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/security.yml)
[![Version](https://img.shields.io/badge/version-v1.0.0-blue.svg)](https://github.com/pcguest/atb/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

ATB gives you tamper-evident, local-first audit trails for AI workflows so engineers and auditors can verify exactly what happened.

## Quick Start

```bash
pip install atb
atb init
atb view
```

Verify trace integrity anytime with `atb verify`.

If you prefer Go binaries, install with:

```bash
go install github.com/pcguest/atb/cmd/atb@latest
```

## Core Features

| Feature | What you get |
| --- | --- |
| Encryption | AES-256-GCM bundle protection with password-based key derivation. |
| SOC2/GDPR Exports | Deterministic compliance exports for control evidence and DSR/ROPA workflows. |
| AI Integrations | LangChain (Python) and Vercel AI SDK (TypeScript) tracing middleware. |
| Local Dashboard | `atb view` timeline, graph, and inspector with verification status and privacy controls. |

## Why Teams Trust ATB

- **Local-First:** Works offline and keeps data under your control.
- **Zero-Knowledge:** Keys and secrets stay client-side.
- **Tamper-Evident:** SHA-256 hash chains and canonical JSON verification on load/export.

## Installation Options

- Python package: `pip install atb`
- TypeScript SDK: `npm install @pcguest/atb-sdk`
- Go CLI: `go install github.com/pcguest/atb/cmd/atb@latest`
- Docker: `docker run --rm -it -v $(pwd):/data ghcr.io/pcguest/atb:latest`

## Documentation

- [Quickstart](docs/quickstart.md)
- [ATB Specification v1.0](docs/spec-v1.0.md)
- [Dashboard Specification](docs/spec-dashboard.md)
- [Launch Specification](docs/spec-launch.md)

## Project Policies

- [Contributing Guide](CONTRIBUTING.md)
- [Security Policy](SECURITY.md)
- [Versioning Policy](VERSIONING.md)
- [Changelog](CHANGELOG.md)

## License

MIT. See [LICENSE](LICENSE).
