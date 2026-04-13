# Bundle Push — Design Intent

> **Status: Planned — v1.6. This interface is not yet implemented.**

## Problem statement

ATB bundles are sealed local artefacts with a cryptographic hash chain. Local tamper-evidence is strong: if any record is modified after the bundle is written, `atb verify` detects it. Local storage alone cannot defend against scenarios where the local filesystem itself is under adversarial control — for example, a compromised host, a shared volume, or a post-incident environment where the operator who needs to provide evidence could also be the subject of the investigation.

WORM (Write Once Read Many) object storage addresses the local-control gap. Once a bundle is committed to a WORM bucket with object lock enabled, the object cannot be overwritten or deleted for the lock duration. The ATB trust guarantee (integrity of the recorded chain) is complemented by the WORM storage guarantee (the stored object cannot be replaced without bucket-level compromise).

`atb push` will export a sealed bundle to a configurable external target, recording the push as an auditable action so the bundle itself reflects when and where it was exported.

## Planned interface

```bash
atb push --target s3://bucket/prefix [--lock-until YYYY-MM-DD] [--bundle path/to/bundle.atb]
```

### Flags

| Flag | Description |
| --- | --- |
| `--target` | Push destination URI. Currently planned: `s3://` scheme. |
| `--lock-until` | Object lock retain-until date in `YYYY-MM-DD` format. Requires the target bucket to have Object Lock enabled in COMPLIANCE mode. |
| `--bundle` | Bundle path. Defaults to `run.atb/bundle.atb` (same convention as `atb verify`). |

### Content-addressed key

The bundle will be written as a single object with a content-addressed key:

```
<prefix>/sha256:<bundle-head-hash>.atb
```

Using the bundle's head hash as the key makes the object address a commitment to the bundle's contents at push time.

### After push: remote verification

```bash
atb verify --remote s3://bucket/sha256:<hash>.atb
```

This will verify a remotely stored bundle without downloading it in full (streaming verify). The hash in the object key is checked against the on-the-fly computed head hash as part of the verification pass.

## Trust model note

`atb push` exports the bundle to the target and records the push event in the local bundle. After export, ATB's integrity guarantee applies to the exported object — if the object was intact when pushed and the WORM lock prevents modification, the remote copy is tamper-evident for the lock duration.

ATB does not enforce WORM. The storage bucket policy enforces WORM. ATB requests the lock when `--lock-until` is supplied and the bucket supports it; whether the lock actually prevents deletion depends on bucket-level configuration outside ATB's control.

ATB cannot verify that the push reached the target without error unless the target confirms receipt (HTTP 200 with the object ETag). Network failures between write and confirmation are reported as errors, not silent successes.

## AWS S3 Object Lock — what ATB will set

When `--target` is an S3 URI and `--lock-until` is provided:

- `x-amz-object-lock-mode: COMPLIANCE` — prevents deletion or overwrite by any user, including the bucket owner, until the retain-until date.
- `x-amz-object-lock-retain-until-date: <lock-until>T00:00:00Z` — the lock expiry in RFC 3339 format.

The bucket must have Object Lock enabled at creation time. ATB does not enable Object Lock on existing buckets.

## AWS credentials

`atb push` will use the standard AWS credential chain (environment variables, `~/.aws/credentials`, instance metadata, etc.). No ATB-specific credential configuration is planned.

## Current workaround

Until `atb push` is implemented, the equivalent operation using the AWS CLI is:

```bash
HASH=$(atb status --hash)
atb export --format bundle --output /tmp/${HASH}.atb
aws s3 cp /tmp/${HASH}.atb s3://bucket/sha256:${HASH}.atb \
  --object-lock-mode COMPLIANCE \
  --object-lock-retain-until-date "2028-01-01T00:00:00Z"
```

This achieves the same object-addressed WORM upload. The push event will not be recorded in the bundle because the workaround bypasses ATB's push command.

## Other WORM-capable targets (v1.6 consideration)

The v1.6 design will be evaluated against Azure Blob Storage immutable storage and Google Cloud Storage object lock as equivalent targets. The `--target` URI scheme will distinguish between providers:

- `s3://` — AWS S3 or S3-compatible
- `az://` — Azure Blob (tentative)
- `gcs://` — Google Cloud Storage (tentative)

Only the S3 target is committed for v1.6. Azure and GCS support depend on demand and implementation capacity.
