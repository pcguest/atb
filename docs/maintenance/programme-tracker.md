# Programme tracker — ATB + Custos systematic pass

Living checklist for the phased programme that follows the June 2026 baseline
(`docs/maintenance/baseline-handoff.md`). One focused PR per phase. Updated each
working session. Shipped vs planned is kept honest: an item is only ticked when
it is merged on `main` and CI-green.

## Phase 0 — Hygiene & truth ✅ (complete)

- ✅ **Dependabot backlog cleared.** The ten stale PRs (#58, #67, #95, #96,
  #98–#103) were all red only because they predated the June baseline `#nosec`
  gosec annotations. Superseded in one rebased batch — **PR #109** (merged,
  `002a389`): AWS SDK + gax-go bumps, three SHA-pinned actions (trivy v0.36.0,
  docker/login v4.2.0, gh-pages v4.1.0), vitest 4 across `web` + `sdk/typescript`.
  Dependabot auto-closed the npm PRs and re-based the rest to newer increments.
- ✅ **Issue triage.** #77 (auto audit-capture profiles) and #79 (intake
  adapters) closed as shipped, with implementation pointers. #49 (reconciliation
  / assurance packs) re-scoped to just the unbuilt `atb reconcile` command (the
  assurance-pack half ships via `atb incident export`). #78 (signer-verification
  hardening) kept open — genuine P0.
- ✅ **Stale branches archived.** `audit/complete-atb` and
  `chore/main-toolchain-alignment` (content already on `main`) tagged
  `archive/*` and deleted. `private/demo-prep` kept (Phase 2 source).
  `chore/dependabot-batch` kept **but never to be merged** — it is a divergent
  branch that deletes ~34k lines; salvage by manual port only.
- ✅ **Front-door honesty checked.** Repo description already accurate
  ("Not a compliance certification service"); README EU AI Act Article 12 badge
  reads *logging*, not *compliant* — defensible per
  `docs/research/eu-ai-act-mapping.md`. No change needed.

## Phase 1 — Hardening ✅ (complete)

- ✅ **Windows advisory locks** — PR #111 (merged, `8d80f4f`). Real `LockFileEx`
  exclusive locking ported into `internal/bundle/lock_windows.go` (was a no-op
  placeholder), `openLockFile` extracted to shared `lock_open.go`, Windows
  contention tests added and **green on the Windows CI runner**. Salvaged by
  manual port from `chore/dependabot-batch` (never merged).
- ✅ Deps batch — folded into Phase 0 PR #109; routine future increments
  (#100–#103, #110) swept periodically, not chased per-commit.
- ✅ `scripts/check-support-matrix.sh` confirmed green (locally and in CI on #109/#111).

## Phase 2 — ATB demonstrability (P0) 🚧 (in progress)

**Baseline correction:** `private/demo-prep` is 64 commits behind main (its 10
viewer commits are *polish*, not the wiring) and the api-client hash-token fix
already landed — so the real P0 was the orphaned components.

- 🚧 **Mount session/schema views** (`feat/wire-session-schema-views`).
  `SessionList` + `SchemaStatus` were built, unit-tested, but imported by nothing
  and used raw `fetch()` (unauthenticated). Rewired to the authenticated
  api-client (added `useSchemaStatusQuery` + zod schema for `SchemaStatusResponse`)
  and mounted on `/sessions`, replacing the static placeholder. Tests + page
  mount test green; lint/typecheck clean; `spec-dashboard.md`/`public-surface.md`/
  CHANGELOG updated to shipped.
- ⬜ Cherry-pick the 10 `private/demo-prep` polish commits onto main (separate PR).
- ⬜ `ActorSessions` / `RoleSelector` mounting (still orphaned).
- ⬜ tamper→verify→profile-gaps smoke/E2E.
- ⬜ `trust-score.ts` vs CAS labelling — Viewer Health rename already landed in a
  prior pass; re-confirm no residual inconsistency.

## Phase 3 — Custos reference layer ⬜

Per `docs/custos-handoff.md` Receipt MVP: implement `custos/registry/`,
`onboarding/`, `discovery/` with auth tests; harden `custosd` ops doc (TLS,
token rotation, max-ingest); E2E `atb intercept --custos` → ingest →
attestation; keep `test/custos/conformance_test.go` green. Hold the scope
guardrails in `docs/research/capture-and-custos-scope.md` (no generative
`insights`, no hosted workflow in `oversight`/`onboarding`).

## Phase 4 — Capture & integration (P2) ⬜

Wire `pkg/otel` decode to the receiver path; thin opt-in SDK auto-capture
wrappers (Claude/OpenAI) on existing emitters, profile-bound, blind spots
documented; align `atb.capture.scope` with the EU AI Act mapping as *support*,
not certification.

## Phase 5 — Release & narrative ⬜

**Truth gap to close:** README says `Current release: v1.13.0` and the CHANGELOG
is consolidated to v1.13.0, but the latest git tag is `v1.12.0` and the latest
GitHub Release is `v1.9.0`. Run the full release gates, cut tags/Releases for
v1.10–v1.13, then map `docs/research/*` to public roadmap bullets without
promising hosted features in the ATB repo.

## Out of scope (not in the ATB repo)

Multi-tenant auth, billing, legal-hold UI, SIEM platform, KMS product,
compliance-certification claims, production transparency-log service.
