# WORM/S3 export

> `atb push` uploads a bundle to an S3 or S3-compatible endpoint.
> Full behaviour and limits are specified in [`docs/spec/bundle-push.md`](../spec/bundle-push.md).
>
> **WORM boundary:** ATB requests lock headers for S3 Object Lock, but it does not enforce WORM itself. Enforcement depends entirely on the bucket's Object Lock and retention configuration. If the bucket is not configured correctly, the uploaded bundle is not immutable. For regulated deployments, pair ATB with filesystem integrity monitoring and a correctly configured WORM-capable store.

## How this actually behaves

Local capture → local bundle → optional push to S3 Object Lock. Three stages:

1. **Local bundle**: events are appended to a local `.atb` file and hash-chained. `atb verify` detects any tampering at any point.
2. **Push** (`atb push s3://bucket/prefix`): the sealed bundle is uploaded as a single content-addressed object (`sha256-<head-hash>.atb`). The local bundle is not modified by the export.
3. **WORM enforcement**: S3 Object Lock COMPLIANCE mode and the bucket retention policy prevent the uploaded object from being overwritten or deleted. ATB requests the lock header (`x-amz-object-lock-mode: COMPLIANCE`); the bucket configuration enforces it.

ATB does not enforce WORM. All WORM guarantees live in the storage layer.
Do not rely on `--lock-until` alone. If Object Lock is absent, misconfigured, or set to a weaker mode than required, the storage layer remains mutable.

`atb push` uploads the bundle object. It does not upload the compliance
export zip, and it does not make local exports immutable by itself.

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

## CLI interface

```bash
atb push s3://bucket/prefix [--lock-until YYYY-MM-DD] [--bundle path/to/bundle.atb]
```

The bundle will be written as a single object with a content-addressed key:

```
s3://bucket/prefix/sha256-<bundle-head-hash>.atb
```

When `--lock-until` is supplied and the bucket has Object Lock enabled in COMPLIANCE mode, ATB will set:

- `x-amz-object-lock-mode: COMPLIANCE`
- `x-amz-object-lock-retain-until-date: <lock-until>T00:00:00Z`

ATB does not enforce WORM — the bucket policy enforces WORM. ATB requests the lock; whether it is applied depends on bucket-level configuration. ATB does not override or validate the bucket's retention policy.

Remote verification after push:

```bash
atb verify --remote s3://bucket/prefix/sha256-<hash>.atb
```

## Other WORM-capable targets

Azure Blob Storage immutable storage and Google Cloud Storage object lock are under consideration as equivalent targets. Azure and GCS support depend on demand and implementation capacity.

| Target | Object lock mechanism | Lock mode equivalent |
| --- | --- | --- |
| AWS S3 | Object Lock | COMPLIANCE or GOVERNANCE |
| Azure Blob | Immutable storage (container or blob level) | Legal Hold or Time-Based Retention |
| Google Cloud Storage | Object retention locks | Retention policy or object hold |
