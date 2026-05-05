# Push transports

## S3 WORM

For S3 push behaviour, lock semantics, and WORM trust boundaries, see [`docs/spec/bundle-push.md`](../spec/bundle-push.md).

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
