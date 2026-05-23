# Changelog

All notable changes to ATB will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Docs
- Added `AGENTS.md` as the canonical maintainer and coding-agent harness and aligned the core Markdown estate around it.
- Tightened README scope and quickstart flow so current capability, non-goals, and planned work are more clearly separated.
- Aligned contributing, release, roadmap, security, and versioning docs with the current local-first viewer and release model.

### Changed
- Main branch CI and toolchain hygiene now align with the agent-safe subset from
  `audit/complete-atb`: Go 1.26.3, support-matrix drift checks, quality evidence,
  and refreshed npm lockfiles for the TypeScript SDK and web viewer.

### Added
- `atb capture run` — wrap any child command with ATB capture environment
  variables; stamps the resulting bundle with a capture run ID for provenance.
- `atb import chatlog` — import saved AI chat logs (Claude, OpenAI, generic
  JSONL) into a local ATB bundle, mapping turns onto the canonical event taxonomy.
- Capture v1 event mapping: user turns become `ai.request.received`, assistant
  turns become `ai.model.invoked` + `ai.model.output` + `ai.response.sent`,
  tool turns become `ai.tool.exec`, and system context turns are recorded as
  prompt-window context.
- `internal/evidence` package and `atb evidence --bundle <path> [--format text|json]`
  for structured local bundle evidence summaries, including manifest, snapshot, and
  per-signature provenance.
- `exitLockContention = 9` for advisory bundle lock contention so automation can
  distinguish retryable contention from general system failures.
- Python SDK local signing provenance fields (`backend`, `key_id`, `signed_at`) and
  a `verify()` signatures array matching the Go JSON shape.
- `atb.corroboration.external` event type in the new `atb.corroboration.*` namespace.
  Required fields: `source`, `reference_id`, `digest`, `retrieved_at`. Optional fields:
  `adapter`, `raw_evidence` (base64, capped at 4 KB), `truncated`. Schema locked at v1.
- HTTP gateway receipt adapter in `internal/corroboration/`. Fetches a JSON receipt from a
  configured URL, computes the SHA-256 digest of the response body, and returns a record
  ready to append as an `atb.corroboration.external` event. Raw evidence is stored up to
  4 KB; payloads larger than 4 KB set `Truncated=true` and omit the raw body.
- `atb corroborate` subcommand: fetches a receipt from an external adapter and appends an
  `atb.corroboration.external` event to the active bundle. Flags: `--source` (required),
  `--url` (required for `http-gateway`), `--ref` (required), `--bundle` (optional),
  `--dry-run`, `--format text|json`.
- Verifier awards XC sub-score credit for well-formed `atb.corroboration.external` events.
  One valid corroboration event earns XC=1.0. Bundles without corroboration events return
  their anchor-based XC score unchanged — identical to v1.9.0 behaviour.
- `internal/push/`: `Push` interface, `S3Pusher`, and `QueuePusher` for signed queue
  gateway envelopes.
- `atb push`: `--queue <endpoint-url>` and `--hmac-key <hex-key>` flags. Queue pushes
  POST a signed JSON envelope after any S3 upload completes.
- `atb push --dry-run`: previews the queue endpoint and envelope JSON as well as the
  existing S3 target resolution path.
- S3 push coverage now checks that Object Lock PUT requests carry
  `x-amz-object-lock-mode: COMPLIANCE` and
  `x-amz-object-lock-retain-until-date`.

### Fixed
- Bundle signature append now uses `writeAtomic` (temp file, fsync, rename) instead
  of truncate-in-place writes, preserving the original bundle if the final write fails.
- Stale documentation and maintenance references to the removed legacy viewer flag.

### Docs
- `docs/spec-v1.0.md` and `schemas/event.v1.json`: signature provenance fields
  documented as current optional `atb.bundle.signature` payload fields.
- `docs/spec-ai-traces.md`: `atb.corroboration.*` namespace and required field schema
  documented, corroboration event added to the complete event type registry table.
- `docs/architecture.md`: corroboration model section added, covering the problem addressed,
  what the event records, XC scoring, the trust limitation, and adapter extension points.
- All six built-in profile templates: blind-spot text updated to note XC credit conditions.
- `docs/integrations/push-transports.md`: S3 WORM headers, queue gateway envelope and
  HMAC signing, and the transport security boundary.

## [v1.10.0] - 2026-04-21

### Fixed
- Viewer events endpoint returning incorrect data
- Version string consistency across all packages (cmd/atb/main.go, SDKs, web)
- Integration test golden value updated to v1.10.0
- check-versions.sh now derives expected version from latest git tag

