# ATB Versioning Policy

This document is the authoritative source for breaking-vs-non-breaking decisions across ATB. It covers the four versioning universes, the rules that govern each, and the release-time obligations that bind them together. If another document conflicts with this one, follow this file and bring the surrounding docs back into line.

## Versioning universes

ATB has **four** independently-tracked version concepts. Confusing them is the most common source of accidental breaking changes.

| Universe                       | Identifier                                    | Bump rule                                                                                          | Example                            |
|--------------------------------|-----------------------------------------------|----------------------------------------------------------------------------------------------------|------------------------------------|
| CLI / SDK release version      | `vMAJOR.MINOR.PATCH` (SemVer)                 | Bumped per release. SemVer applies to the user-observable CLI and to the public Python/TypeScript SDK API. | `v1.10.0`                          |
| Bundle manifest version        | Integer (`ManifestVersion` constant)          | Bumped only when the **canonical hash input** or the **on-disk manifest schema** changes in a way that would cause an existing reader to mis-parse or mis-hash. | `1` (default), `2` (opt-in)         |
| JSON schema version            | Filename: `schemas/event.v1.json`             | Bumped only when the JSON Schema describing event records gains required fields or removes existing ones. Additive optional fields stay on `v1`. | `event.v1.json`                    |
| Canonicalisation profile       | Implicit in the canonicaliser's behaviour     | Bumped (de facto) when RFC 8785 implementation rules change in a way that produces different bytes for the same input. Treated as a manifest-version bump for compatibility purposes. | v1.1.2 float profile (current)      |

The relationship between them:

- A **CLI MAJOR** bump may but need not change the manifest version.
- A **manifest version** bump always implies a CLI MINOR bump at minimum (and often MAJOR if existing bundles cease to verify).
- A **JSON schema filename** bump (`event.v1.json` → `event.v2.json`) implies a manifest version bump and is a CLI MAJOR.
- A **canonicalisation profile** change is the most contagious of all — it invalidates every existing bundle's hash chain and is treated as a CLI MAJOR plus a manifest version bump plus a fresh golden-vector regeneration plus a synchronised SDK release.

## Breaking vs non-breaking changes

### Breaking (require manifest version bump, new golden vectors, cross-language SDK update)

- Any change to the **canonical hash input**: adding a hashed field, removing a hashed field, changing a `json:"…"` tag, changing `omitempty`, changing field ordering rules, or changing how the canonicaliser serialises a value (notably floats, strings with surrogate pairs, or empty containers).
- Any change to the **encoding** of a hashed field — e.g. switching `signed_at` from RFC 3339 to RFC 3339 nano in a way that affects already-signed bundles.
- Removing a **required** field from the manifest record, the signature record, or any reserved system event type.
- Changing the **wire form of the manifest data payload** for an existing manifest version (e.g. moving v1 from a JSON-encoded string to a structured object). Add a new manifest version instead.
- Changing the meaning of an existing exit code constant, removing a CLI flag, or removing a public SDK function.
- Removing or renaming a public Python/TypeScript SDK export.
- Changing the algorithm-dispatch contract in the verifier (e.g. flipping the default algorithm from `ed25519`).

### Breaking (require a CLI MAJOR bump but not a manifest version bump)

- Renaming a CLI subcommand without a deprecation alias.
- Changing JSON output shapes for an existing CLI subcommand in a way that breaks documented parsers.
- Removing a hosted endpoint or signed-record field that an SDK depends on.
- Tightening a previously-permissive parser to reject inputs that older releases accepted.

### Non-breaking (additive — safe in MINOR or PATCH releases)

- Adding optional fields with `omitempty` to a `data` payload, provided the field is **not** part of the canonical hash input for already-emitted records. New optional fields on the signature record (`algorithm`, `key_id`, `backend`, `signed_at`) are the canonical example.
- Adding new event types to the registry (`internal/event/types.go`).
- Adding new CLI subcommands.
- Adding new CLI flags that do not change existing output for unset values.
- Adding new exit-code constants with new numeric values.
- Adding new optional configuration keys whose absence preserves previous behaviour.
- Adding new SDK exports (functions, types, or modules) without changing the existing surface.

### Always-allowed (no version implications)

- Doc-only changes that do not modify behaviour.
- Refactors that preserve byte-for-byte output of every supported command and the canonical hash of every fixture bundle (the golden tests are the gate).
- Adding tests, fuzz corpora, or fixtures.

