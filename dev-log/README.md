# ATB Dev Log

This directory contains ATB trace bundles that record the development of ATB itself — a practice known as **dogfooding**.

Each bundle is a tamper-evident, hash-chained NDJSON file recording key development sessions, architectural decisions, and release events. These bundles serve two purposes:

1. **Testing** — They validate that the ATB format and SDK work correctly on real data.
2. **Marketing** — They demonstrate ATB's value proposition with a concrete, authentic case study.

## Bundle Index

| File | Version | Events | Description |
|------|---------|--------|-------------|
| `v0.1.0-bootstrap.atb` | v0.1.0 | 5 | Initial project bootstrap — repo setup, CLI, SDKs, CI/CD |

## Verifying a Bundle

Using the CLI:
```bash
atb verify dev-log/v0.1.0-bootstrap.atb
```

Using the Python SDK:
```python
from atb import Bundle
b = Bundle.load("dev-log/v0.1.0-bootstrap.atb")
b.verify()
print(f"Verified {len(b)} events.")
```

## Event Types Used

| Type | Description |
|------|-------------|
| `dev.session` | A development session recording features built and blockers encountered |
| `decision` | An architectural or product decision with rationale and alternatives |
| `release` | A version release event with download statistics |
