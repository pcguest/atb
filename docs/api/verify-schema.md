# `atb verify --format json` Schema

Source of truth: `cmd/atb/main.go` (`verifyResult`).

## JSON Shape

```json
{
  "status": "valid",
  "chain_length": 42,
  "genesis_hash": "0000000000000000000000000000000000000000000000000000000000000000",
  "head_hash": "f3f53f...",
  "verified_at": "2026-03-12T01:23:45Z",
  "algorithm": "SHA-256||RFC8785",
  "path": "run.atb/bundle.atb"
}
```

## Fields

- `status` (`string`): one of `valid`, `invalid`, `empty`, `error`.
- `chain_length` (`integer`): number of records in bundle.
- `genesis_hash` (`string`): constant genesis hash (64 zero hex chars).
- `head_hash` (`string`, optional): omitted when chain is empty.
- `verified_at` (`string`): UTC RFC3339 timestamp.
- `algorithm` (`string`): currently `SHA-256||RFC8785`.
- `path` (`string`, optional in struct): bundle path used by verify command.
- `error` (`string`, optional): present for `invalid` and `error` statuses.
- `message` (`string`, optional): present for user-friendly informational states (for example `empty`).

## Status Semantics

- `valid`: integrity checks passed.
- `invalid`: hash-chain tamper/integrity failure.
- `empty`: bundle has no records.
- `error`: bundle load/parse/runtime error.
