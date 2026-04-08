# ATB Repo Guide

## What ATB Is

ATB (Audit Trail Bundles) is a local-first, tamper-evident audit trail system for AI agent workflows. It records workflow events as hash-chained bundles that can be verified locally, exported deterministically, and reviewed without default external trace storage. The project is aimed at evidence quality and portability, not hosted observability.

## Repo Layout

```text
cmd/atb/                  Go CLI entrypoint and subcommands
internal/                 Core Go implementation
  anchor/                 RFC 3161 anchoring
  archive/                Archive ledger support
  bundle/                 Bundle load/save/verify helpers
  canonicalize/           RFC 8785 canonical JSON
  encrypt/                AES-256-GCM bundle encryption
  event/                  Event type registry and event model
  profiles/               Embedded YAML profile schemas and validation
  sign/                   Ed25519 bundle and policy signing
  trust/                  Trust report generation
  verify/                 Integrity and profile verification
pkg/api/v1/               Viewer/dashboard API handlers and OpenAPI template
sdk/python/               Python SDK, tests, packaging metadata
sdk/typescript/           TypeScript SDK, tests, packaging metadata
web/                      Next.js dashboard used by `atb view --ui-experimental`
docs/                     Product, spec, security, workflow, and integration docs
examples/                 Public examples and sample bundles
schemas/                  JSON schema artefacts
scripts/                  Release and maintenance helpers
test/                     Golden, adversarial, integration, and performance tests
```

## Build And Run

Primary build:

```bash
go build -o atb ./cmd/atb
./atb help
./atb bundle new
./atb verify
```

If the change touches `atb view`, the embedded dashboard, or the viewer API, build the web export first:

```bash
cd web
npm ci
npm run build
cd ..
go build -o atb ./cmd/atb
./atb view --ui-experimental --no-open
```

## Test Commands

Go:

```bash
go test ./...
make hygiene-quick
make hygiene-full
```

Python SDK:

```bash
cd sdk/python
python3 -m venv venv
source venv/bin/activate
pip install -e .[dev]
pytest -v
```

TypeScript SDK:

```bash
cd sdk/typescript
npm ci
npm run typecheck
npm run build
npm test
```

Web dashboard:

```bash
cd web
npm ci
npm run lint
npm run typecheck
npm test
npm run test:e2e
npm run test:a11y
```

Use the narrower commands when the change is clearly scoped, but do not call work done if the relevant language surface is untested.

## Canonical Event And Profile System

The system has two sources of truth and they are not interchangeable:

- Event type strings and the `atb events` registry live in `internal/event/types.go`.
- The canonical event envelope and hash fields live in `internal/event/event.go` and `docs/spec-v1.0.md`.
- The actual built-in profile behaviour used by the verifier lives in `internal/profiles/templates/*.yaml`.
- Profile parsing and validation live in `internal/profiles/loader.go` and `internal/profiles/schema.go`.
- Runtime profile evaluation lives in `internal/verify/profiles.go` and `internal/verify/profile_loader.go`.

When extending the event or profile system:

1. Add or update event constants and registry metadata in `internal/event/types.go`.
2. Update the relevant embedded YAML profile template in `internal/profiles/templates/`.
3. Update the prose docs that describe the same behaviour: at minimum `docs/spec-v1.0.md`, `docs/profiles.md`, and any integration docs affected.
4. Add or update tests in Go and, if the event envelope changed, the Python and TypeScript schema-compat tests as well.

Do not treat `internal/event/types.go` alone as the profile source of truth. The verifier evaluates the YAML templates.

## Documentation Standards

- Use British English.
- Do not use em dashes.
- Prefer technical precision over marketing language.
- Keep README, spec, profile docs, SDK docs, and examples aligned with the shipped code.
- If behaviour is provisional, say so plainly instead of implying GA or attestation strength that does not exist.

## Known Issues And Sharp Edges

These are current repo realities. Do not "fix" them casually or by editing docs only:

- Version drift exists. The Go CLI reports `0.9.1-beta`, while `sdk/python/pyproject.toml`, `sdk/typescript/package.json`, and `web/package.json` still carry `0.9.0` prerelease versions.
- Profile drift exists. The verifier's built-in profiles are the YAML templates, but `internal/event/types.go` profile metadata is not fully aligned with them. In particular, `ai.request.received` and `ai.policy.decision` are under-mapped for several profiles, `data_export` still points at `data.export.*`, and `background_automation` is documented in the registry/spec as `ai.job.*` while the embedded template currently evaluates an `ai.action.*` control-plane flow.
- `docs/spec-v1.0.md` contains live spec text plus transitional notes. It already notes that `data_export` currently evaluates `ai.action.*`, and it still references a missing future doc at `docs/spec/bundle-push.md`.
- `docs/ci-known-issues.md` is not a finished document. It is a rough tracking note for the Windows runner flake and should not be expanded with invented details.
- Archived docs exist and are explicitly not maintained: `docs/guides/getting-started-v1.1.0.md`, `docs/security/actions-pinning-policy.md`, `docs/security/dependency-audit.md`, and `docs/security/known-vulns.md`.
- `docs/security.md` records an important nuance: TSA certificate-chain verification is implemented for anchor scoring, but the terminal summary wording is intentionally conservative. Do not over-claim TSA assurance in docs or output changes.

## Definition Of Done

A task in this repo is done only when all of the following are true:

- Relevant tests pass for every touched surface.
- Docs match code and examples still reflect the current behaviour.
- No new spec drift is introduced between code, templates, and prose docs.
- Any version or packaging changes are kept consistent across the affected artefacts.

## Do Not Rules

- Do not commit `patch-spec.patch`.
- Do not overstate TSA verification, CAS scoring, or compliance posture.
- Do not modify `docs/spec-v1.0.md` without also updating the corresponding profile templates and any affected registry/docs.
- Do not "fix" profile drift by editing only docs or only `internal/event/types.go`.
- Do not delete archived docs unless the change explicitly handles their references.
