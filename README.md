# ATB: portable evidence for AI-agent incidents

Source version: [`v2.0.0`](CHANGELOG.md)

[![CI](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/badge/go-1.26.10-blue)](https://go.dev/)
[![MIT licence](https://img.shields.io/badge/licence-MIT-green.svg)](LICENSE)

ATB captures agent and tool activity in portable, tamper-evident bundles that
can be verified offline.

**ATB proves the integrity and order of records in a bundle. It does not prove
that every relevant event was captured, that the actor was honest, or that a
recorded claim is true.**

## Quickstart

Install the CLI:

```sh
go install github.com/pcguest/atb/v2/cmd/atb@latest
```

Create a bundle, record an event, and verify it:

```sh
atb bundle new
atb append ai.request.received --data='{"request_id":"req-1042"}'
atb verify run.atb/bundle.atb
```

For a complete walkthrough, including Python and TypeScript SDKs, see the
[five-minute quickstart](docs/getting-started/quickstart.md). To build the
embedded local viewer from source:

```sh
git clone https://github.com/pcguest/atb.git
cd atb
make build
./atb view run.atb/bundle.atb
```

## Use ATB

- [Examples](examples/README.md), including a reproducible incident walkthrough
- [Capture and integrations](docs/integrations/README.md)
- [Go, Python, and TypeScript SDKs](sdk/README.md)
- [Offline verification](docs/investigate/verify.md)
- [Incident investigation and evidence export](docs/investigate/incidents.md)

Verification is local. Optional signing, timestamp, encryption, and external
custody integrations add capabilities without changing ATB's capture boundary.

## Documentation

Start with the [quickstart](docs/getting-started/quickstart.md) →
[architecture](docs/concepts/architecture.md) →
[verification and limitations](docs/verification-meaning.md) →
[integrations](docs/integrations/README.md) →
[contributing](CONTRIBUTING.md). See the [documentation hub](docs/README.md) for
reference material and task guides.

[Documentation hub](docs/README.md) ·
[Architecture](docs/concepts/architecture.md) ·
[Trust model](docs/concepts/trust-model.md) ·
[Bundle format](docs/specification/bundle-v1.md) ·
[Event semantics](docs/specification/events.md) ·
[Local viewer API](docs/api/README.md)

## Security and contributing

See [SECURITY.md](SECURITY.md) for private vulnerability reporting and
[CONTRIBUTING.md](CONTRIBUTING.md) for setup, development, and validation.
ATB's licence is [MIT](LICENSE).
