# ATB maintainer and agent harness

This file is the canonical maintainer and coding-agent harness for ATB. If
another document appears to conflict with it, follow this file and then bring
the surrounding docs back into line.

## 1. What ATB is and is not

ATB is a narrow-scope product for tamper-evident audit trails of AI agent
workflows. It records events into a local append-only bundle, hash-chained over
RFC 8785 canonical JSON, so the recorded sequence can be verified later without
a hosted backend. The cryptographic chain, optional Ed25519 bundle signature,
and optional RFC 3161 timestamp anchor together provide the evidence shape that
EU AI Act Article 12 audit obligations expect.

ATB is local-first by default. The CLI runs entirely offline unless the user
explicitly invokes a network subcommand (`anchor`, `push`, `verify --remote`,
or a KMS sign backend).

### Non-goals

- Not a general AI operations platform.
- Not a hosted tracing or observability service.
- Not a key-management system.
- Not a SIEM or query engine.
- Not a workflow orchestrator or policy decision engine.
- Does not prove model correctness, actor identity, or capture completeness.
  CAS is a structural score over what a bundle contains; it is not an audit
  opinion.

## 2. Repository map

Stable surface (treat as product contract):

- `cmd/atb/` — CLI entry point and subcommand wiring.
- `internal/` — runtime packages (bundle, hash, canonicalize, sign, signer,
  verify, encrypt, anchor, archive, push, profiles, capture, evidence, event,
  trust, mcp, corroboration, integration, export).
- `pkg/api/v1/` — local viewer HTTP API handlers.
- `schemas/event.v1.json` — canonical event schema (see §6).
- `docs/` — product documentation; see §16 for tone rules.

Test infrastructure:

- `test/golden/` — cross-language hash and verifier parity fixtures.
- `test/integration/` — end-to-end push and gateway tests.
- `test/performance/` — bundle load benchmarks.
- `scripts/` — release, hygiene, and CI helpers.

In-flight or auxiliary surface (handle with extra care):

- `web/` — local viewer frontend; do not modify without checking current state.
- `tools/` — build-time tool dependency pin.
- `sdk/python/`, `sdk/typescript/` — language SDKs. Cross-language parity is
  a hard requirement; see §12.
- `examples/`, `demo/` — runnable examples and demo bundles. Treat as
  documentation, not as runtime code.

## 3. Build, test, and lint

Required Go version: **1.25.9** (`go.mod`). Newer toolchains are accepted via
`GOTOOLCHAIN=go1.25.9` (the Makefile sets this).

CLI binary:

```sh
go build -o ./atb ./cmd/atb
```

Unit and race tests:

```sh
go test ./... -count=1
go test ./... -count=1 -race
```

Targeted suites:

```sh
go test ./internal/bundle/... -run Adversarial   # adversarial corpus
go test ./internal/bundle/... -run Fuzz -fuzz=.   # fuzz corpus (interactive)
go test ./test/golden/... -count=1                # cross-language goldens
go test ./test/integration/... -count=1           # integration tests
```

Python SDK:

```sh
cd sdk/python && python -m pytest -q
```

TypeScript SDK:

```sh
cd sdk/typescript && npm install && npm run build && npm test
```

Pre-commit gate:

```sh
make hygiene-quick   # fmt, vet, staticcheck, go test, web lint and typecheck
```

`make hygiene-quick` is wired as a pre-commit hook and must succeed before any
commit lands.

## 4. CI contract

Workflows under `.github/workflows/`:

- `ci.yml` — primary build, lint, and test matrix on every PR.
- `codeql.yml` — static analysis on push and PR.
- `security.yml` — `govulncheck` and dependency scanning.
- `version-gate.yml` — schema and SDK version parity gate.
- `release.yml` — tag-driven release artefact build and publish.
- `gold-release.yml` — golden corpus regeneration on release.
- `docker-publish.yml` — image publish on release tag.
- `ops.yml` — operational housekeeping.

