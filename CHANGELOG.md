# Changelog

All notable changes to ATB will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v1.6.0] - 2026-04-01

### Added
- `atb trust-report --format text` adds a human-readable trust report with ANSI
  status colour (PASS/FAIL/WARN) and a conditional CAS block showing profile,
  grade, anchor quality label, and all eight sub-scores.
- `atb snapshot <name>` appends an `atb.snapshot` record containing `name`,
  `bundle_hash` (SHA-256 hex of serialised bundle), `record_count`, and
  `snapshot_at` (RFC 3339 UTC). Accepts `--bundle` and `--quiet`.
- `internal/event` adds `TypeSnapshot = "atb.snapshot"`.
- `internal/verify` adds an offline RFC 3161 fixture
  (`testdata/anchor_token_verified.tsr`) and generator, and unskips
  `TestClassifyAnchor_Verified`.
- SDK version parity updates `sdk/python` and `sdk/typescript` to `1.6.0`.

### Changed
- `cmd/atb/main.go` replaces the snapshot stub with the real command in
  `snapshot.go`.
- `internal/verify/anchor_classify.go` adds a narrow root-pool hook for test
  override while leaving the production path unchanged.

## [v1.5.0] - 2026-03-31

### Features
- Add `atb bundle new` as an alias for `atb init`.

### Bug Fixes
- Guard against a `nil` `SubScores` map in verify output and add SC fallback
  handling for unmatched profiles.
- Copy all embedded source files into the Go builder Docker stage.

### CI/Chores
- Build web assets before `go vet` in the Gold Release Gate workflow.
- Install `sdk/typescript` dependencies before Gold Release Gate tests.
- Bump SDK versions to `1.4.0` for release tag parity.
- Pin GitHub Actions workflow steps to full commit SHAs.
- Update `vitest` from `4.0.18` to `4.1.2` to address the `flatted` CVE.
- Update `docker/setup-buildx-action` from `3.12.0` to `4.0.0` for Node 24.
- Update `actions/upload-artifact` from `4.6.2` to `7.0.0` for Node 24.

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
