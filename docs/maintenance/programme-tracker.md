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

## Phase 3 — Custos reference layer 🚧 (in progress)

Per `docs/custos-handoff.md` Receipt MVP. Hold the scope guardrails in
`docs/research/capture-and-custos-scope.md` (no generative `insights`, no hosted
workflow in `oversight`/`onboarding`).

- ✅ **`custos/registry/`** — PR #113 (merged, `c5032ce`). Repurposed the inert
  tool-signature stub into the handoff's **receipt + digest registry**:
  `InMemoryRegistry` indexes receipts by ID and by bundle hash (the reverse
  lookup the receipt store lacks), idempotent `Register` upsert, `Build` from a
  store, deterministic ordering, race-tested. Tool-sig concept deferred with
  `discovery`.
- 🚧 **Wire registry into `custosd`** (`feat/custosd-receipts-by-hash`).
  Authenticated `GET /receipts/by-hash?bundle_hash=<hash>` backed by the registry
  (built from the receipt store per request) — the registry's real consumer.
  Handler + routing-precedence tests; README/CHANGELOG updated.
- ⛔ `custos/onboarding/` — **out of scope** per AGENTS.md + research guardrail
  (multi-tenant account provisioning). Left as a boundary-documenting stub.
- ⬜ `custos/discovery/` — deferred (tool-signature scaffold; no real consumer yet).
- ⬜ Harden `custosd` operator doc (TLS, token rotation, max-ingest) — the
  hardening checklist in `docs/custos-handoff.md` already covers these; light touch.
- ✅ **E2E ingest → attestation → digest lookup** (`test/custos-e2e-ingest-attestation`).
  Drives the real `newMux` with a real ATB fixture bundle through the full custody
  path; proves the wire contract `atb intercept --custos` relies on. (Module
  isolation + flaky-binary risk make an in-process daemon test the robust choice
  over an `atb intercept` exec test; the ATB push side is covered in
  `internal/proxy/push_test.go`.)

## Phase 4 — Capture & integration (P2) ⬜

Wire `pkg/otel` decode to the receiver path; thin opt-in SDK auto-capture
wrappers (Claude/OpenAI) on existing emitters, profile-bound, blind spots
documented; align `atb.capture.scope` with the EU AI Act mapping as *support*,
not certification.

## Phase 5 — Release & narrative 🚧 (in progress)

- ✅ **Release-truth gap closed.** `v1.13.0` is now tagged at `863303d` (#104 —
  where the `[v1.13.0]` CHANGELOG heading and `version = "1.13.0"` were
  finalised; #105–#108 were already `[Unreleased]` and correctly excluded).
  GitHub Releases created for the tagged-but-unreleased `v1.10.0`, `v1.11.0`,
  `v1.12.0` and the new `v1.13.0` (marked **Latest**), from their CHANGELOG
  sections. `scripts/check-versions.sh` passes — all version strings agree with
  the latest tag. README's "Current release: v1.13.0" is now true.
- ⬜ This session's `[Unreleased]` work (deps batch, Windows locks, viewer
  wiring, Custos registry + by-hash + E2E) is a future **v1.14.0** (minor; new
  features, no hash/schema break) — cut when ready.
- ⬜ Map `docs/research/*` to public roadmap bullets without promising hosted
  features in the ATB repo.

## Out of scope (not in the ATB repo)

Multi-tenant auth, billing, legal-hold UI, SIEM platform, KMS product,
compliance-certification claims, production transparency-log service.
