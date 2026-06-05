# Custos

Custos is the custody and attestation layer for ATB evidence: it ingests `.atb`
bundles, holds them under WORM storage, and issues independently verifiable
receipts. It is a separate Go module from the ATB core runtime.

## Implemented (`custosd`)

- `POST /ingest` — verify a submitted bundle's hash chain (via `pkg/custody`),
  store it content-addressed in WORM, and return a receipt.
- **Signed custody receipts** — each receipt carries an Ed25519 `attestation`
  (`internal/receipt.Signer`) proving Custos received bundle `<hash>` at
  `<submitted_at>`. Verify with `receipt.VerifyAttestation` against the embedded
  public key — no trust in the receipt store required. The signing key is
  generated and persisted under the receipt directory on first run.
- `GET /receipts/{id}` — fetch a stored receipt (including its attestation).
- `GET /receipts/{id}/verify` — re-verify the stored bundle's hash chain.
- `GET /receipts/by-hash?bundle_hash=<hash>` — the digest-keyed reverse lookup:
  every receipt custodying a given bundle hash. Backed by `registry` (the
  receipt + digest index), built from the receipt store. Auth-gated like
  `GET /receipts`.
- WORM (`FileSystemWORMStore`) and receipt (`FileSystemReceiptStore`) stores,
  content-addressed and idempotent by SHA-256.
- Bearer-token transport auth (a guard, **not** an identity system — see
  `docs/custos-handoff.md` for the hardening checklist before any non-local
  exposure).

## Scaffold (interfaces / TODOs only)

`discovery`, `onboarding`, `oversight`, `insights` — type and interface stubs
for the planned enterprise workflows; not yet implemented. (`registry` is now
implemented as the receipt + digest index, surfaced via `GET /receipts/by-hash`.)
