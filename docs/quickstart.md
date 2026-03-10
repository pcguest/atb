# ATB Quick Start

ATB gives you tamper-evident, replayable traces for AI and agent workflows.

## 1. README Quick Start (v1.0.0)

```bash
pip install atb
atb init
atb view
```

## 2. Install Options

### Go CLI

```bash
# From source (development)
git clone https://github.com/pcguest/atb.git
cd atb
go build -o atb ./cmd/atb
./atb version

# From release (users)
# Download your OS binary from GitHub Releases.

# If installing from private GitHub source with `go install`
go env -w GOPRIVATE=github.com/pcguest/*
go install github.com/pcguest/atb/cmd/atb@latest
```

### Python SDK

```bash
# Preferred (PyPI)
pip install atb

# Local source install
cd sdk/python
python3 -m venv venv
source venv/bin/activate
pip install -e .
```

## 3. Trace Your First Agent Run

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

# Record a gate decision as a snapshot event
bundle.append("snapshot.build", {
    "gate": "pass",
    "timestamp": "2026-03-03T10:00:03Z",
})

bundle.save("my-trace.atb")
bundle.verify()  # Raises if tampered
```

## 4. View the Trace

```bash
# Open local dashboard viewer
atb view my-trace.atb --port 8080

# Default bundle path is ./run.atb/bundle.atb
atb view

# Enable privacy reveal audit logging
atb view --bundle my-trace.atb --log-reveals

# Launch the visual dashboard
atb view run.atb/bundle.atb --log-reveals
```

The viewer shows:

- hash-chain verification status
- gate status badge (PASS/FAIL/UNKNOWN)
- event timeline cards
- expandable event JSON + hash details

- [Dashboard Specification](./spec-dashboard.md)

## 5. Use CLI-Only Flow

```bash
atb init
atb append dev.session --data '{"focus":"quickstart"}'
atb snapshot build --gate pass
atb verify
atb view
```

## 6. Retention, Archive, and Compliance Export

- Local workflows are fully supported in v1.0.x.
- Set retention policy:

```bash
atb config retention --days 90
```

- Archive bundles older than policy cutoff (or pass `--before YYYY-MM-DD`):

```bash
atb archive
```

- Build an auditor-friendly evidence package:

```bash
atb export --format soc2 --bundle run.atb/bundle.atb --output soc2-evidence.zip
atb export --format gdpr --type dsr --subject-id usr_123 --bundle run.atb/bundle.atb --output gdpr-dsr.zip
atb export --format gdpr --type ropa --bundle run.atb/bundle.atb --output gdpr-ropa.zip
```

- Reference docs:
  - [Retention](./compliance/retention.md)
  - [Compliance Export Overview](./compliance/export.md)
  - [SOC 2 Export Specification](./compliance/soc2.md)
  - [GDPR Export Specification](./compliance/gdpr.md)

## 7. AI Agent Integration

Trace your LangChain or Vercel AI agents automatically:

```python
from atb.langchain_callback import ATBCallbackHandler
handler = ATBCallbackHandler(privacy_mode="hash")
llm = ChatOpenAI(callbacks=[handler])
```

- [LangChain Integration](./integrations/langchain.md)
- [Vercel AI SDK Integration](./integrations/vercel-ai.md)