PR merge gate: `ci.yml`, `codeql.yml`, `security.yml`, and `version-gate.yml`
must all be green. Reproduce locally with `make hygiene-quick` plus
`scripts/govulncheck.sh`.

## 5. Core invariants

1. **Append-only.** A bundle is an append-only NDJSON file. Existing records
   are never edited.
2. **Hash chain.** Each record's hash is `SHA-256(UTF-8(hex(prev_hash)) ||
   RFC8785(event))`. The genesis sentinel is 64 zero hex characters.
3. **Manifest first.** Every valid bundle begins with an
   `atb.bundle.manifest` record at sequence 0. Bundles without a manifest
   record predate schema v1 and are accepted only by the legacy reader path.
4. **Signature is verifiable in isolation.** The signature record carries
   enough material (`signature`, `pubkey`, `algorithm`) to verify with no
   network call.
5. **No silent migration.** A bundle either re-verifies byte-for-byte against
   the same canonicalisation rules used to produce it, or it fails verify.
6. **Canonical hash input is frozen.** Any change to the `Event` struct JSON
   tags, field set, or RFC 8785 serialisation is a BREAKING change. It
   requires a manifest version bump, cross-language golden regeneration, and
   a `CHANGELOG.md` entry under a new `[X.Y.Z]` heading.
7. **Local-first by default.** The CLI must not initiate network traffic
   unless the user explicitly invokes a network subcommand
   (`anchor`, `push`, `verify --remote`, KMS sign backends).

## 6. Schema authority and change policy

`schemas/event.v1.json` and `docs/spec-v1.0.md` are the two-part authoritative
schema. Neither is secondary; the schema file expresses validation rules and
the spec expresses prose contract and worked examples. Both must be updated
together.

Change policy:

- **Additive (no version bump):** adding an optional field to an event
  type's `data` payload, adding a new event type to `documented_event_types`,
  adding a new `reserved_event_types` entry that does not collide with
  existing readers.
- **Breaking (manifest version bump required):** renaming a field, removing
  a field, changing the `Event` envelope struct, changing canonicalisation,
  changing the genesis sentinel, changing the hash algorithm.

SDK parity obligation: every schema change is accompanied by updated golden
fixtures in `test/golden/` and corresponding SDK regeneration in the same PR.

## 7. How to add a new event type

1. Add the constant to `internal/event/types.go` and append a `Registry` row
   describing it.
2. Add a documentation entry under `documented_event_types` in
   `schemas/event.v1.json` with `description`, `properties`, and `required`.
3. Add or extend a worked example in `docs/spec-v1.0.md` and, where relevant,
   `docs/spec-ai-traces.md`.
4. Update any obligation profile that should track the new type (see
   `internal/profiles/` and `docs/profiles.md`).
5. Regenerate the cross-language goldens under `test/golden/` and run
   `go test ./test/golden/...`.

## 8. Obligation profiles

A profile is a declarative DSL document that names the events and conditions
required to satisfy an audit obligation, such as `atb.profile.rag_answer` or
`atb.profile.privileged_tool_action`. Profiles live under
`internal/profiles/templates/` and are evaluated by `atb verify --profile`.

To add a profile, follow `docs/profiles.md`: copy an existing template,
declare the required events and gates, write a fixture under
`internal/verify/testdata/`, and add it to the verifier integration tests.

## 9. CLI surface conventions

- Exit codes are constants in `cmd/atb/exit_codes.go`. The current set
  includes `exitSuccess` (0), `exitUserError` (1),
  `exitIntegrityFailure` (2), `exitVerifyFailure` (3),
  `exitSystemError` (3), and `exitLockContention` (9). New exit codes require
  a `docs/spec-v1.0.md` update.
- Every command that emits structured output supports `--format text|json`.
  Default is `text`.
- Flag names are lowercase hyphenated. Single-letter flags are reserved
  (`-h` for help, `-v` for verbose where present). Do not invent new
  abbreviations.
- All user-visible strings use British English (see §16).

## 10. Cryptographic contract

- **Hashing:** SHA-256 over `UTF-8(hex(prev_hash)) || RFC8785(event)`.
- **Genesis:** 64 zero hex characters. Sentinel only; not derivable from any
  event.
- **Signatures:** Ed25519 (default) or ECDSA P-256 over the SHA-256 of the
  pre-signature bundle bytes. The signature record stores
  `signature`, `pubkey`, `algorithm`, and optional `key_id`, `backend`,
  `signed_at`.
- **Key format:** raw 32-byte Ed25519 public key, base64-encoded.
  ECDSA P-256 public keys may be PKIX DER or 65-byte uncompressed point.
- **Canonicalisation:** RFC 8785 JCS with the float rule pinned at v1.1.2
  (Go `encoding/json` default for values ≥ 1e21 or ≤ 1e-6).

See `docs/security.md` and `docs/key-management.md` for depth.

Adding a new signing backend requires a new implementation under
`internal/signer/` and must not change the pre-image contract. The backend
label appears verbatim in the signature record's `backend` field.

## 11. Key and secret handling

- Never commit private keys. `*.pem`, `atb-key*`, and `*.atb` (root-level)
  are gitignored.
- Test fixtures use pinned deterministic test keys, generated inside the test
  with `crypto/ed25519`. Production keys never appear in tests.
- `atb keygen` writes to the current working directory by default. Run it
  outside the repository working tree, or use `--output` to direct it
  elsewhere.
- `.gitignore` covers `coverage.out`, `trivy-report.json`, `*.pem`, root
  `*.atb`, `.tmp*`, build output, and SDK build directories.

## 12. Cross-language parity obligations

Go, Python, and TypeScript SDKs must produce byte-identical canonical hashes
for the same event corpus. Golden fixtures under `test/golden/` are the
regression contract: `input.json`, `output-go.json`, `output-python.json`,
`output-typescript.json`, `hash-go.txt`, `hash-python.txt`,
`hash-typescript.txt`.

Any change to `internal/hash/` or `internal/canonicalize/` must be accompanied
by regenerated goldens for all three languages in the same PR.

Float serialisation edge cases — values ≥ 1e21 and values ≤ 1e-6 — are
pinned in the golden corpus. Do not change the Go `encoding/json` default
behaviour without a manifest version bump.

## 13. Dependabot and vulnerability policy

As of 2026-04-27 the private repository has 3 open Dependabot alerts: 1 high,
2 moderate. They are informational while the repository is private and
unreleased.

Policy:

- All high alerts must be resolved before any public release or v1.0 tag.
- Moderate alerts must be triaged within 30 days.
- `govulncheck` must pass cleanly in `security.yml` before a release tag is
  cut.

## 14. Release process

1. Run `scripts/check-versions.sh` and `scripts/release-check.sh`.
2. Move the `[Unreleased]` block in `CHANGELOG.md` under a new `[X.Y.Z]`
   heading dated today.
3. Bump `sdk/python/pyproject.toml` and `sdk/typescript/package.json` to
   match.
4. Tag: `git tag -s vX.Y.Z -m "release: vX.Y.Z"`.
5. Push the tag: `git push origin vX.Y.Z`.
6. Verify the GitHub Actions release job and `gold-release.yml` both pass,
   and the published artefacts (binaries, Docker image, Python wheel, npm
   package) match the tag.

## 15. Security reporting

See `SECURITY.md` for the disclosure path and supported versions.

Threat model in scope: tamper-evidence of the bundle, key confidentiality at
rest, CLI input validation, parser robustness against adversarial inputs.

Out of scope: network transport security (rely on TLS), key rotation policy
(operational concern), and the trustworthiness of upstream model providers.

## 16. Tone and documentation rules

- British English throughout: colour, behaviour, analyse, organise,
  serialise, recognise, fulfilment.
- No marketing copy. Do not write "planned", "not yet implemented", or
  "coming soon" anywhere user-facing — either the feature ships or it is not
  mentioned.
- `README.md` is the public-facing entry point. `docs/` is the technical
  depth. `AGENTS.md` (this file) is the maintainer harness.
- Schema and spec changes require a matching documentation update in the
  same commit.

## 17. Agent operation rules

- Prefer small, single-concern patches. One commit per logical change.
- Never invent capabilities not present in the current codebase.
- Files requiring human review before merge:
  `schemas/event.v1.json`, `docs/spec-v1.0.md`, `internal/bundle/`,
  `internal/hash/`, `internal/canonicalize/`, `LICENSE`, `SECURITY.md`.
- Always run `make hygiene-quick` before committing.
- Never use `git add -A` or `git add .`. Stage files by explicit name.
- Append a session summary to `~/atb/agents.md` (lowercase) under a dated
  heading after each session.

## 18. Known hot areas and in-flight work

- `internal/bundle/lock.go` is now wired for advisory file locking on Unix
  and Windows. Single writer per bundle remains the contract; lock failures
  surface as `exitLockContention` (9).
- `internal/bundle/sign.go` writes via temp-file plus rename; concurrent
  Sign calls against the same bundle are still unsupported and are
  protected by the bundle lock.
- `cmd/atb/capture.go` does not yet propagate child signal exit codes
  (`128 + N`). Child failures surface as a generic non-zero exit.
- KMS signing backends (`internal/signer/awskms`, `gcpkms`, `vault`) are
  wired and tested against fakes; no live cloud integration test runs in
  CI.
- `atb push` ships the local-to-S3 transport and a stubbed HTTP transport.
  Verify transport status in `internal/push/` before extending it.
- `web/` viewer is feature-complete for v1.10.0; do not modify without
  reviewing `docs/spec-dashboard.md` first.

## 19. Phase 0 (K1-K14) operational notes

### Summary

- `Save` and `SignTo` now write through crash-safe temp-file, fsync, atomic
  rename, and parent-directory fsync.
- Advisory bundle locking is wired for writers; contention surfaces as
  `ErrBundleLocked` and CLI code `exitLockContention` (9).
- Signing is TOCTOU-hardened: digest, parsed bundle, and source mode come from
  one locked read before the atomic write.
- `Load` is intentionally non-validating; use `LoadVerified` when structure,
  manifest presence, and hash-chain integrity must be enforced.
- Snapshot names are validated by `appendSnapshot` / `validateSnapshotName`
  before bundle I/O.
- `snapshotExitCode` is the shared snapshot error-to-exit-code mapper used by
  snapshot, capture, and import snapshot paths.
- Context propagation is available on bundle load/save/sign paths, with
  five-minute operation timeouts on the long-running CLI paths that load,
  snapshot, import, capture, or verify bundles.

### When writing new code

- Use `LoadVerified` for integrity-sensitive paths.
- Use context-aware `Load`, `Save`, `Sign`, and `SignTo` from `cmd/atb`
  call sites.
- Respect snapshot naming rules by routing snapshot writes through
  `appendSnapshot` / `validateSnapshotName`.
- Use `writeAtomic` for any new bundle-file-write path.
- Use `snapshotExitCode` for snapshot-related error to exit-code mapping.

### Future cleanups

- Consolidate legacy call forms with context-first APIs once all internal and
  CLI callers have migrated.
- `exitSystemError` and `exitVerifyFailure` both map to code 3 for
  compatibility; treat a numeric split as a future breaking or major refactor,
  not a patch-release change.

## 2026-05-02 Session Summary

- Added helper bridge records so helper-only bundles satisfy `atb.profile.rag_answer`;
  verified Python and TypeScript helper bundles with `atb verify`.
- Aligned request-event documented required fields with the README quickstart
  and policy profile.
- Corrected TypeScript SDK README scope wording.
- Added the GitHub Releases binary install path before the Go install option.
- Added the README inspect step before the local review UI command.
- Final checks run: `go build ./...`, `go test -race ./...`,
  `bash scripts/check-versions.sh`, and
  `gh run list --repo pcguest/atb --workflow ci.yml --limit 1`.
