# WORM/S3 export — design intent

> **Status: Planned — v1.6. The `atb push` command described here is not yet implemented.**

## Why WORM storage

ATB provides tamper-evidence locally: if any record in a bundle is modified after it is written, `atb verify` detects the change through the hash chain. Local tamper-evidence is strong as long as the local filesystem is trustworthy.

WORM (Write Once Read Many) object storage addresses scenarios where the local environment itself may be compromised — for example, a shared build host, a container with a writable volume mount, or a post-incident investigation where the operator providing evidence could also be the subject. Once a bundle is committed to a WORM bucket with object lock enabled, the storage layer prevents overwrite or deletion for the lock duration regardless of IAM permissions. The ATB integrity guarantee (hash chain) and the WORM storage guarantee (immutable object) together cover the full trust surface.

Operators working in regulated environments — EU AI Act Article 12, SOC 2, ISO 42001 — typically need to demonstrate that audit records were both tamper-evident at collection time and preserved without alteration thereafter. WORM storage is the conventional mechanism for the second requirement. ATB's local hash chain provides the first.

## AWS S3 Object Lock — how to configure a bucket

Object Lock must be enabled at bucket creation time. It cannot be added to an existing bucket.

```bash
aws s3api create-bucket \
  --bucket your-atb-audit-bucket \
  --region us-east-1 \
  --object-lock-enabled-for-bucket

aws s3api put-object-lock-configuration \
  --bucket your-atb-audit-bucket \
  --object-lock-configuration '{
    "ObjectLockEnabled": "Enabled",
    "Rule": {
      "DefaultRetention": {
        "Mode": "COMPLIANCE",
        "Days": 365
      }
    }
  }'
```

Use `COMPLIANCE` mode for regulated retention requirements. `GOVERNANCE` mode allows privileged users to shorten or remove the lock and is not appropriate for audit evidence where the operator is the subject.

For the retain-until date, use your jurisdiction's audit retention period. `docs/compliance/retention.md` documents ATB's local retention configuration; the WORM retain-until date should match or exceed that period.

## Planned CLI interface

> **Status: Planned — v1.6. This interface is not yet implemented.**

```bash
atb push --target s3://bucket/prefix [--lock-until YYYY-MM-DD] [--bundle path/to/bundle.atb]
```

The bundle will be written as a single object with a content-addressed key:

```
s3://bucket/prefix/sha256:<bundle-head-hash>.atb
```

When `--lock-until` is supplied and the bucket has Object Lock enabled in COMPLIANCE mode, ATB will set:

- `x-amz-object-lock-mode: COMPLIANCE`
- `x-amz-object-lock-retain-until-date: <lock-until>T00:00:00Z`

ATB does not enforce WORM — the bucket policy enforces WORM. ATB requests the lock; whether it is applied depends on bucket-level configuration.

Remote verification after push:

```bash
atb verify --remote s3://bucket/prefix/sha256:<hash>.atb
```

## Current workaround

Until `atb push` is implemented, use the AWS CLI directly:

```bash
HASH=$(atb status --hash)
atb export --format bundle --output /tmp/${HASH}.atb

aws s3 cp /tmp/${HASH}.atb \
  s3://your-atb-audit-bucket/sha256:${HASH}.atb \
  --object-lock-mode COMPLIANCE \
  --object-lock-retain-until-date "2028-01-01T00:00:00Z"

rm /tmp/${HASH}.atb
```

This achieves content-addressed WORM upload. The push event will not be recorded in the bundle because the workaround bypasses `atb push`. Record a snapshot before export to mark the bundle state:

```bash
atb snapshot pre_worm_export
atb export --format bundle --output /tmp/${HASH}.atb
```

## Other WORM-capable targets

The v1.6 plan will evaluate Azure Blob Storage immutable storage and Google Cloud Storage object lock as equivalent targets. Only the S3 target is committed for v1.6. Azure and GCS support depend on demand and implementation capacity.

| Target | Object lock mechanism | Lock mode equivalent |
| --- | --- | --- |
| AWS S3 | Object Lock | COMPLIANCE or GOVERNANCE |
| Azure Blob | Immutable storage (container or blob level) | Legal Hold or Time-Based Retention |
| Google Cloud Storage | Object retention locks | Retention policy or object hold |
