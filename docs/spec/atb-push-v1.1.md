# atb push — Cloud Sharing Spec (v1.1)

Status: Draft for Week 2 design review

## Goal

Allow optional cloud sharing of ATB bundles without changing the local-first default.

## Command

```bash
atb push <bundle_path> [--share] [--password <pw>] [--expiry <24h|7d|30d|never>]
```

## Non-Negotiables

- Core tracing remains local-first and offline-capable.
- Every push validates integrity before upload.
- Encryption is client-side (zero-knowledge for server operators).

## High-Level Flow

1. Resolve and load bundle path.
2. Verify hash chain (`atb verify` equivalent).
3. Derive key from password using Argon2id (salt per upload).
4. Encrypt bundle bytes with AES-256-GCM.
5. Upload encrypted blob to Cloudflare R2 (`atb-traces-prod`).
6. Generate signed link with expiry.
7. Return URL plus local recovery metadata.

## Security Model

- Password is never sent to server in plaintext.
- Optional password fragment lives in URL hash (`#...`) to avoid server logs.
- Blob objects are unreadable without client-side key derivation.
- Signed URLs are short-lived by default (`24h`).

## Metadata Schema (Server-Side)

```json
{
  "id": "trace_abc123",
  "created_at": "2026-03-03T00:00:00Z",
  "expires_at": "2026-03-10T00:00:00Z",
  "cipher": "AES-256-GCM",
  "kdf": "Argon2id",
  "salt_b64": "...",
  "nonce_b64": "...",
  "bundle_hash": "sha256:...",
  "size_bytes": 12345
}
```

## CLI UX

```bash
$ atb push run.atb/bundle.atb --share --expiry 7d
✓ Bundle verified: 24 events, chain intact.
✓ Uploaded encrypted bundle (12.1 KB)
Share URL: https://atb.dev/share/trace_abc123#k=...
Expires: 2026-03-10T12:00:00Z
```

## Failure Modes

- Verification fails: abort upload.
- R2 upload fails: no URL returned, local data unchanged.
- Invalid expiry: reject with usage guidance.

## Out of Scope (v1.1)

- Team workspaces / RBAC
- Usage analytics beyond aggregate error counts
- Organization billing and quotas

## v1.2+ Extension Path

- Team workspaces via Supabase metadata
- Webhook notifications on upload/download
- Optional revocation list for issued links
