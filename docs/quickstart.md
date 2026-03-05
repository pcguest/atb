# ATB Quick Start

ATB gives you tamper-evident, replayable traces for AI and agent workflows.

## 1. Install

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
pip install atb-sdk

# Local source install
cd sdk/python
python3 -m venv venv
source venv/bin/activate
pip install -e .
```

## 2. Trace Your First Agent Run

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

## 3. View the Trace

```bash
# Open local timeline viewer (new in v1.0.1)
./atb view my-trace.atb --port 8080

# Default bundle path is ./run.atb/bundle.atb
./atb view
```

The viewer shows:

- hash-chain verification status
- gate status badge (PASS/FAIL/UNKNOWN)
- event timeline cards
- expandable event JSON + hash details

## 4. Use CLI-Only Flow

```bash
./atb init
./atb append dev.session --data '{"focus":"quickstart"}'
./atb snapshot build --gate pass
./atb verify
./atb view
```

## 5. Coming Soon (v1.1)

```bash
# Planned cloud sharing workflow
atb push my-trace.atb --share
```

`atb push` is intentionally not shipped in v1.0.x to keep the core local-first.
