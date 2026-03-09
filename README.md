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

## Status

> Phase 3 completed on 2026-03-05: retention config, archive ledger, and compliance export are available.
> Track shipped and planned work in [docs/STATUS.md](docs/STATUS.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## License

MIT. See [LICENSE](LICENSE).
