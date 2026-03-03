# atb push MVP Implementation Checklist

## Pre-Implementation

- [ ] Validate demand (3+ users request cloud sharing)
- [ ] Confirm Cloudflare R2 free tier limits (10GB storage, 10M ops/month)
- [ ] Finalize key derivation strategy (Argon2id + per-upload salt)
- [ ] Finalize URL format (`https://atb.dev/share/<id>#<password_fragment>`)

## Core Implementation

- [ ] Add `internal/crypto` package
- [ ] Implement AES-256-GCM encrypt/decrypt
- [ ] Implement key derivation (password + salt -> key)
- [ ] Add `atb push <bundle>` command
- [ ] Add `atb pull <share_url>` command
- [ ] Add R2 upload/download adapter
- [ ] Generate share links with configurable expiry

## Security

- [ ] Password never sent to server
- [ ] URL fragment (`#...`) used for client-only secret material
- [ ] Salt stored with encrypted payload
- [ ] Download endpoint has rate limiting and abuse guardrails

## Testing

- [ ] Unit tests for encrypt/decrypt and KDF
- [ ] Integration test: push -> pull -> verify hash chain
- [ ] Negative tests: wrong password, expired link, tampered payload
- [ ] Security review checklist completed

## Documentation

- [ ] Update README with `atb push` examples
- [ ] Add `/docs/security.md` FAQ section for cloud sharing
- [ ] Update quickstart with optional cloud-sharing flow

## Launch

- [ ] Beta with 5 trusted users
- [ ] Collect UX + security feedback
- [ ] Iterate based on user pull
- [ ] Ship in `v1.1.0`
