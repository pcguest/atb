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

## Operating `custosd`

`custosd` (`custos/cmd/custosd`) is the reference ingest/receipt daemon — a
**transport guard, not an identity system**. Defaults are conservative; read
this before exposing it beyond a developer machine. The authoritative,
exhaustive operator checklist lives in
[`docs/custos-handoff.md`](../docs/custos-handoff.md#production-hardening-checklist).

- **Bind (loopback by default).** `--host` defaults to `127.0.0.1`, `--port` to
  `9090`. The daemon **refuses to start** on a non-loopback interface (e.g.
  `0.0.0.0`) unless `CUSTOS_AUTH_TOKEN` is set — there is no accidental
  unauthenticated network exposure.
- **Auth token.** `CUSTOS_AUTH_TOKEN` requires a bearer token on every route
  except `GET /health`, compared in constant time (`crypto/subtle`). Empty token
  is local-dev only.
- **TLS.** The daemon speaks **bare HTTP** by design. Terminate TLS at an
  operator-controlled reverse proxy when exposing it; never put bare HTTP on an
  untrusted network. Rate limiting and per-tenant identity are likewise the
  proxy's job — none are built in.
- **Token rotation.** `CUSTOS_AUTH_TOKEN` is a single static secret with no
  built-in rotation, revocation, or expiry. To rotate: issue the new token at the
  proxy, restart `custosd` with the new `CUSTOS_AUTH_TOKEN`, then retire the old
  one. There is no dual-token overlap window, so schedule rotation for a
  maintenance moment (clients see brief 401s until they present the new token).
- **Ingest limit.** `--max-ingest-bytes` (default 32 MiB) bounds the
  `POST /ingest` body at the HTTP boundary (`http.MaxBytesReader`): oversize
  uploads get **HTTP 413** before any buffering or verification, so they cannot
  exhaust memory ahead of validation. Verify-before-persist ordering means only
  hash-chain-valid bundles reach the WORM store.
- **Storage.** `--worm-dir` / `--receipt-dir` (default under `~/.atb/custos/`)
  select the filesystem WORM and receipt stores; the signing key is persisted
  under the receipt directory on first run. Setting both empty selects in-memory
  stores (tests only); a half-configured pair is rejected at startup.

## Scaffold (interfaces / TODOs only)

`discovery`, `onboarding`, `oversight`, `insights` — type and interface stubs
for the planned enterprise workflows; not yet implemented. (`registry` is now
implemented as the receipt + digest index, surfaced via `GET /receipts/by-hash`.)
