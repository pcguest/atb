# AI Integration Guide

This guide defines the stable CLI and JSON contract for automated ATB integrations in `0.9.2-beta`.

## CLI Contract

### Machine-readable commands

- `atb verify --format json`
- `atb trust-report --format json`
- `atb init --format json`
- `atb append ... --format json`
- `atb snapshot ... --format json`
- `atb verify --trace` (debug hash-step logging to stderr)
- `atb help --format json` (machine-discoverable command contract)

If a command does not support `--format json`, treat its output as human-only text.

For JSON-enabled mutating commands (`init`, `append`, `snapshot`), both success and error responses use a JSON envelope:

- `status`: `ok|error`
- `action`: operation identifier (`init`, `append`, `snapshot`, `preview_append`, `preview_snapshot`, `noop`)
- `dry_run`: boolean
- `path`: target bundle path
- `message` (success) or `error` (failure)
- `exit_code` (on failure)

For verification workflows, prefer `atb verify --format json`. That command returns the structured `VerifierReport` contract. `atb verify --json` emits the internal verification report shape and should be treated as a diagnostic surface rather than the stable automation contract.

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
- `3`: profile verification failure or system/runtime error

Automation should always gate on non-zero exit codes and parse JSON only when `--format json` is explicitly passed.

### Safe automation pattern

1. Run from the directory that holds your local ATB evidence, or pass explicit bundle paths.
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
  "type": "atb.snapshot",
  "data": {
    "name": "build_complete"
  }
}
```

## Verifier Report Schema

`atb verify --format json` returns:

- `bundle_path`: bundle file path used for evaluation
- `profile_id`: canonical profile id in the form `atb.profile.<name>` when a profile matched or was selected
- `pass`: whether the evaluated profile passed
- `cas_score` (optional): completeness assurance score when the profile supports CAS
- `cas_grade` (optional): `High|Medium|Low|Insufficient`
- `sub_scores` (optional): CAS sub-score map keyed by `EC`, `FC`, `RC`, `TC`, `SC`, `XC`, `AC`, `GC`
- `critical_failures[]`: blocking failures, each with `kind` and `detail`
- `required_warnings[]`: non-blocking required warnings
- `informational_notes[]`: informational notes from verification
- `exclusions[]` (optional): declared blind spots for the matched profile
- `residual_risk`: `Low|Medium|High|Critical`

## Trust Report Schema

`atb trust-report --format json` returns:

- `bundle_path`: bundle file path used for evaluation
- `profile_id`: matched or explicitly selected canonical profile id in the form `atb.profile.<name>` when present
- `workflow_class`: workflow class from the evaluated profile when present
- `pass`: whether the evaluated profile passed
- `cas_score` (optional): completeness assurance score when the profile supports CAS
- `cas_grade` (optional): `High|Medium|Low|Insufficient`
- `residual_risk`: `Low|Medium|High|Critical`
- `chain`: hash-chain summary object
- `anchoring`: TSA anchoring summary object
- `sections[]`: profile-specific evidence sections
- `warnings[]` (optional): required warnings from the evaluated profile
- `blind_spots[]` (optional): declared profile blind spots

Chain object:

- `valid`: whether hash-chain verification succeeded
- `record_count`: number of records in the bundle
- `first_seq`: first observed sequence number
- `last_seq`: last observed sequence number
- `canonicalisation`: currently `rfc8785`
- `hash_algo`: currently `sha256`

Anchoring object:

- `present`: whether an anchor record is present
- `tsa_verified`: whether TSA verification succeeded
- `anchor_hash` (optional): bundle or TSR hash surfaced from the anchor payload

Section object:

- `title`: evidence section label
- `pass`: whether that section's evidence checks passed
- `fields` (optional): string fields surfaced for review
- `notes` (optional): section notes

The command exits `0` when `pass` is `true`, and `1` otherwise.

## CI Assertion Examples

```bash
atb verify --format json --profile atb.profile.rag_answer > verify-report.json
jq -e '.pass == true' verify-report.json
jq -e '.critical_failures | length == 0' verify-report.json
atb trust-report --format json --profile atb.profile.rag_answer > trust-report.json
jq -e '.pass == true' trust-report.json
jq -e '.chain.valid == true' trust-report.json
```

## AI Self-Audit Loop

1. Load existing bundle and run `atb verify --format json`.
2. Apply code/documentation changes.
3. Re-run `atb verify --format json` and compare `pass`, `critical_failures`, `required_warnings`, and `cas_score` where applicable.
4. Run `atb trust-report --format json` and block if the trust report no longer passes.
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

- Wire-format version and PBKDF2 iterations:
  - `0x01`: `100000` iterations for legacy decrypt compatibility
  - `0x02`: `600000` iterations for newly encrypted payloads
- Hash algorithm (must be `SHA-256`)
- AES-GCM tag size (must be `16` bytes)
- Wire format byte order and field ordering
