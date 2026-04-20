# Push transports

## S3 WORM

`atb push` can upload a sealed bundle to an S3 or S3-compatible object store.
When `--lock-until` is supplied, ATB sets these headers on every PUT:

- `x-amz-object-lock-mode: COMPLIANCE`
- `x-amz-object-lock-retain-until-date: <RFC3339 timestamp>`

These headers ask the storage layer to apply Object Lock retention. The guarantee
comes from the bucket configuration and the storage provider's enforcement. ATB
does not verify server-side lock enforcement after upload, and it does not prove
that the remote bucket was configured correctly.

## Queue gateway

`atb push --queue <endpoint-url> --hmac-key <hex-key>` publishes a JSON envelope
to an HTTP receiver:

```json
{
  "bundle_id": "0123456789abcdef0123456789abcdef",
  "digest": "f0c1...",
  "seal_timestamp": "2026-04-20T03:05:06Z",
  "profile_id": "atb.profile.privileged_tool_action",
  "atb_version": "1.9.0"
}
```

ATB signs the exact JSON request body with HMAC-SHA256 and sends the lowercase
hex digest in `X-ATB-Signature`.

Recommended receiver validation:

- Require HTTPS.
- Recompute HMAC-SHA256 over the received request body and compare it with
  `X-ATB-Signature`.
- Reject bodies with missing or malformed `bundle_id`, `digest`, or
  `seal_timestamp`.
- Treat `profile_id` as descriptive metadata, not proof of workflow coverage.
- Store or forward the envelope only after signature verification succeeds.

## Security boundary

ATB proves non-alteration of the bundle that was recorded locally. It does not
prove transport security. TLS, IAM policy, queue ACLs, endpoint authentication,
replay protection, and remote retention policy remain the operator's
responsibility.
