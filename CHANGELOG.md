# Changelog

All notable changes to ATB will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Removed the deprecated `internal/viewer` package by moving the legacy `atb view` HTML renderer into the CLI package.
- Moved planned web testing follow-up items out of `web/README.md` into `docs/roadmap/web-testing.md`.
- Clarified that legacy `langchain.*` event types remain for backward compatibility while new integrations must use the Phase 5 `ai.*` taxonomy.
- Trivy scans moved to weekly schedule; security-scan.yml removed (duplicate).
- chore: schemas/event.v1.json updated to match implementation
- feat: timestamp, trace_id, span_id, parent_span_id added as integrity-protected canonical event fields
- feat: timestamp auto-populated (RFC 3339 UTC) on all new bundle events
- feat: OTel trace_id, span_id wired from span context where available
- chore(web): ESLint upgraded to v9, migrated to flat config format
- note: events carrying these fields written before this change will produce different hashes; events without them are unaffected

### Fixed
- Added `genesis_hash` and `verified_at` to `GET /api/v1/bundle/meta`.
- Updated Cypress mock fixtures to use the current runtime AI event type names.
- Replaced unsigned export signature placeholders with `null` plus explicit `signature_status` fields.
- Added Python SDK deprecation guidance for `atb.integrations.langchain`.
- Filled security findings log placeholders with the verified resolving commit metadata.
- fix: RFC 8785 number serialisation corrected for floats >= 1e21 or < 1e-6
- fix: hash_algo field added to canonical event (always "sha256" in v1)
- fix: event type validated against dot-namespace pattern on append

## [v1.1.0] - 2026-03-12

### Added
- **Trust Dashboard UI**: Embedded React dashboard for local review
- **Role-based views**: Engineer (debug), Auditor (compliance), Executive (summary)
- **Trust Score**: 0-100 algorithmic integrity metric with visual radial display
- **Real-time polling**: New blocks appear in UI within 5 seconds
- **Interactive hash graph**: Recharts-based visualization of chain integrity
- **Privacy reveal endpoint**: `/api/v1/privacy/reveal` with auth + rate limiting
- **Hash-chained audit logging**: All privacy reveals logged to bundle.atb
- **CSP headers**: Strict Content-Security-Policy on embedded UI
- **PII masking**: Configurable via `docs/compliance/pii-fields.json`
- **OpenAPI generation**: `atb doc gen-openapi` command
- **E2E test suite**: Cypress tests for critical UI flows
- **Performance benchmarks**: Bundle load time tests for 100/1k/10k blocks

### Security
- Added token-based auth for `/api/v1/privacy/reveal` (X-ATB-Viewer-Token header)
- Added rate limiting (10 requests/minute per token)
- Added hash-chained audit trail for privacy reveals
- Added CSP, X-Frame-Options, X-Content-Type-Options headers
- Added dependency vulnerability scanning (govulncheck, npm audit)
- Added GitHub Actions security scan workflow

### Documentation
- Added security findings log (docs/security/findings-log.md)
- Added getting started guide for v1.1.0
- Added API documentation (docs/api/openapi.yaml)
- Added release process documentation (CONTRIBUTING.md)
- Added main-only workflow documentation

### Changed
- Moved API handlers from `internal/viewer` to `pkg/api/v1` (public package)
- Updated dashboard build dependencies for security and export stability
- Updated React/ReactDOM pinned to 18.2.0 (stability)
- Changed audit log from sidecar file to bundle.atb hash chain

### Fixed
- 64KiB bundle line limit: scanner buffer not yet increased. Fix tracked for v1.1.1 — see internal/bundle/bundle.go Load().
- Fixed CSP headers not being set on embedded UI
- Fixed rate limit threshold mismatch (now 10/min exactly)
- Fixed PII masking not using configurable field list

### Known Issues
- G304 gosec findings in export.go/config.go (pre-existing, low risk)
- Lighthouse harness requires Puppeteer/Chrome (Docker workaround available)
- Cypress E2E may be unstable in some environments (Docker runner recommended)

### Deprecated
- `internal/viewer` package removed (superseded by `pkg/api/v1`)
- Version-tag pinned GitHub Actions (use SHA pins)

## [v1.0.3] - release date not reconstructed in this repository snapshot

### Added
- Initial production release
- Cryptographically verifiable audit trail (SHA-256 + RFC8785)
- Basic CLI commands (init, append, verify, export, view)
- SOC2/GDPR evidence export (ZIP manifests)
- Local-first storage (NDJSON bundle files)

---

**For full release notes**, see `docs/releases/`.