## [v1.9.0] - 2026-04-20

### Added
- **CAS v1 corroboration bonus**: `CorroborationPolicy` struct (`AnchorBonus` 0.05,
  `SignatureBonus` 0.03, `SnapshotBonus` 0.02, `MaxBonus` 0.10) with `Validate()` and
  `DefaultCorroborationPolicy()`. `EvaluateBundle` accepts a new `WithCorroborationPolicy`
  option; when set, `CASResult` gains `corroboration_bonus` and `effective_score` fields
  (grade derives from `effective_score`; nil policy produces output identical to v1.8.0).
  `atb verify` automatically applies the default policy when `--with-anchor` is present.
- **`--corroboration-policy <path>`** flag on `atb verify`: accepts a JSON file matching
  `CorroborationPolicy` to override the default bonus values.
- **Typed error sentinels** in `internal/verify/evaluate.go`: `ErrBundleNotFound`,
  `ErrChainInvalid` (returned when `RequireValidChain` is set and the chain fails), and
  `ErrProfileUnknown` (all supplied profiles were nil). Callers can use `errors.Is`.
- **`--policy-doc <path>`** flag on `atb append` (`ai.policy.decision` events only):
  reads the file, computes `SHA-256(contents)` hex, and embeds it as `policy_doc_hash`.
  When `--sign-policy` is also set, stores a compound Ed25519 `policy_doc_signature`
  over `SHA-256(canonical payload) || SHA-256(doc bytes)`.
- **`VerifyPolicyDocSignature`** in `internal/sign`: verifies the compound policy-doc
  signature. `policy_doc_signature_valid` boolean surfaced in `TrustReport` (nil when
  no `policy_doc_hash` present, true/false otherwise).
- **`atb version --json`**: outputs `{"version":"1.9.0","algorithm":"SHA-256+RFC8785","anchor":"RFC3161-optional"}` and exits 0.
- **TypeScript SDK `version()` and `SDK_VERSION`**: `version()` returns
  `{ version: SDK_VERSION, algorithm: "SHA-256+RFC8785" }`; `SDK_VERSION` exported as
  a named constant.

### Fixed / CI
- CodeQL workflow upgraded from floating `@v3` to SHA-pinned `@v4` refs; UI embed seed
  step added before Autobuild to resolve the `uiembed.go` pattern error.
- `version-gate.yml` floating `@v4` tag replaced with pinned SHA (`actions/checkout`).

### Tests
- `atb profiles validate --format json` snapshot test: asserts exit 0, all built-in
  profiles present, every entry reports `valid: true` with no errors.
- Corroboration bonus: nil-policy, signature-only, signature+snapshot, all-three-capped
  test cases in `internal/verify/evaluate_test.go`.
- Policy-doc signature: round-trip, absent-signature, tampered-doc, tampered-event
  cases in `internal/sign/policy_test.go` and CLI integration tests in
  `cmd/atb/main_test.go`.
- `TrustReport.PolicyDocSignatureValid`: nil-when-absent, true-when-verified,
  false-when-absent-signature test cases in `internal/verify/trust_report_test.go`.

### Docs
- `docs/roadmap.md` updated to reflect CAS v1 corroboration bonus and policy-doc compound
  signature shipped in v1.9.0; long-term objective section added.

## [v1.8.0] - 2026-04-19

### Added
- Added a verifier evaluation shim in `internal/verify/evaluate.go`: `EvaluateBundle` and `EvaluateConfig` centralise bundle loading, hash-chain integrity, RFC 3161 anchor verification, CAS normalisation, profile stamping, residual risk, and post-profile transformations in one place. The CLI, viewer, and API surfaces now derive reports from this function.
- Added `atb profiles validate`: validates all built-in profiles and any additional profiles supplied via `--file` or `--dir`; checks required fields, duplicate IDs, and CAS weight-vector sums; exits 0 or 1 and supports text or stable JSON output via `--format json`.
- Added `docs/roadmap.md`: in-repo roadmap covering short-term hardening for the profile DSL and verifier report path, medium-term CAS v1 and source signatures, and longer-term corroboration adapters, queue and storage gateways, reconciliation, and assurance-pack exports. Linked from `README.md`.

### Fixed
- TypeScript SDK parity suite now passes after completing the TypeScript 6.0.3 migration; `sdk/typescript/tsconfig.json` now carries the TS 6 declaration-build compatibility setting required by the updated toolchain.

### Tests
- Added `internal/verify/evaluate_test.go`: shim tests covering healthy bundles, broken hash-chains, and missing-event obligation failures.
- Added `cmd/atb/profiles_test.go`: coverage for built-in profile validation, malformed DSL files with bad weight sums and missing required fields, duplicate profile IDs, and JSON output format.

