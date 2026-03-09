# ATB Versioning Policy

ATB uses Semantic Versioning (`MAJOR.MINOR.PATCH`) for the CLI, SDKs, and release artifacts.

## What Triggers a Major Version

A MAJOR bump is required for any breaking change:

- CLI behavior incompatible with previous stable usage
- Event schema or canonicalization incompatibility
- Breaking change to SOC2/GDPR export contract fields
- Breaking public API changes in Python/TypeScript SDKs

## What Triggers a Minor Version

A MINOR bump is used for backward-compatible feature additions:

- New commands
- New integrations and middleware
- New export/report formats
- New dashboard capabilities

## What Triggers a Patch Version

A PATCH bump is used for backward-compatible fixes:

- Bug fixes
- Security patches
- Documentation and release tooling fixes

## Deprecation Policy

- Deprecations are announced in changelog and release notes
- Deprecated behavior remains available for at least one subsequent MINOR release
- Deprecated behavior is only removed in a later MAJOR release

## Tagging Rules

- Release tags are annotated and formatted as `vMAJOR.MINOR.PATCH`
- CI publish workflows trigger only on tags matching `v*.*.*`
- Tag version must match package versions validated in release workflow

## Pre-Release Versioning

- Pre-release versions use SemVer pre-release suffixes (for npm) and PEP 440 style for Python
- Example: `1.0.0.dev0` for Python while release candidate work is in progress
