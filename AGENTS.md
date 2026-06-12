# ATB Maintainer And Agent Harness

This file is the canonical maintainer and coding-agent harness for ATB. If
another repository document conflicts with it, follow this file and then bring
the surrounding docs back into line.

## Product Boundary

ATB is a local-first tamper-evident audit trail for AI and agent workflows. It
records events into append-only, hash-chained `.atb` bundles so a reviewer can
verify later whether the recorded sequence was altered.

ATB is not a hosted observability platform, model evaluator, compliance
certification service, key-management system, or SIEM. It proves integrity of
what was recorded; it does not prove capture completeness, model correctness,
actor identity, or regulatory compliance by itself.

ATB remains MIT-licensed open source. Hosted custody, auditor access,
multi-tenant auth, retention policy, legal hold, billing, and custodian-of-record
work belong in Custos or another external product, not in the ATB runtime.

## Core Invariants

1. Bundles are append-only NDJSON. Existing records are never edited.
2. Each record hash is `SHA-256(UTF-8(hex(prev_hash)) || RFC8785(event))`.
3. The genesis sentinel is 64 zero hex characters.
4. Canonical hash input is frozen unless a manifest-version/canonicalisation
   migration is deliberately performed.
5. The default writer remains manifest version `1` unless VERSIONING.md says
   otherwise.
6. Local-first is the default. Network activity must be tied to explicit network
   commands or configured signing/push backends.
7. `LoadVerified` is required for integrity-sensitive reads. Non-validating
   `Load` is for inspection paths only.
8. ATB Agent is optional and loopback-only; CLI and SDK workflows must continue
   to work without it.

## Versioning Rules

`VERSIONING.md` is authoritative. In short:

- CLI/SDK release versions follow SemVer.
- Manifest versions change only for canonical hash input or on-disk manifest
  schema compatibility changes.
- `schemas/event.v1.json` changes only when the event schema contract changes;
  optional additive event payload fields may stay on v1.
- Canonicalisation changes are the highest-risk class: they require manifest
  version review, golden-vector regeneration, and synchronized Go/Python/TS SDK
  parity.

Go is canonical. Python and TypeScript must agree byte-for-byte with Go via
`make test-golden` before release.

## Release Gates

Before a public release:

```bash
GOCACHE=$(pwd)/.gocache/release GOTOOLCHAIN=go1.26.4 make test-golden
GOCACHE=$(pwd)/.gocache/release GOTOOLCHAIN=go1.26.4 go test ./... -count=1
bash scripts/check-versions.sh
```

Run the broader release preflight when the local environment can install
dependencies and build web artefacts:

```bash
bash scripts/release-check.sh
```

If a command fails because it cannot write to the default Go cache, rerun with a
repo-local `GOCACHE` instead of weakening the test.

## Files Requiring Extra Review

Treat these as public or compatibility-sensitive surfaces:

- `internal/bundle/`, `internal/hash/`, `internal/canonicalize/`
- `schemas/event.v1.json`, `docs/spec-v1.0.md`, `docs/spec-ai-traces.md`
- `internal/verify/report.go`, `pkg/custody/schema/`
- `cmd/atb/exit_codes.go`, public CLI help, and documented JSON output
- `sdk/python/`, `sdk/typescript/`
- `LICENSE`, `SECURITY.md`, `VERSIONING.md`

Schema or canonicalisation changes require corresponding docs, tests, and
CHANGELOG entries in the same release-preparation pass.

## Documentation Tone

Use direct technical language. Separate shipped behaviour from planned work.
Do not describe ATB as certifying compliance, replacing a custodian, or proving
facts beyond recorded bundle integrity. Compliance documents may map how ATB can
support an obligation, but must state the operational and legal limits.

## Session Log

### 2026-05-24 Session Summary

- Pre-flight: 12 provability commits on `main` ahead of `origin/main`;
  `.git-commit-plan.txt` and `scripts/split-provability-commits.sh` untouched.
- Phase 1 survey written to `/tmp/atb-md-survey.md` (filtered inventory, drift
  flags, proposed deletions).
- Phase 2-5 doc cleanup: 6 signed commits on top of provability stack — case
  study move + `roadmap.md` normalisation, redundant page removal, Go 1.26.4
  alignment, governance/spec tone pass, README front-door restructure.
- README quickstart verified end-to-end in `/tmp/atb-readme-quickstart`.
- Deferred without edit: duplicate CHANGELOG `[v1.6.0]` headings,
  `docs/compliance/{gdpr,soc2,...}` and `docs/spec/bundle-push.md`
  (export/CLI dependencies).
- All cleanup commits signed `G` by Paddy Guest.

### 2026-06-13 Finalization handoff

- Practitioner review P1 absorbed on `main`: intercept shutdown finalisation,
  session-index profile inference, evidence-pack Markdown, README/CISO honesty.
- Release pipeline: Docker `v1.14.3` green; gold gate green on `main`; tag
  `v1.14.3` Release/npm/PyPI still blocked historically — next green tag advances registries.
- Maintainer handoff: [`docs/maintenance/agent-handoff.md`](docs/maintenance/agent-handoff.md).