### Docs
- Added `docs/roadmap.md` and linked it from `README.md` under Planned work.

## [v1.7.3] - 2026-04-18

### Added
- Added a GitHub CodeQL static analysis workflow for Go on pushes and pull requests to `main`, plus a weekly scheduled scan.

### Changed
- Bumped `golang.org/x/crypto` to `v0.50.0`.
- Bumped `actions/setup-python` from `v5.6.0` to `v6.2.0`.
- Updated the `/web` dependency group: `next` to `16.2.3`, `axios` to `1.15.0`, `basic-ftp` to `5.3.0`, `follow-redirects` to `1.16.0`, `lodash` to `4.18.1`, and `vite` to `7.3.2`.
- Updated `llama-index` in `sdk/python` to `>=0.14.16`.
- Added `.atb-agent/` to `.gitignore` to prevent local agent tool binaries from being committed accidentally.

### Fixed
- Static Cypress runner now passes `--env MOCK_API=true` so the dashboard reaches loaded state before assertions run; fixes the gold release gate E2E failure introduced when `waitForDashboard` began checking that `trust-score-value` leaves the loading (`TEST-MODE`) state.

### Fixed
- Trust Score dashboard card: background colour corrected to comma-separated
  `hsl(H, S%, L%)` syntax on the `motion.div` wrapper and inner `Card` (loaded
  state) and on the loading-state `Card`; resolves axe `color-contrast`
  violations caused by space-separated HSL being rejected by some browsers.
- Accessibility test (`cypress/support/e2e.ts`): `waitForDashboard` now waits
  for the trust-score value element to leave the loading state before running
  axe, ensuring the audit targets rendered content rather than skeletons;
  violation details are now written to stderr via `console.error` so they
  surface in headless CI logs.

## [v1.7.2] - 2026-04-16

### Added
- Profile DSL v1: user-defined profile format defined in YAML. Custom profiles are
  evaluated identically to built-in profiles; `atb verify --profile <path>` and
  `atb trust-report --profile <path>` both accept a YAML file.
- CAS support for custom profiles via `genericSchemaSubScores`; any DSL profile with
  `supports_cas: true` now produces a full CAS object including sub-scores.
- `atb push` supports S3-compatible storage endpoints via `--endpoint-url`; push
  defaults (`target`, `endpoint-url`, `region`, `lock-mode`, `lock-until`,
  `credentials-source`) can be stored in `.atb/config.json` under a `push` key.
- `atb view --profile <id-or-path>`: evaluates the bundle against a named built-in
  profile or a DSL YAML file at startup and serves the result at
  `GET /api/v1/bundle/profile` (204 when no report; 200 + `ProfileReportSummary` when
  computed). `POST /api/v1/bundle/verify` recomputes and caches a fresh report.
- Legacy viewer redirect guidance added during the transition to the current single-view dashboard.

### Fixed
- Bundle save is now atomic: written to a temp file then renamed, preventing partial
  writes on crash or disk-full.
- `loadSchemaIfAvailable` replaced panic-recover with a proper error return.
- `classifyAnchorRoots` global eliminated; root certificates are now passed through
  the call chain, removing a global mutable state hazard.
- Misaligned `else` brace and struct field alignment in `verify.go` corrected.
- Read-only directory test skipped on Windows where the permission model differs.
- Go toolchain pinned to 1.24.2 across `go.mod`, Makefile, CI, and Dockerfile;
  all affected stdlib CVEs cleared.

### Changed
- Anchor verification consolidated: `internal/anchorverify` package removed; logic
  merged into `internal/trust`. No change to the public `atb verify` behaviour.

### CI
- Security scans now run on every push and PR, not only on schedule.
- `-race` flag added to the Go test step; TypeScript SDK tests added to CI.
- `check-versions.sh` runs in CI via a dedicated version-gate workflow.

### Tests
- Added RAG GC fixed-score, empty-bundle verify, and MCP RAG tool round-trip tests.
- Integration CAS assertions updated to expect non-nil for all named profiles.

### Docs
- `docs/integrations/worm-s3.md`: S3-compatible endpoint support documented; push
  event language removed to match implementation (local bundle is not modified on export).
- `docs/spec-dashboard.md`: `--profile` CLI interface, new API routes, and
  Profile/CAS summary panel spec added.
- `docs/compliance/eu-ai-act.md`: identity attribution boundary section clarifies
  ATB proves non-alteration but not truthfulness of claimed actor identities.
