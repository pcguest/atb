# ATB: portable evidence for AI-agent incidents

Source version: [`v2.0.0`](CHANGELOG.md) (unreleased; not a publication claim).

[![CI](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/badge/go-1.26.10-blue)](https://go.dev/)
[![MIT licence](https://img.shields.io/badge/licence-MIT-green.svg)](LICENSE)

ATB captures agent and tool activity in portable, tamper-evident bundles that
can be verified offline.

**ATB proves the integrity and order of records in a bundle. It does not prove
that every relevant event was captured, that the actor was honest, or that a
recorded claim is true.**

The selected developer-preview candidate is the Go CLI at commit
`07ec8ba793a5f0cf5ca99a95e7e280131b6863b4`. It covers local capture, portable
bundles, integrity verification, optional signing, profile evaluation, and
machine-readable output. It is not a published release, a custody service, or
a compliance certification. The source-built investigation UI remains
available as an experimental local surface. It is implemented, and its
independent manual accessibility acceptance has not been run.

## Quickstart

Choose an installation route:

| Route | CLI verification | Visual investigation |
| --- | --- | --- |
| Published CLI release | Features of that release; use its matching docs | Check that release’s assets and documentation |
| `go install` | Yes, for the module version Go resolves | Install-guidance page only; no embedded investigation UI |
| Source checkout + `make build` | Yes | Experimental embedded UI; implemented, manual accessibility acceptance not run |

As of **2026-11-08**, the latest [GitHub release](https://github.com/pcguest/atb/releases/latest),
[Python SDK](https://pypi.org/project/atb-sdk/) and
[TypeScript SDK](https://www.npmjs.com/package/@pcguest/atb-sdk) are **1.18.1**.
The current source reports **2.0.0**; no 2.0.0 tag or SDK publication exists at
that checkpoint. SDK packages are libraries, not the full Go CLI or viewer.

Install the preview candidate without the embedded UI:

```sh
atb_install_dir="$HOME/.local/bin"
GOBIN="$atb_install_dir" go install github.com/pcguest/atb/v2/cmd/atb@07ec8ba793a5f0cf5ca99a95e7e280131b6863b4
export PATH="$atb_install_dir:$PATH"
"$atb_install_dir/atb" version
go version -m "$atb_install_dir/atb"
```

That query resolves to module version `v2.0.0-20261109054510-07ec8ba793a5`,
commit `07ec8ba793a5f0cf5ca99a95e7e280131b6863b4`. `atb version` still prints
`2.0.0` from the source string. That string is not a published release.
`@latest` follows [Go’s version-selection rules](https://go.dev/ref/mod#version-queries)
and does not mean this commit. For prerequisites, see
[installation options](docs/getting-started/quickstart.md#2-installation-options).

Create a bundle, record an event, and verify it:

```sh
atb bundle new
atb append ai.request.received --data='{"request_id":"req-1042"}'
atb verify run.atb/bundle.atb
```

For a complete walkthrough, including Python and TypeScript SDKs, see the
[five-minute quickstart](docs/getting-started/quickstart.md).

The source build below is optional. It is outside the CLI preview acceptance
sequence. The embedded viewer is implemented and remains experimental.
Independent manual keyboard, screen-reader, and 200%/400% zoom acceptance has
not been run.

```sh
git clone https://github.com/pcguest/atb.git
cd atb
git checkout 07ec8ba793a5f0cf5ca99a95e7e280131b6863b4
make build
./atb version
./atb bundle new
./atb append ai.request.received --data='{"request_id":"req-1042"}'
./atb view run.atb/bundle.atb
```

Use Go 1.26.10 and Node.js 22 with npm; dependency downloads require network
access. `make build` builds the web assets before embedding them in `./atb`.
`go install` does not embed that UI.

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
