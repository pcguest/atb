# ATB Versioning Policy

ATB uses Semantic Versioning (`MAJOR.MINOR.PATCH`) for the CLI, SDKs, and release artifacts.

## What triggers a major version

A MAJOR bump is required for any breaking change:

- CLI behavior incompatible with previous stable usage
- Event schema or canonicalization incompatibility
- Breaking change to SOC2/GDPR export contract fields
- Breaking public API changes in Python/TypeScript SDKs

## What triggers a minor version

A MINOR bump is used for backward-compatible feature additions:

- New commands
- New integrations and middleware
- New export/report formats
- New dashboard capabilities

## What triggers a patch version

A PATCH bump is used for backward-compatible fixes:

- Bug fixes
- Security patches
- Documentation and release tooling fixes

## Deprecation policy

- Deprecations are announced in changelog and release notes
- Deprecated behavior remains available for at least one subsequent MINOR release
- Deprecated behavior is only removed in a later MAJOR release

## Tagging rules

- Release tags are annotated and formatted as `vMAJOR.MINOR.PATCH`
- CI publish workflows trigger only on tags matching `v*.*.*`
- Tag version must match package versions validated in release workflow

## Pre-release versioning

- Pre-release versions use SemVer pre-release suffixes for release tags and npm, and PEP 440 pre-release forms for Python
- Example forms: `v1.5.0-rc.1` for the release tag, `1.5.0-rc.1` for npm, and `1.5.0rc1` for Python