## When to bump the manifest version

Bump the **manifest version** integer (`ManifestVersion` constant in `internal/bundle/bundle.go`) under any of these triggers:

1. The canonical hash input changes for any record type — fields added, removed, or re-tagged on a struct that participates in canonicalisation.
2. The wire shape of `event.data` changes for an existing reserved system event type (manifest, signature, anchor) in a way that requires a different parser.
3. The canonicaliser's output bytes change for any input that an older version produced. This includes changes to RFC 8785 float serialisation, string escaping rules, or key-ordering behaviour.
4. A required field is added to the manifest, signature, or anchor record.
5. The hash algorithm itself changes (e.g. SHA-256 → SHA-3-256). This is a CLI MAJOR.

When you bump the manifest version, you must also:

- Update `ManifestVersionMax` if the new version is larger than the previous maximum.
- Update the parse guard in `parseManifestData` so unsupported versions return an error wrapping `ErrMalformed` with a clear "this bundle requires a newer version of atb" message.
- Regenerate the golden vectors (`go test ./internal/hash/... -run TestGoldenVectors -update-vectors`).
- Re-run the cross-language `make test-golden` so Python and TypeScript continue to match byte-for-byte.
- Add a CHANGELOG entry under the next unreleased version that names the bump explicitly.
- Document the new shape in `docs/spec-v1.0.md` alongside the previous shape, side-by-side.

### Historical: the v1.1.2 float profile break

Releases prior to ATB v1.1.2 used a different float-serialisation rule from the current canonicaliser: floats with absolute value ≥ `1e21` or strictly less than `1e-6` are now emitted in exponential form, where pre-v1.1.2 code emitted decimal form. **Bundles signed by pre-v1.1.2 ATB that contain such floats will not re-verify** under any current build. This is acknowledged as a one-off historical break, documented in `internal/hash/hash.go`'s package comment, and pinned by the canonical-hash golden corpus. A reader encountering such a bundle should re-export events using the original release and re-import them under a current build.

The v1.1.2 break is the only sanctioned historical break in the canonicalisation profile. Any future change to the profile must be treated as a deliberate manifest version bump with full migration documentation.

## Capture v1 additions (non-breaking)

This block records what was added in the Capture v1 development cycle. Every item below is purely additive: no existing canonical-hash input, no existing JSON schema field, and no existing golden vector was modified, so the changes are appropriate to a MINOR release and do not require a manifest version bump on their own.

- **New CLI subcommands**: `atb import chatlog` (ingests chat transcripts via the `generic-jsonl` provider) and `atb capture run` (wraps a child process and exports capture context to it via environment variables). Both are documented in `AGENTS.md` § Capture v1 layer.
- **New internal packages**: a `capture` package under `internal/` that owns chatlog parsing, mapping, and the in-memory append helpers shared by both subcommands. The bundle write path also gained an OS-level advisory lock helper (`flock` on Unix, `LockFileEx` on Windows) that surfaces `ErrBundleLocked` on contention.
- **New optional signature fields**: the `atb.bundle.signature` data payload gained `algorithm`, `key_id`, `backend`, and `signed_at`. All four are optional with `omitempty`; readers that pre-date them treat their absence as `algorithm = "ed25519"` and otherwise unset, preserving compatibility with bundles signed by older releases.
- **Manifest version 2 (read support, opt-in writer)**: a v2 manifest format is supported by the reader and may be written via `--manifest-version 2`. The default writer remains v1. Reader-side dispatch is governed by `ManifestVersionMax`; bundles declaring a manifest version greater than the maximum return an error wrapping `ErrMalformed`.
- **New exit code**: `exitLockContention` (value `9`) is returned when a bundle write path cannot acquire the advisory lock. Callers should retry after a short delay; the mutating subcommands accept `--lock-wait <duration>` to extend the in-process wait window before this exit code is surfaced. The constant is documented in `AGENTS.md` § Exit codes; it has a freshly-allocated numeric value and does not collide with any pre-existing exit code.
- **Golden corpus integrity**: the cross-language canonical-hash corpus was *not* modified by Capture v1. Every Go, Python, and TypeScript golden vector that existed before the cycle still hashes byte-for-byte to the same value. New tests were added; no existing vector was edited or regenerated.

## Bundle compatibility matrix

