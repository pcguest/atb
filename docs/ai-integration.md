# AI Integration Guide

This guide defines a stable contract for AI agents that read, verify, and extend ATB workflows.

## CLI Contract

### Machine-readable commands

- `atb verify --format json`
- `atb trust-report --format json`

If a command does not support `--format json`, treat its output as human-only text.

### Mutation safety

Mutating commands support `--dry-run` previews with no filesystem side effects:

- `atb init --dry-run`
- `atb append ... --dry-run`
- `atb snapshot ... --dry-run`

`atb init` is idempotent by default. If the bundle already exists, it returns success with a warning message and makes no changes.

### Exit codes

- `0`: success
- `1`: user/input error
- `2`: integrity failure
- `3`: system/runtime error

Automation should always gate on non-zero exit codes and parse JSON only when `--format json` is explicitly passed.

### Safe automation pattern

1. Run from the repository root (or pass explicit bundle paths).
2. Use `--format json` for machine assertions.
3. Treat missing fields as contract drift and fail closed.
4. Never assume implied defaults in CI; pass all critical flags explicitly.

## Event Schema

Canonical event schema is at `schemas/event.v1.json`.

Example valid event:

```json
{
  "seq": 3,
  "prev_hash": "4271a98316d03e240233c0e0a759f0c8e9998523ffff985e5e6ea528871d884d",
  "type": "agent.snapshot",
  "data": {
    "gate": "pass",
    "reason": "all checks green"
  }
}
```

## Trust Report Schema (for CI)

`atb trust-report --format json` returns:

- `status`: `pass|warn|fail`
- `generated_at`: RFC3339 UTC timestamp
- `bundle_path`: bundle file path used for evaluation
- `chain_length`: event count when load succeeds
- `head_hash`: last event hash when available
- `categories[]`: category objects

Category object:

- `key`: `cryptographic_integrity|operational_safety|test_coverage|documentation`
- `title`: human-readable category title
- `status`: `pass|warn|fail`
- `checks[]`: check objects

Check object:

- `id`: stable machine identifier
- `title`: human-readable check label
- `status`: `pass|warn|fail`
- `details`: assertion explanation
- `evidence[]` (optional): file paths or bundle path

## CI Assertion Examples

```bash
atb verify --format json | jq -e '.status == "valid"'
atb trust-report --format json > trust-report.json
jq -e '.categories[] | select(.key=="cryptographic_integrity") | .status == "pass"' trust-report.json
```

## AI Self-Audit Loop

1. Load existing bundle and run `atb verify --format json`.
2. Apply code/documentation changes.
3. Re-run `atb verify --format json` and compare status + head hash behavior.
4. Run `atb trust-report --format json` and block if critical category status regresses.
5. Store report artifacts alongside CI logs for audit replay.
