# Bundle push — specification

> **Status: In Progress — v1.6. Minimal S3 PUT support is implemented; release hardening and CI rollout remain pending.**
> See `cmd/atb/push.go` and `internal/push/`.
>
> **WORM boundary:** ATB requests object-lock headers when `--lock-until` is used, but it does not enforce WORM at the storage layer. Enforcement depends entirely on S3 Object Lock and bucket retention configuration outside ATB's control. If the bucket is not configured correctly, the upload is not immutable. For regulated deployments, pair ATB with filesystem integrity monitoring and a correctly configured WORM-capable store.

## Problem statement

ATB bundles are sealed local artefacts with a SHA-256 hash chain. Local tamper-evidence is strong: if any record is modified after the bundle is written, `atb verify` detects it. Local storage alone cannot defend against scenarios where the local filesystem itself is under adversarial control — a compromised host, a shared volume, or a post-incident environment where the operator who needs to provide evidence could also be the subject of the investigation.

WORM (Write Once Read Many) object storage addresses the local-control gap. Once a bundle is committed to a bucket with object lock enabled, the object cannot be overwritten or deleted for the lock duration. The ATB integrity guarantee (hash chain) and the storage layer's WORM guarantee together cover the full trust surface.

`atb push` exports a sealed bundle to a configurable external target without modifying the local bundle. The local hash chain remains the primary integrity primitive for the file on disk.

## CLI interface

```
atb push <s3://bucket/prefix> [--bundle <path>] [--lock-until YYYY-MM-DD] [--dry-run] [--format text|json]
```

### Positional argument

| Argument | Description |
| --- | --- |
| `s3://bucket/prefix` | Destination URI. Only the `s3://` scheme is committed for v1.6. Azure (`az://`) and GCS (`gcs://`) are under consideration for later milestones. |

### Flags

| Flag | Description |
| --- | --- |
| `--bundle <path>` | Bundle to push. Defaults to `run.atb/bundle.atb` (same convention as `atb verify`). |
| `--lock-until YYYY-MM-DD` | Object lock retain-until date. Requires the target bucket to have S3 Object Lock enabled in COMPLIANCE mode. ATB requests the lock header; the bucket policy enforces WORM. |
| `--dry-run` | Validate arguments and resolve the object key without uploading. |
| `--format text\|json` | Output format. JSON output follows the `pushResult` schema defined below. |

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success: bundle uploaded and, if `--lock-until` was supplied, lock header accepted. |
| `1` | User/input error: bad arguments, missing target, unreachable bundle. |
| `3` | System error: upload failure, credential error, or bucket configuration problem. |

## Content-addressed key scheme

The bundle is written as a single object with a content-addressed key:

```
<prefix>/sha256-<bundle-head-hash>.atb
```

The bundle head hash is the SHA-256 hash of the last record in the chain at push time. Using the head hash as the key makes the object address a commitment to the bundle's exact contents at push time. If the bundle is re-pushed after new events are appended, a new object with a different key is created; the prior object is not overwritten.

Example:

```
s3://my-audit-bucket/atb-prod/sha256-4f2a9b...c3e1.atb
```

## WORM lock headers

When `--lock-until` is supplied and the target bucket has S3 Object Lock enabled in COMPLIANCE mode, ATB sets the following headers on the PUT request:

```
x-amz-object-lock-mode: COMPLIANCE
x-amz-object-lock-retain-until-date: <YYYY-MM-DD>T00:00:00Z
```

ATB does not enforce WORM. S3 Object Lock COMPLIANCE mode and the bucket policy enforce WORM. ATB requests the lock; whether it is applied and honoured depends entirely on bucket-level configuration outside ATB's control.

Do not treat `--lock-until` as proof that WORM is active. If Object Lock or retention policy configuration is missing or incorrect, the storage layer may still permit overwrite or deletion.

GOVERNANCE mode is not an acceptable substitute for regulated audit evidence where the operator is also the subject: privileged users can remove GOVERNANCE locks. Use COMPLIANCE mode.

## Trust model

After a successful push:

- The uploaded object is tamper-evident for the lock duration, enforced by S3.
- The local hash chain remains the primary integrity primitive; the push does not alter or replace it.
- ATB cannot verify that the push reached the target without error unless S3 returns HTTP 200 with an ETag. Network failures between write and confirmation are reported as errors, not silent successes.

ATB does not claim that using S3 WORM with ATB satisfies any specific regulation. S3 WORM supports the "tamper-resistant storage" pattern called for by frameworks such as EU AI Act Article 12, SOC 2, and ISO 42001; whether a specific deployment satisfies those frameworks depends on the broader system design and correct bucket-side enforcement.

## JSON output schema (`pushResult`)

```json
{
  "status": "ok | error",
  "action": "push",
  "dry_run": false,
  "target": "s3://bucket/prefix",
  "bundle_path": "run.atb/bundle.atb",
  "bundle_hash": "4f2a9b...c3e1",
  "object_key": "prefix/sha256-4f2a9b...c3e1.atb",
  "lock_until": "2028-01-01",
  "message": "bundle pushed",
  "error": "",
  "exit_code": 0
}
```

`bundle_hash` and `object_key` are omitted on error or in dry-run mode where the push did not complete.

## Remote verification

After push, a stored bundle can be verified without downloading it in full:

```bash
atb verify --remote s3://bucket/prefix/sha256-<hash>.atb
```

The hash embedded in the object key is checked against the on-the-fly computed head hash during the streaming verify pass. `--remote` is planned as part of the `atb push` implementation milestone.

## AWS credentials

`atb push` uses the standard AWS credential chain: environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`), `~/.aws/credentials`, instance metadata, or ECS task role. No ATB-specific credential configuration is planned.

For CI/CD, use the secrets `ATB_WORM_S3_ACCESS_KEY_ID` and `ATB_WORM_S3_SECRET_ACCESS_KEY` as documented in `docs/release/secrets.md`. Map these to the standard AWS credential environment variables in the workflow.

Minimum IAM permissions: `s3:PutObject`, `s3:PutObjectRetention`.

## Current workaround

Until `atb push` is implemented, the equivalent operation using the AWS CLI is:

```bash
HASH=$(atb status --hash)
atb snapshot pre_worm_export
atb export --format bundle --output /tmp/sha256-${HASH}.atb

aws s3 cp /tmp/sha256-${HASH}.atb \
  s3://your-audit-bucket/atb/sha256-${HASH}.atb \
  --object-lock-mode COMPLIANCE \
  --object-lock-retain-until-date "2028-01-01T00:00:00Z"

rm /tmp/sha256-${HASH}.atb
```

The `atb snapshot pre_worm_export` call marks the bundle state before export.

## Other WORM-capable targets (post-v1.6 consideration)

| Target | Object lock mechanism | Mode |
| --- | --- | --- |
| AWS S3 | Object Lock | COMPLIANCE or GOVERNANCE |
| Azure Blob | Immutable storage | Legal Hold or Time-Based Retention |
| Google Cloud Storage | Object retention locks | Retention policy or object hold |

Only the S3 target is committed for v1.6. Azure (`az://`) and GCS (`gcs://`) support depends on demand and implementation capacity.