| ATB version range            | Manifest versions read    | Manifest versions written (default)        |
|------------------------------|---------------------------|--------------------------------------------|
| `v1.0.x` – `v1.1.1`          | `1`                       | `1`                                        |
| `v1.1.2` – `v1.9.x`          | `1`                       | `1` (post-v1.1.2 float profile)            |
| `v1.10.x` and later          | `1`, `2`                  | `1` (default; `2` opt-in via `--manifest-version 2`) |

Notes:

- The default writer remains manifest version `1` for maximum compatibility with deployed readers.
- A `v1.10+` build opening a bundle whose manifest declares any version greater than `ManifestVersionMax` (currently `2`) returns an error wrapping `ErrMalformed` rather than silently dropping fields.
- Bundles created without a manifest record (legacy pre-v1.0 captures) are still readable; the reader treats them as implicit version `1`.

## Migration policy

There is **no `atb migrate` tool today.** Cross-version migration is a known gap tracked for a future release.

In the interim, the manual procedure for moving a bundle across an incompatible canonicalisation profile or manifest version is:

1. Verify the source bundle with the **original** ATB version that wrote it, capturing its `atb verify --format json` output as evidence of the integrity claim at the source.
2. Export the events as plain NDJSON or via `atb export` using the original version.
3. Re-import the exported events under the current ATB build using `atb import chatlog` or `atb capture run`, producing a fresh bundle whose hash chain is computed under the current canonicalisation profile.
4. Sign the new bundle. Retain the original bundle and the verification output from step 1 as the audit record of the migration; the new bundle's chain does not transitively certify the original chain.

This procedure is intentionally conservative: it explicitly does not pretend that re-imported events carry forward the original tamper-evidence guarantee. A future `atb migrate` would automate steps 2–4 while preserving a recorded link to the source bundle.

## SDK parity obligation

**Go is the canonical implementation.** The Python and TypeScript SDKs MUST agree byte-for-byte with Go for every input in the cross-language golden corpus (`internal/hash/testdata/canonical_vectors.json`). The corpus is exercised by:

- `go test ./internal/hash/... -run TestGoldenVectors`
- `.venv/bin/python -m pytest sdk/python/tests/test_canonical_hash.py`
- `cd sdk/typescript && npm test -- --run canonical_hash`
- `make test-golden` (runs all three)

A failing golden test in **any** language blocks the release across all three. Specifically:

- A Python or TypeScript SDK MINOR or MAJOR release MUST NOT be cut while `make test-golden` fails for that SDK.
- A Go release that introduces a deliberate canonicalisation change must regenerate the corpus and ship synchronised Python and TypeScript releases. The Go release tag and the SDK release tags should land in the same release window.
- If a third-party verifier produces a different hash from the Go reference for any vector, the Go reference is presumed correct and the third-party implementation is presumed buggy — but the discrepancy must be investigated and recorded in `CHANGELOG.md` so future readers know the corpus is the contract.

## Deprecation policy

- Deprecations are announced in `CHANGELOG.md` and in release notes.
- Deprecated behaviour remains available for at least one subsequent MINOR release.
- Deprecated behaviour is only removed in a later MAJOR release.
- Deprecating a CLI flag requires keeping it parseable (with a warning to stderr) until removal; deprecating an SDK export requires keeping it exported (with a runtime or compile-time warning) until removal.

## Tagging rules

- Release tags are annotated and formatted as `vMAJOR.MINOR.PATCH`.
- CI publish workflows trigger only on tags matching `v*.*.*`.
- The tag version must match the checked version locations validated by `scripts/check-versions.sh`.
- Pre-release versions use SemVer pre-release suffixes for release tags and npm, and PEP 440 pre-release forms for Python.
  - Example forms: `v1.5.0-rc.1` for the release tag, `1.5.0-rc.1` for npm, `1.5.0rc1` for Python.
- A pre-release MUST pass `make test-golden` against the corpus committed at the same SHA — pre-release status does not relax the SDK parity obligation.

## Documentation and release coordination

- If a release changes user-visible behaviour, update `CHANGELOG.md`, release-facing `README.md` text, and any affected operational docs in the same release-preparation pass.
- Roadmap targets and issue milestones are planning aids only. They do not override Semantic Versioning, the manifest-version bump rules, or the SDK parity obligation.
- A schema change without a corresponding CHANGELOG entry, manifest-version review, and golden-vector regeneration is automatically blocked at review time. See `AGENTS.md` § "Files that require human review before merge".
