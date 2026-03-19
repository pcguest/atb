# Getting Started

Start here if you are new to ATB.

## First 5 Minutes

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb init
atb append agent.run --data='{"workflow":"support-triage","case_id":"case-1042","severity":"sev2"}'
atb append policy.alert --data='{"check":"pii_redaction","outcome":"fail","ticket_id":"case-1042"}'
atb snapshot incident_review --gate fail
atb verify
```

## Recommended Path

1. [Incident Review Workflow](../guides/incident-review-workflow.md)
2. [Quickstart](../quickstart.md)
3. [Compliance Export Overview](../compliance/export.md)
4. [AI Integrations](../guides/README.md)

## Install Options

- Go CLI: `go install github.com/pcguest/atb/cmd/atb@latest`
- Python SDK: `pip install atb-sdk`
- TypeScript SDK: `npm install @pcguest/atb-sdk`

The Go CLI is the authoritative CLI path. The Python and TypeScript packages are SDKs only. Their installed `atb` command is a compatibility stub that prints Go CLI install guidance and will be removed in a future major release.
