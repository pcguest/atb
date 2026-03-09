# Changelog

All notable changes to ATB are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Release infrastructure for v1.0.0 preparation (workflow, Docker packaging, release preflight script)
- Launch/versioning docs (`docs/spec-launch.md`, `VERSIONING.md`)

### Changed

- TypeScript lockfile regenerated to resolve `npm ci` lock mismatch failures

## [Phase 6] Dashboard + Viewer APIs

### Added

- Local-first visual dashboard route (`/view`) with verification banner
- Viewer API endpoints for verification, metadata, events, graph, and privacy reveal
- Privacy reveal audit log support via `--log-reveals`

### Changed

- `atb view` now serves dashboard assets when available and enforces verify-first data gating

## [Phase 5] AI Integration Middleware

### Added

- Python LangChain callback middleware
- TypeScript Vercel AI middleware
- AI trace mapping for chain/LLM/tool lifecycle events

## [Phase 4] Compliance Exports

### Added

- SOC2 and GDPR export formats in CLI
- Deterministic export fixtures and format dispatching

## [Phase 3] Retention + Archive

### Added

- Retention configuration, archive flow, and compliance export foundation
- Hash-chain protected archival behavior

## [Phase 2] Event Schema Expansion

### Added

- Optional actor/org/workspace identifiers with backward compatibility

## [Phase 1] Encryption Foundation

### Added

- Encryption and canonical hash-chain foundation across Go/Python/TypeScript
