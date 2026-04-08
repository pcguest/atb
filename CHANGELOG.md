# Changelog

All notable changes to ATB will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Roadmap and versioning narrative doc (`docs/roadmap.md`) explaining the reset from v1.x to v0.9.0-beta.
- Obligation profile and CAS documentation (`docs/profiles.md`).
- SIEM and GRC integration documentation (`docs/integrations/siem-grc.md`).
- Strengthened adversarial test coverage for bundle parsing and dashboard APIs.
- Key management guidance for bundle/policy signing and encryption operations (`docs/key-management.md`).

## [v0.9.1-beta] — 2026-04-08

### Added
- `atb trust-report --format json` TrustReport output with profile-specific evidence sections for all six built-in profiles
- `atb export --format compliance --json` ComplianceManifest output
- MCP integration guide (`docs/integrations/mcp.md`)
- Event type constants for Python and TypeScript SDKs
- AGENTS.md repo orientation and working standards

### Fixed
- `background_automation` profile migrated to `ai.job.*` taxonomy; template, verifier, and auto-detection all consistent
- ResidualRisk now set to Critical when chain integrity fails
- TSA and CAS capability claims corrected in README and security.md

### Tests
- Snapshot tests for `atb verify --format json` across all six profiles
- Snapshot tests for `atb trust-report --format json` across all six profiles, including negative and edge cases
- Golden fixture for compliance manifest JSON output

## [v0.9.1-beta] — 2026-04-07

### Changed
- Align release metadata across CLI version output, README badges/status, and SECURITY supported-version table.

## [v0.9.0-beta] — 2026-04-XX

### Changed
- Versioning reset to v0.9.0-beta to accurately reflect pre-production status
- TSA verification: certificate chain validation is implemented and used for CAS scoring
- Bundle-level Ed25519 signing: fully implemented via `atb sign` and `atb verify`

## [v1.8.1] - 2026-04-02

### Changed
- `atb trust-report --format text --profile <id>` now surfaces profile
  obligation failures (`missing_event`, `temporal_violation`) directly in text
  output instead of hiding them behind a category summary.

### Added
- Unit and integration test coverage for `required_when` temporal DSL
  evaluation and schema-driven `profileSupportsCAS` / `computeSC`.

### Notes
- `atb trust-report` without `--profile` remains profile-agnostic by design.
- `atb keygen` must be run before `--sign-policy` can be used; `atb init`
  does not generate a keypair.

## [v1.8.0] - 2026-04-01

### Added
- Profile DSL v1: `required_when` temporal conditions on optional event rules.
  When a condition event is present, the target event is required; if `at_or_after`
  is set, target must have a timestamp at or after the condition event's timestamp.
  Hard violations produce `CriticalFailures`; missing or unparseable timestamps
  produce `RequiredWarnings`.
- `ProfileSchema.SupportsCAS` and `ProfileSchema.SCMode` replace the concrete-type
  switch in `profileSupportsCAS` and the hard-coded profile ID list in `computeSC`.
  CAS support and SC mode are now declared in each profile's YAML template.
- Ed25519 source signatures on `ai.policy.decision` events via `--sign-policy`
  flag. `atb verify` and `atb trust-report` surface a verified/absent/failed note
  per policy decision record.
- `internal/profiles.HasSchema` exported helper for schema presence checks.

### Changed
- `privileged_tool_action` and `data_export` YAML templates: `ai.human.approval`
  is now a `required_when` obligation triggered by `ai.action.executed`, with
  `at_or_after` ordering enforced.
- `rag_answer` YAML template: `supports_cas: true`, `sc_mode: retrieval_executed`.
- SDK version parity: `sdk/python` and `sdk/typescript` bumped to `1.8.0`.

## [v1.7.0] - 2026-04-01

### Added
- Obligation-Profile DSL v1: all six built-in profiles are now defined in YAML
  (`internal/profiles/templates/`). Each file is a `ProfileSchema` with
  required events, optional events, relation rules, weights, and blind spots.
- `internal/profiles` package: `ProfileSchema`, `EventRule`, `RelationRule`,
  `ValidateSchema`, `Evaluate`, and `loadSchema` (go:embed backed).
- Generic walker (`profiles.Evaluate`) replaces all per-profile inline
  evaluation logic. The verifier requires no changes.

### Changed
- `internal/verify/profiles.go`: all 6 profile structs delegate `Evaluate`,
  `DefaultWeights`, `WorkflowClass`, and `BlindSpots` to the YAML-backed schema
  path. The `Profile` interface is unchanged.

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
