# Baseline handoff — post-hygiene feature work

Status: written after the June 2026 baseline-trustworthiness pass (`chore/unblock-ci` + release/doc honesty commits).

> **Update (2026-06-05):** the systematic programme pass (PRs #109–#116) has since
> closed most of the "what remains" list below — viewer `/sessions` mount, the
> Custos receipt + digest registry, the `v1.10.0`–`v1.13.0` GitHub Releases, and
> the Windows advisory-lock implementation all shipped. **`docs/maintenance/programme-tracker.md`
> is the live checklist; treat this file as the original baseline record.** The
> items still genuinely open are called out inline below.

## What baseline work closed

- **CI docs-sync:** README again satisfies `ci.yml` greps (`Current release`, `atb bundle new`, `--profile atb.profile.policy_decision`).
- **Security gate:** gosec HIGH findings in `internal/proxy/push.go`, `internal/proxy/forward.go`, `cmd/atb/view.go`, and `cmd/atb/incident.go` addressed (validation + justified `#nosec`).
- **Web CI:** Vitest (`cd web && npm test -- --run`) runs on all three OS matrix jobs, not only ubuntu `quality-evidence.sh`.
- **Release truth:** `[Unreleased]` consolidated to `v1.13.0`; version strings bumped; duplicate `## [v1.5.0]` CHANGELOG heading corrected to `v1.4.1`.
- **Doc honesty:** `docs/spec-dashboard.md`, `docs/public-surface.md`, and CHANGELOG no longer claim SessionList/SchemaStatus are wired when they are not.

## What remains for the feature prompt (do not start in baseline PRs)

### P0 — Viewer wiring (`docs/spec-dashboard.md`)

1. ✅ `SessionList` + `SchemaStatus` mounted on `web/app/sessions/page.tsx` (#112), rewired to the authenticated `web/lib/api-client.ts`.
2. ⬜ `ActorSessions` + `RoleSelector` still orphaned — mount via the same authenticated client (replace their raw `fetch()`).
3. ✅ Viewer Health vs verifier CAS labelling reconciled in a prior pass (`web/lib/trust-score.ts`).
4. ⬜ Live E2E: tamper bundle, profile verify + provability gaps, privacy reveal append.

### P1 — Custos reference layer (`docs/custos-handoff.md`, `docs/custos/ui-spec.md`)

Implement custody primitives before UI fantasy:

- ✅ `custos/registry/` — receipt + digest registry with real lookup semantics + tests (#113), surfaced over HTTP as `GET /receipts/by-hash` (#114).
- ⛔ `custos/onboarding/` — **out of scope** (multi-tenant account provisioning per `AGENTS.md` + `docs/research/capture-and-custos-scope.md`); kept as a boundary-documenting stub.
- ⬜ `custos/discovery/` — deferred (tool-signature scaffold; no real consumer yet).
- ⬜ Minimal read-only auditor page reusing `ProfileCAS` patterns (no SSO/billing).

Scaffolds only today: `discovery`, `onboarding`, `oversight`, `insights` (`registry` is now implemented).

### P2 — Capture completeness (`docs/roadmap.md`)

- Claude/OpenAI SDK automatic capture callbacks
- EU AI Act Art.12 enforcement in automatic capture path
- ✅ OTLP/JSON decode in `pkg/otel/` (`DecodeTraceJSON`, shipped); receiver wire-up + protobuf/gRPC transport still open

## GitHub hygiene still manual

- ✅ GitHub **Releases** cut for `v1.10.0`–`v1.13.0` (v1.13.0 marked Latest); `v1.14.0` is the next tag (this PR).
- ✅ Issues triaged: #77/#79 closed as shipped, #49 narrowed to `atb reconcile`, #78 kept.
- ⬜ Open Dependabot PRs (#110, #103, #102, #101, #100, #67, #58) — batch-supersede next.
- ✅ Repo description already accurate (no "EU AI Act ready" oversell).
- ✅ Stale branches archived (`audit/complete-atb`, `chore/main-toolchain-alignment`); `private/demo-prep` + `chore/dependabot-batch` kept as cherry-pick sources, **never merged**.

## Release gates (unchanged)

```bash
GOCACHE=$(pwd)/.gocache/dev GOTOOLCHAIN=go1.26.3 make test-golden
GOCACHE=$(pwd)/.gocache/dev GOTOOLCHAIN=go1.26.3 go test ./... -count=1
GOCACHE=$(pwd)/.gocache/dev GOTOOLCHAIN=go1.26.3 make hygiene-quick
ATB_SKIP_TAG_CHECK=1 bash scripts/check-versions.sh
bash scripts/check-support-matrix.sh
```

Tag `v1.14.0` from `main` only after CI green and maintainer review per `docs/release.md`. (`v1.13.0` is already tagged and released.)
