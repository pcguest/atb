# ATB Quickstart

ATB provides tamper-evident, replayable traces for AI workflows.

## 1. Quick Start (Exact README Flow)

```bash
pip install atb-sdk
atb init
atb view
```

Integrity check:

```bash
atb verify
```

## 2. Installation Options

### Go CLI

```bash
go install github.com/pcguest/atb/cmd/atb@latest
```

### Python (Local Source)

```bash
cd sdk/python
python3 -m venv venv
source venv/bin/activate
pip install -e .
```

### TypeScript SDK

```bash
cd sdk/typescript
npm ci
npm run build
```

## 3. Record Your First Trace (Python)

```python
from atb import Bundle

bundle = Bundle()

bundle.append("agent.prompt", {
    "timestamp": "2026-03-03T10:00:00Z",
    "actor": "planner",
    "model": "gpt-4",
    "prompt": "Outline a blog post about AI safety",
})

bundle.append("agent.response", {
    "timestamp": "2026-03-03T10:00:02Z",
    "actor": "planner",
    "tokens": 142,
    "output": "Draft outline...",
})

bundle.append("snapshot.build", {
    "gate": "pass",
    "timestamp": "2026-03-03T10:00:03Z",
})

bundle.save("my-trace.atb")
bundle.verify()
```

## 4. Open the Visual Dashboard

```bash
# Default bundle path
atb view

# Custom bundle path
atb view my-trace.atb --port 8080

# Enable privacy reveal audit logging
atb view --bundle my-trace.atb --log-reveals
```

Dashboard details:

- [Dashboard Specification](./spec-dashboard.md)

## 5. CLI-Only Workflow

```bash
atb init
atb append dev.session --data '{"focus":"quickstart"}'
atb snapshot build --gate pass
atb verify
atb view
```

## 6. Compliance Exports

```bash
atb config retention --days 90
atb archive
atb export --format soc2 --bundle run.atb/bundle.atb --output soc2-evidence.zip
atb export --format gdpr --type dsr --subject-id usr_123 --bundle run.atb/bundle.atb --output gdpr-dsr.zip
atb export --format gdpr --type ropa --bundle run.atb/bundle.atb --output gdpr-ropa.zip
```

Reference docs:

- [Retention](./compliance/retention.md)
- [Compliance Export Overview](./compliance/export.md)
- [SOC 2 Export Specification](./compliance/soc2.md)
- [GDPR Export Specification](./compliance/gdpr.md)

## 7. AI Integrations

```python
from atb.langchain_callback import ATBCallbackHandler

handler = ATBCallbackHandler(privacy_mode="hash")
```

Integration docs:

- [LangChain Integration](./integrations/langchain.md)
- [Vercel AI SDK Integration](./integrations/vercel-ai.md)

## 8. Next Steps

- [Docs Home](./README.md)
- [ATB Specification v1.0](./spec-v1.0.md)
- [Security Model](./security.md)
- [Contributing](../CONTRIBUTING.md)
