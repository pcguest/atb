# CI/CD secrets inventory

This file lists **secret names only**, not values. All secrets are
managed via CI/CD pipeline configuration and hosting provider secret
stores. No secret value should appear in this file, in any tracked
file, or in commit history.

## Current secrets

- `GITHUB_TOKEN` — GitHub-provided token used for release automation
  (creating releases, uploading release assets). Scoped to the
  repository; expires per-job.

- `NPM_TOKEN` — Package-scoped granular publish token for
  `@pcguest/atb-sdk` on npm. It requires read/write access and bypass
  2FA because the release workflow is non-interactive.

- `ATB_SIGNING_KEY_PEM` — PKCS#8 Ed25519 private key used to sign the
  release evidence bundle before external publication. Generate it
  with `atb keygen`, retain the public key and a protected recovery
  copy of the private key, and rotate it according to
  `docs/key-management.md`.

PyPI publishing uses GitHub OIDC trusted publishing. No `PYPI_TOKEN`
repository secret is required.

## WORM/S3 export secrets

These are required when using `atb push` to upload bundles to an
S3-compatible WORM store from CI.

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
