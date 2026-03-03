# Hacker News Draft — Show HN: ATB (Agent Trace Bundle)

Title: Show HN: ATB — tamper-evident audit trails for AI agents (local-first, open source)

Hi HN,

I built ATB (Agent Trace Bundle), an open-source runtime format for recording AI/agent workflows as tamper-evident event bundles.

What it does:
- Hash-chains every event with SHA-256
- Uses RFC 8785 canonical JSON for cross-language consistency
- Stores bundles locally as NDJSON (`run.atb/`)
- Verifies integrity with a simple CLI (`atb verify`)

Why I built it:
- Agent workflows are hard to audit after the fact
- Teams need replayable traces for debugging and compliance
- I wanted a local-first format that works without extra infrastructure

Repo:
- https://github.com/pcguest/atb

Quick demo:
```bash
go build -o atb ./cmd/atb
./atb init
./atb append dev.session '{"goal":"ship v1"}'
./atb verify
```

I’d love feedback on:
- event schema conventions
- signing/compliance exports roadmap
- hosted viewer vs. staying strictly file-first
