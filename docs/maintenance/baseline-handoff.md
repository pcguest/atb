# Baseline handoff — post-hygiene feature work

Status: written after the June 2026 baseline-trustworthiness pass (`chore/unblock-ci` + release/doc honesty commits).

## What baseline work closed

- **CI docs-sync:** README again satisfies `ci.yml` greps (`Current release`, `atb bundle new`, `--profile atb.profile.policy_decision`).
- **Security gate:** gosec HIGH findings in `internal/proxy/push.go`, `internal/proxy/forward.go`, `cmd/atb/view.go`, and `cmd/atb/incident.go` addressed (validation + justified `#nosec`).
- **Web CI:** Vitest (`cd web && npm test -- --run`) runs on all three OS matrix jobs, not only ubuntu `quality-evidence.sh`.
- **Release truth:** `[Unreleased]` consolidated to `v1.13.0`; version strings bumped; duplicate `## [v1.5.0]` CHANGELOG heading corrected to `v1.4.1`.
- **Doc honesty:** `docs/spec-dashboard.md`, `docs/public-surface.md`, and CHANGELOG no longer claim SessionList/SchemaStatus are wired when they are not.

## What remains for the feature prompt (do not start in baseline PRs)

### P0 — Viewer wiring (`docs/spec-dashboard.md`)

1. Mount `SessionList` and/or `ActorSessions` on `/view/` or replace `web/app/sessions/page.tsx` stub.
2. Fix orphan components to use `web/lib/api-client.ts` session token (raw `fetch()` fails against real `atb view`).
3. Mount `SchemaStatus` and `RoleSelector`; prove auditor/executive role hiding in browser + tests.
4. Live E2E: tamper bundle, profile verify + provability gaps, privacy reveal append.
5. Resolve Viewer Health vs verifier CAS labelling (`web/lib/trust-score.ts`).

### P1 — Custos reference layer (`docs/custos-handoff.md`, `docs/custos/ui-spec.md`)

Implement custody primitives before UI fantasy:

- `custos/registry/` — real lookup/update semantics + tests
- `custos/onboarding/` — minimal API key provisioning HTTP routes
- `custos/discovery/` — scan result storage with auth tests
- Minimal read-only auditor page reusing `ProfileCAS` patterns (no SSO/billing)

Scaffolds only today: `discovery`, `registry`, `onboarding`, `oversight`, `insights`.

### P2 — Capture completeness (`docs/roadmap.md`)

- Claude/OpenAI SDK automatic capture callbacks
- EU AI Act Art.12 enforcement in automatic capture path
- OTLP decode in `pkg/otel/`

## GitHub hygiene still manual

- Create GitHub **Releases** for tags `v1.10.0`–`v1.13.0` if not cut during baseline (tags exist; latest release was `v1.9.0` as of June 2026).
- Triage open Dependabot PRs and stale issues `#76`–`#79`, `#81`, `#48`–`#49`.
- Update repo description: remove "EU AI Act ready" oversell; use AGENTS.md tone.
- Delete or document stale branches: `audit/complete-atb`, `private/demo-prep`, `chore/dependabot-batch`.

## Release gates (unchanged)

```bash
GOCACHE=$(pwd)/.gocache/dev GOTOOLCHAIN=go1.26.3 make test-golden
GOCACHE=$(pwd)/.gocache/dev GOTOOLCHAIN=go1.26.3 go test ./... -count=1
GOCACHE=$(pwd)/.gocache/dev GOTOOLCHAIN=go1.26.3 make hygiene-quick
ATB_SKIP_TAG_CHECK=1 bash scripts/check-versions.sh
bash scripts/check-support-matrix.sh
```

Tag `v1.13.0` from `main` only after CI green and maintainer review per `docs/release.md`.
