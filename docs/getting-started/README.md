# Getting Started

Start here if you are new to ATB.

## First 5 Minutes

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb init
atb view
```

Then verify integrity:

```bash
atb verify
```

## Recommended Path

1. [Quickstart](../quickstart.md)
2. [Dashboard Specification](../spec-dashboard.md)
3. [Compliance Export Overview](../compliance/export.md)
4. [AI Integrations](../guides/README.md)

## Install Options

- Go CLI: `go install github.com/pcguest/atb/cmd/atb@latest`
- Python SDK: `pip install atb-sdk`
- TypeScript SDK: `npm install @pcguest/atb-sdk`

The Python and TypeScript packages are SDKs. Their `atb` wrapper expects a local ATB CLI binary or an `ATB_BIN` override.
