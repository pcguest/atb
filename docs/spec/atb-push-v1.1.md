# atb push — Encrypted Handoff Exploration

Status: Exploratory only. Not scheduled for implementation or release.

## Purpose

Define the narrowest possible encrypted handoff path if repeated buyer conversations show that secure transfer is the blocker after local bundle creation and verification.

`atb push` is not part of the current shipped CLI. ATB remains focused on local verification, incident review, customer handoff, and deterministic evidence export.

## Proposed Command Shape

```bash
atb push <bundle_path> [--password <pw>] [--expiry <24h|7d|30d>]
```

## Guardrails

- Core tracing remains local-first and offline-capable.
- `atb push` must stay optional and must never become the default storage path.
- Every handoff validates integrity before transfer.
- Encryption stays client-side; any backend only stores ciphertext.
- No hosted workspaces, tenancy, RBAC, billing, or control-plane scope.
- Backend choice is an implementation detail, not a product promise.

## Proposed Flow

1. Resolve and load bundle path.
2. Verify the hash chain (`atb verify` equivalent).
3. Derive an encryption key from user-supplied secret material with a memory-hard KDF.
4. Encrypt bundle bytes with AES-256-GCM.
5. Upload ciphertext to a minimal handoff backend.
6. Return a time-bounded retrieval link plus local recovery metadata.

## Validation Gate

Only start implementation if multiple qualified buyers independently confirm that:

- local bundles are already useful
- secure transfer is the missing step
- the encrypted handoff path can stay narrow without turning ATB into a hosted platform

## Security Expectations

- Secrets are never sent to the server in plaintext.
- Stored objects are unreadable without client-side decryption material.
- Expiry should default to short-lived links.
- Retrieval metadata must be sufficient to verify and decrypt locally after download.

## Failure Modes

- Verification fails: abort handoff.
- Backend upload fails: no link returned, local data unchanged.
- Invalid expiry or missing secret material: reject with usage guidance.

## Explicitly Out of Scope

- Hosted dashboards or always-on trace storage
- Team workspaces, RBAC, or admin tooling
- Product analytics beyond operational error visibility
- Billing, quotas, or procurement surfaces
- Broad collaboration features
