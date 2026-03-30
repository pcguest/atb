# Changelog

All notable changes to ATB will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v1.2.0] - 2026-03-31

### Added
- Bundle manifest record (`atb.bundle.manifest`) written as sequence 0 on
  `New()`; includes `version`, `created_at`, and `bundle_id`.
- RFC 3161 TSA anchoring via `atb anchor`; anchor events recorded in chain as
  `atb.bundle.anchor`.
- `atb verify --with-anchor` validates anchor event against `.tsr` token.
- `timestamp`, `trace_id`, `span_id`, `parent_span_id` added as
  integrity-protected canonical event fields; `timestamp` auto-populated
  (RFC 3339 UTC) on all new bundle events.
- OTel `trace_id` and `span_id` wired from span context where available.
- Six verification profiles registered:
  `atb.profile.privileged_tool_action`, `atb.profile.rag_answer`,
  `atb.profile.policy_decision`, `atb.profile.human_override`,
  `atb.profile.background_automation`, `atb.profile.data_export`.

### Changed
- Legacy `atb view` HTML renderer moved from `internal/viewer` into the CLI
  package; `internal/viewer` package removed.
- Planned web testing follow-up items moved from `web/README.md` to
  `docs/roadmap/web-testing.md`.
- Legacy `langchain.*` event types retained for backward compatibility;
  new integrations must use the Phase 5 `ai.*` taxonomy.
- Trivy scans moved to weekly schedule; `security-scan.yml` removed (duplicate).
- ESLint upgraded to v9 with flat config format.

### Fixed
- `GET /api/v1/bundle/meta` now includes `genesis_hash` and `verified_at`.
- Cypress mock fixtures updated to current runtime AI event type names.
- Unsigned export signature placeholders replaced with `null` plus explicit
  `signature_status` fields.
- Python SDK deprecation guidance added for `atb.integrations.langchain`.
- Security findings log placeholders filled with verified resolving commit
  metadata.
- Bundle scanner buffer increased to 16 MiB in `internal/bundle/bundle.go`.
- RFC 8785 number serialisation corrected for floats ≥ 1e21 or < 1e-6.
- `hash_algo` field added to canonical event (always `"sha256"` in v1).
- Event type validated against dot-namespace pattern on append.
- `schemas/event.v1.json` updated to match implementation.

### Notes
- Events carrying `timestamp`/`trace_id`/`span_id`/`parent_span_id` written
  before this change will produce different hashes; events without them are
  unaffected.
- Bundles created before this change load correctly; `Manifest()` returns `nil`
  for legacy bundles.

## [v1.1.0] - 2026-03-12
...