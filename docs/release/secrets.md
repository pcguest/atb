# CI/CD secrets inventory

This file lists **secret names only**, not values. All secrets are
managed via CI/CD pipeline configuration and hosting provider secret
stores. No secret value should appear in this file, in any tracked
file, or in commit history.

## Current secrets

- `GITHUB_TOKEN` — GitHub-provided token used for release automation
  (creating releases, uploading release assets). Scoped to the
  repository; expires per-job.

- `NPM_TOKEN` — Publish access token for the `atb` TypeScript SDK
  package on npm. Required for `sdk/typescript` publish step in the
  release workflow.

- `PYPI_TOKEN` — Publish access token for the `atb` Python package on
  PyPI. Required for `sdk/python` publish step in the release workflow.

## Planned secrets (v1.6 WORM export)

These will be needed when `atb push` S3 support ships. They must be
registered in CI before the v1.6 release workflow is enabled.

- `ATB_WORM_S3_ACCESS_KEY_ID` — AWS access key ID for the IAM user or
  role that writes sealed bundles to the WORM-configured S3 bucket.
  Minimum permissions: `s3:PutObject`, `s3:PutObjectRetention`.

- `ATB_WORM_S3_SECRET_ACCESS_KEY` — Corresponding AWS secret access
  key. Pair with `ATB_WORM_S3_ACCESS_KEY_ID`.

## Process

Any new secret introduced by a release or infrastructure change must
be added to this file by name before the change is merged. Include
the secret's scope (which workflow steps use it) and the minimum
required permissions or token scopes. Do not include values,
environment-specific endpoints, or account identifiers.
