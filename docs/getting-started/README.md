# Getting Started

Start here if you are new to ATB.

## First 5 Minutes

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb init
atb view --ui-experimental
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

The Go CLI is the authoritative CLI path. The Python and TypeScript packages are SDKs only. Their installed `atb` command is a compatibility stub that prints Go CLI install guidance and will be removed in a future major release.