- `docs/security.md`: identity attribution caveat strengthened; recommended controls
  updated to require an independent identity layer or signing scheme.
- Quickstart Python example corrected to use canonical `rag_answer` event types.
- PageIndex event type mismatch with `rag_answer` profile and fixed GC sub-score
  documented.

## [v1.6.0] - 2026-04-15

### Pre-launch surface polish

- `atb view` now accepts `--profile <id-or-path>`: evaluates the bundle against the
  named built-in profile or a DSL YAML file at startup and exposes the result via
  `GET /api/v1/bundle/profile`. Without `--profile`, a "Run verify" button triggers
  `POST /api/v1/bundle/verify` to compute and cache a fresh `ProfileReportSummary`.
- Added `GET /api/v1/bundle/profile` (204 No Content when no report; 200 +
  `ProfileReportSummary` when computed) and `POST /api/v1/bundle/verify` to the
  dashboard API server.
- Rewrote README above-the-fold: explicit Why ATB bullets, What ATB does not do
  sub-section, `atb push` WORM export added to multi-surface list, demo asset hooks.
- Updated `docs/spec-dashboard.md` with new API routes, `--profile` CLI interface,
  and Profile/CAS summary panel spec; updated `docs/quickstart.md` section 4 to cover
  `--profile` usage and the verify report screenshot reference.
- Added `docs/launch/assets/` canonical path table to `docs/launch/README.md` with
  regeneration reminders for demo GIF and verify-report screenshot.

## [v1.5.1] - 2026-04-13

### Changed
- Align version constants to v1.5.1 across CLI, Python SDK, TypeScript SDK, and web package.

## [v1.5.0] - 2026-04-13

### Added
- MCP in-process integration test (`TestServeMCPVerifyIntegration`) exercises `atb_init` and `verify` via the MCP server without shelling out: initialises a bundle, appends an event via the bundle package, calls `verify` through the MCP tool handler, and asserts `exit_code=0` with no critical failures (implies `chain_valid=true`). Closes the test gap noted by the TODO in `internal/mcp/server_test.go`.
- `docs/integrations/mcp.md`: Claude Code CLI configuration example (`claude mcp add` and `.mcp.json`); MCP `initialize` handshake response shape; full input schemas and required-field tables for `verify`, `rag_index_record`, and `rag_retrieval_record`; tool list order now matches the implementation.

## [v1.4.0] - 2026-04-12

### Added
- `atb verify --with-snapshot-check` validates each `atb.snapshot` `bundle_hash` against the serialised prefix; mismatches report `snapshot_hash_mismatch at seq N`.
- `cas` object on `atb trust-report --json` output (alongside existing `cas_score` / `cas_grade`), aligned with verify report CAS.
- EU AI Act Article 12 mapping (`docs/compliance/eu-ai-act.md`) with profile table and limitations.
- Text verify output note when obligations fail: CAS is diagnostic only and does not overturn FAIL.
- NIST AI RMF practitioner mapping (`docs/compliance/nist-ai-rmf.md`) for CAS sub-scores and built-in profiles.
- End-to-end quickstart example under `examples/quickstart/`, including a runnable script and captured terminal output for a verifiable `privileged_tool_action` bundle with a CAS grade line.
- OpenTelemetry comparison guide (`docs/comparisons/opentelemetry.md`).

### Changed
- CAS weight vectors for `background_automation`, `policy_decision`, and `human_override` YAML templates to the documented profile-specific values (`data_export` unchanged).
- README: Install moved to follow Trust Model (verification narrative before installation).
- `docs/security.md` Limitations expanded: intra-bundle integrity vs capture completeness, local-first filesystem trust boundary.
- `docs/spec-v1.0.md`: snapshot `bundle_hash` definition, `--with-snapshot-check` behaviour, and that the field is not verified without the flag.
- `docs/compliance/eu-ai-act.md` rewritten to match the Article 12 mapping structure (overview, profile table, out-of-scope paragraph).
- `atb view` keeps a loopback default host and accepts an explicit `--host` override.
- `atb verify` and `atb trust-report` now report RFC 3161 anchor state explicitly as verified, partial, or failed.
- `docs/security.md` now states the local viewer exposure boundary and the exact RFC 3161 checks performed during `--with-anchor`.
- `docs/key-management.md` now states the versioned PBKDF2-SHA256 parameters for new and legacy encrypted bundles.
- CI now checks internal Markdown links under `docs/` and `README.md`.
- `sdk/python/README.md` now reflects the actual exported Python SDK import paths, append flow, and event type constants from `sdk/python/atb/`.
- `sdk/typescript/README.md` now reflects the actual exported TypeScript SDK package imports, append flow, and event type constants from `sdk/typescript/src/`.
- Docs and examples navigation updated to add the quickstart and comparison entries after auditing the edited index links.

