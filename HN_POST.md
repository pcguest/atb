# Hacker News Draft — Show HN: ATB (Agent Trace Bundle)

Title: Show HN: ATB — tamper-evident audit trails for AI agents (local-first, open source)

Hi HN,

I built ATB (Agent Trace Bundle), an open-source local-first audit trail for recording AI workflows as tamper-evident event bundles.

What it does:
- Hash-chains every event with SHA-256
- Uses RFC 8785 canonical JSON for cross-language consistency
- Stores bundles locally as NDJSON (`run.atb/`)
- Verifies integrity with a simple CLI (`atb verify`)

Why I built it:
- Agent workflows are hard to audit after the fact
- Teams often need a reviewable record, not just logs
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
- where incident review breaks down in current AI stacks
- whether customer handoff without a vendor control plane is a real pain
- where the local-first boundary should stay sharp
