# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Added root `SECURITY.md` with responsible disclosure process and 48-hour response target.
- Added root `CODE_OF_CONDUCT.md` (Contributor Covenant adaptation).
- Added security and operations docs:
  - `docs/security.md`
  - `docs/config.md`
  - `docs/incident-response.md`
  - `docs/maintenance/disaster-recovery.md`
- Added `.github/ISSUE_TEMPLATE/security_report.yml` for structured vulnerability intake.

### Changed

- Updated `README.md` to reference security docs and corrected TypeScript package import to `@pcguest/atb-sdk`.
- Updated `LAUNCH.md` with pricing tiers and corrected npm package name reference.
- Updated `.gitignore` to explicitly ignore `run.atb/*.atb`.
- Updated maintenance checklist with quarterly disaster recovery drill task.

### Removed

- Removed stale backup artifact `cmd/atb/main.go.bak`.
- Removed pending-work marker from waitlist component placeholder logic.