### Fixed
- `--sign-policy` exits non-zero when no signing key is present (`no signing key found; run 'atb keygen' before using --sign-policy`).
- `atb verify` no longer reports a successfully validated TSA anchor as unverified in text or JSON output.

### Security
- `atb encrypt` writes ATBE wire version `0x02` with PBKDF2-SHA256 at `600000` iterations; `atb decrypt` accepts `0x01` (`100000`) and `0x02`.
- `atb verify --with-anchor` now requires TSA certificate-chain verification against the system roots in production before AC receives anchor credit.

### Tests
- CLI: `--sign-policy` without keypair; `--with-snapshot-check` tamper path and verify without the flag unchanged.
- `internal/verify`: CAS scenarios for extended profiles; trust-report JSON asserts `cas` when present.
- Viewer listener test now asserts that the default bind address is `127.0.0.1`.
- Anchor verification tests now cover verified, partial, and failed reporting states.

## [v1.0.0.1] - 2026-04-09

### Changed
- Version metadata aligned to `v1.0.0.1` release tag
- Docker publish hang fixed; login logout disabled, timeout added
- Smoke check drift resolved in release guidance and checklist

## [v1.0.0] - 2026-04-09

### Changed
- Version bumped to `1.0.0` across CLI, Python SDK, TypeScript SDK, and web package
- GitHub Actions updated to Node 24-compatible versions
- Docker publish workflow reviewed against `DOCKERHUB_USERNAME` secret presence; no gate added because the repository secrets are configured

### Notes
- `v1.0.0-rc` tag remains at `a65a70b`; `v1.0.0` supersedes it
- `v1.0.0` tag from 2026-03-10 (`bb2cccb`) refers to an earlier development iteration and is distinct from this release

## [v1.0.0-rc] - 2026-04-09

### Added
- Ed25519 bundle signing integration test
- v1.0.0-rc release-readiness checklist
- Key rotation procedure in `docs/key-management.md`

### Changed
- Version bumped to `1.0.0-rc` across the CLI, Python SDK, TypeScript SDK, and web package
- Roadmap updated to reflect completed `v0.9.x` items and `v1.0.0-rc` scope

## [v0.9.2-beta] - 2026-04-08

### Changed
- Version bump to 0.9.2-beta; closes the April 2026 release window
- Python SDK version aligned to 0.9.2b1

## [v0.9.1-beta] - 2026-04-08

### Added
- `atb trust-report --format json` TrustReport output with profile-specific evidence sections for all six built-in profiles
- `atb export --format compliance --json` ComplianceManifest output
- MCP integration guide (`docs/integrations/mcp.md`)
- Event type constants for Python and TypeScript SDKs
- Contributor orientation and working standards documentation

### Fixed
- `background_automation` profile migrated to `ai.job.*` taxonomy; template, verifier, and auto-detection all consistent
- ResidualRisk now set to Critical when chain integrity fails
- TSA and CAS capability claims corrected in README and security.md

### Tests
- Snapshot tests for `atb verify --format json` across all six profiles
- Snapshot tests for `atb trust-report --format json` across all six profiles, including negative and edge cases
- Golden fixture for compliance manifest JSON output

## [v0.9.1-beta] - 2026-04-07

### Changed
- Align release metadata across CLI version output, README badges/status, and SECURITY supported-version table.

## [v0.9.0-beta] - 2026-04-XX

### Changed
- Versioning reset to v0.9.0-beta to accurately reflect pre-production status
- TSA verification: certificate chain validation is implemented and used for CAS scoring
- Bundle-level Ed25519 signing: fully implemented via `atb sign` and `atb verify`

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

### Added
- Add `atb bundle new` as an alias for `atb init`.

### Fixed
- Guard against a `nil` `SubScores` map in verify output and add SC fallback
  handling for unmatched profiles.
- Copy all embedded source files into the Go builder Docker stage.

### Changed
- Build web assets before `go vet` in the Gold Release Gate workflow.
- Install `sdk/typescript` dependencies before Gold Release Gate tests.
- Bump SDK versions to `1.4.0` for release tag parity.
- Pin GitHub Actions workflow steps to full commit SHAs.
- Update `vitest` from `4.0.18` to `4.1.2` to address the `flatted` CVE.
- Update `docker/setup-buildx-action` from `3.12.0` to `4.0.0` for Node 24.
- Update `actions/upload-artifact` from `4.6.2` to `7.0.0` for Node 24.
