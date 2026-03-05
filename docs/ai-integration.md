# AI Integration Guide

This guide defines a stable contract for AI agents that read, verify, and extend ATB workflows.

## CLI Contract

### Machine-readable commands

- `atb verify --format json`
- `atb trust-report --format json`
- `atb init --format json`
- `atb append ... --format json`
- `atb snapshot ... --format json`
- `atb verify --trace` (debug hash-step logging to stderr)
- `atb --help --format json` (machine-discoverable command contract)

If a command does not support `--format json`, treat its output as human-only text.

For JSON-enabled mutating commands (`init`, `append`, `snapshot`), both success and error responses use a JSON envelope:

- `status`: `ok|error`
- `action`: operation identifier (`init`, `append`, `snapshot`, preview variants)
- `dry_run`: boolean
- `path`: target bundle path
- `message` (success) or `error` (failure)
- `exit_code` (on failure)

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
- `gate`: blocking-check gate object
- `summary`: aggregate check counts
- `categories[]`: category objects

Gate object:

- `status`: `pass|fail` (blocking checks only)
- `blocking_failures`: number of failing blocking checks
- `failed_checks[]` (optional): fully qualified check ids (e.g. `cryptographic_integrity.hash_chain`)

Summary object:

- `total`: total checks
- `pass`: passing checks
- `warn`: warning checks
- `fail`: failing checks

Category object:

- `key`: `cryptographic_integrity|operational_safety|test_coverage|documentation`
- `title`: human-readable category title
- `status`: `pass|warn|fail`
- `checks[]`: check objects

Check object:

- `id`: stable machine identifier
- `title`: human-readable check label
- `status`: `pass|warn|fail`
- `severity`: `critical|advisory`
- `blocking`: whether check failure should fail the CI gate
- `details`: assertion explanation
- `evidence[]` (optional): file paths or bundle path

## CI Assertion Examples

```bash
atb verify --format json | jq -e '.status == "valid"'
atb trust-report --format json > trust-report.json
jq -e '.gate.status == "pass"' trust-report.json
jq -e '.categories[] | select(.key=="cryptographic_integrity") | .status == "pass"' trust-report.json
```

## AI Self-Audit Loop

1. Load existing bundle and run `atb verify --format json`.
2. Apply code/documentation changes.
3. Re-run `atb verify --format json` and compare status + head hash behavior.
4. Run `atb trust-report --format json` and block if critical category status regresses.
5. Store report artifacts alongside CI logs for audit replay.

## Cross-Language Encryption Testing

ATB encryption must produce byte-identical ciphertext across Go, Python, and TypeScript for the same inputs. This is verified via golden fixtures.

### Deterministic Test Path

For testing only, encryption functions accept fixed `salt` and `nonce` parameters:

```go
// Go
ciphertext, err := encrypt.EncryptWithSaltNonce(plaintext, password, salt, nonce)
```

```python
# Python
ciphertext = encrypt_raw(plaintext, password, salt=salt, nonce=nonce)
```

```ts
// TypeScript
const ciphertext = encryptRaw(plaintext, password, { salt, nonce });
```

### Adding a New SDK

To add encryption support for a new language:

1. Implement `encrypt_raw(plaintext, password, salt, nonce)` using AES-256-GCM + PBKDF2.
2. Use the exact wire format: `ATBE` + `version` + `salt(16)` + `nonce(12)` + `tag(16)` + `ciphertext`.
3. Add deterministic test path accepting fixed salt/nonce.
4. Compare output bytes against Go baseline in `test/golden/encrypt-vector.hex`.
5. All SDKs must produce identical ciphertext for same inputs.

### Golden Fixture Format

`test/golden/encrypt-vector.hex` contains hex-encoded ciphertext for:

- Canonical JSON: `{"head_hash":"0000000000000000000000000000000000000000000000000000000000000000","records":[]}`
- Password: `test-password-for-parity`
- Salt: 16 bytes of `0x42`
- Nonce: 12 bytes of `0x7A`

If your implementation produces different output, check:

- PBKDF2 iterations (must be `100000`)
- Hash algorithm (must be `SHA-256`)
- AES-GCM tag size (must be `16` bytes)
- Wire format byte order and field ordering
