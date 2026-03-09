# ATB - Agent Trace Bundle

[![CI](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml)
[![PyPI](https://img.shields.io/pypi/v/atb-sdk?label=PyPI)](https://pypi.org/project/atb-sdk)
[![npm](https://img.shields.io/npm/v/%40pcguest%2Fatb-sdk?label=npm)](https://www.npmjs.com/package/@pcguest/atb-sdk)
[![Security Gate](https://github.com/pcguest/atb/actions/workflows/security.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/security.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-indigo.svg)](https://github.com/pcguest/atb/blob/main/LICENSE)

> Cryptographically verifiable audit trails for AI agent workflows.

## What Is ATB?

- Tamper-evident audit trails using SHA-256 hash chaining and RFC 8785 canonical JSON.
- Local-first by default. Optional cloud sharing with client-side encryption.
- Cross-language SDKs for Go, Python, and TypeScript.
- Compliance exports: `atb export --format soc2` and `atb export --format gdpr --type dsr|ropa`.

## Try It in 60 Seconds

```bash
pip install atb-sdk
atb init
atb append agent.step --data '{"x":1}'
atb verify
```

Install the CLI with `go install github.com/pcguest/atb/cmd/atb@latest` if `atb` is not on your `PATH`.

## Learn More

- [Quickstart](docs/quickstart.md)
- [Security](docs/security.md)
- [AI Integration](docs/ai-integration.md)
- [Specification v1.0](docs/spec-v1.0.md)
- [Retention](docs/compliance/retention.md)
- [Compliance Export](docs/compliance/export.md)

## Integrations

- [LangChain Integration (Python)](docs/integrations/langchain.md)
- [Vercel AI SDK Integration (TypeScript)](docs/integrations/vercel-ai.md)

## Dashboard

Local-first visual trace exploration is now available via `atb view`.

- Launch: `atb view run.atb/bundle.atb --log-reveals`
- Includes tamper detection state, timeline browsing, and trace/span graph exploration.
- Spec: [Dashboard Specification](docs/spec-dashboard.md)

## Status

> ✅ **Phase 6 Complete:** Visual Dashboard (`atb view`), interactive trace exploration, and privacy audit logging.
> Track shipped and planned work in [docs/STATUS.md](docs/STATUS.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## License

MIT. See [LICENSE](LICENSE).
