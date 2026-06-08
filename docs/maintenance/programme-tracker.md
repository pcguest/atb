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

## Phase 2 — ATB demonstrability (P0) ✅ (complete — local main)

**Baseline correction:** `private/demo-prep` is 64 commits behind main (its 10
viewer commits are *polish*, not the wiring) and the api-client hash-token fix
already landed — so the real P0 was the orphaned components.

- ✅ **Mount session/schema views** — PR #112 (merged).
  `SessionList` + `SchemaStatus` were built, unit-tested, but imported by nothing
  and used raw `fetch()` (unauthenticated). Rewired to the authenticated
  api-client (added `useSchemaStatusQuery` + zod schema for `SchemaStatusResponse`)
  and mounted on `/sessions`, replacing the static placeholder. Tests + page
  mount test green; lint/typecheck clean; `spec-dashboard.md`/`public-surface.md`/
  CHANGELOG updated to shipped.
- ✅ **C3 — demo-prep viewer polish cherry-picked (additive-only)** — merged to
  local main (`chore/demo-prep-polish`). `private/demo-prep` was 86 commits behind
  main, which had independently reimplemented much of the polish, so only the four
  genuinely-additive commits were ported: hash truncation + click-to-copy
  (`HashValue`/`hash-display.ts`), the reveal-appends-audit-event warning,
  role-aware event labels (`event-labels.ts` + `TraceTimeline`), and CAS sub-score
  definitions (`cas-subscores.ts`). **Skipped as superseded** (already on main):
  provability-gaps surfacing, re-run-verify button. **Skipped as cosmetic/
  obsolete**: minimap z-index, polling-colour, edge-labels, demo screenshots.
  `private/demo-prep` itself is **never merged**. Web 58/58, typecheck, lint green.
- ✅ **`ActorSessions` / `RoleSelector` mounting** — merged to local main
  (`5badf10`, C1). `ActorSessions` rewired raw `fetch` → authenticated
  `useActorSessionsQuery` (+ `actorSessionsResponseSchema`) and mounted on
  `/sessions`; `RoleSelector` mounted in the `/view` header, driving the
  role-gated panels. Web tests 58/58, lint/typecheck clean, `public-surface.md`
  rows flipped to shipped.
- ✅ **tamper→verify→profile-gaps smoke** — merged to local main (`427a957`, C2).
  `TestProfileGapsOnIntactButIncompleteBundle` proves the *completeness* failure
  mode (intact chain, `Pass=false`, obligation gaps, CAS below complete) is
  distinct from the *integrity* mode (`TestTamperDetection`).
- ✅ `trust-score.ts` vs CAS labelling — re-confirmed clean: Viewer Health Score
  (liveness) stays distinct from verifier CAS obligation grade; no residual
  conflation.

## Phase 3 — Custos reference layer ✅ (complete — in-repo scope)

Per `docs/custos-handoff.md` Receipt MVP. Hold the scope guardrails in
`docs/research/capture-and-custos-scope.md` (no generative `insights`, no hosted
workflow in `oversight`/`onboarding`).

- ✅ **`custos/registry/`** — PR #113 (merged, `c5032ce`). Repurposed the inert
  tool-signature stub into the handoff's **receipt + digest registry**:
  `InMemoryRegistry` indexes receipts by ID and by bundle hash (the reverse
  lookup the receipt store lacks), idempotent `Register` upsert, `Build` from a
  store, deterministic ordering, race-tested. Tool-sig concept deferred with
  `discovery`.
- ✅ **Wire registry into `custosd`** — PR #114 (merged). Authenticated
  `GET /receipts/by-hash?bundle_hash=<hash>` backed by the registry (built from the
  receipt store per request) — the registry's real consumer. Handler +
  routing-precedence tests; README/CHANGELOG updated.
- ⛔ `custos/onboarding/` — **out of scope** per AGENTS.md + research guardrail
  (multi-tenant account provisioning). Left as a boundary-documenting stub.
- ⛔ `custos/discovery/`, `oversight/`, `insights/` — **out of scope / deferred**
  (no real consumer; hosted multi-tenant workflows per the research guardrail).
  Left as boundary-documenting stubs — not in-repo TODOs.
- ✅ **Custos operator doc pass (E)** — merged to local main (`docs/custos-operator`).
  `custos/README.md` gained a self-contained "Operating `custosd`" section
  (bind/auth, TLS-at-proxy, token rotation, `--max-ingest-bytes`, storage)
  consistent with the authoritative checklist in `docs/custos-handoff.md`; the
  handoff's token-rotation item is now an actionable procedure (single static
  token, no overlap window) rather than only noting the gap.
- ✅ **E2E ingest → attestation → digest lookup** (`test/custos-e2e-ingest-attestation`).
  Drives the real `newMux` with a real ATB fixture bundle through the full custody
  path; proves the wire contract `atb intercept --custos` relies on. (Module
  isolation + flaky-binary risk make an in-process daemon test the robust choice
  over an `atb intercept` exec test; the ATB push side is covered in
  `internal/proxy/push_test.go`.)

## Phase 4 — Capture & integration (P2) ✅ (complete — local main)

Wire `pkg/otel` decode to the receiver path; thin opt-in SDK auto-capture
wrappers (Claude/OpenAI) on existing emitters, profile-bound, blind spots
documented; align `atb.capture.scope` with the EU AI Act mapping as *support*,
not certification.

- ✅ **OTLP/JSON ingest wired** — merged to local main (`0894e7c`, D).
  `DecodeTraceJSON` → `Receiver.ReceiveJSON` (decode→translate every span,
  aggregate) → `atb import otel` subcommand appends translated spans to a bundle
  with trace linkage and retrospective provenance. gRPC/protobuf transport stays
  deferred (would pull an OpenTelemetry proto dependency). `public-surface.md`
  OTLP row flipped to shipped.
- ✅ **D2** — opt-in Claude/OpenAI SDK capture adapters — merged to local main
  (`feat/sdk-capture-adapters`). `wrapOpenAI`/`wrapAnthropic` (TS,
  `sdk-capture.ts`) and `wrap_openai`/`wrap_anthropic` (Python, `atb.sdk_capture`)
  wrap the direct clients' `create` method and record request/response/tool-calls/
  error through the existing `atbMiddleware` / `ATBCallbackHandler` recorders — no
  second emit path, no hard SDK dependency. Opt-in, privacy-moded, profile-bound.
  Blind spots documented: streaming (`stream:true`) raises → use `atb intercept`;
  only chat/messages create is instrumented. TS 95/95, Python 125/125, golden
  parity green.

## Phase 5 — Release & narrative ✅ (complete — in-repo scope; tag/push deferred to budget)

- ✅ **Release-truth gap closed.** `v1.13.0` is now tagged at `863303d` (#104 —
  where the `[v1.13.0]` CHANGELOG heading and `version = "1.13.0"` were
  finalised; #105–#108 were already `[Unreleased]` and correctly excluded).
  GitHub Releases created for the tagged-but-unreleased `v1.10.0`, `v1.11.0`,
  `v1.12.0` and the new `v1.13.0` (marked **Latest**), from their CHANGELOG
  sections. `scripts/check-versions.sh` passes — all version strings agree with
  the latest tag. README's "Current release: v1.13.0" is now true.
- ✅ **Cut v1.14.0** (`release/v1.14.0`, merged to local main `fb2a19e`). The
  accumulated `[Unreleased]` work (deps batch, Windows `LockFileEx`, viewer
  `/sessions` mount, Custos registry + by-hash + E2E, `pkg/otel` `DecodeTraceJSON`,
  signing PolicyStore) is a minor release — new features, no hash/schema break.
  CHANGELOG `[Unreleased]` → `## [v1.14.0] - 2026-06-05`; all nine version strings
  bumped 1.13.0 → 1.14.0; README "Current release:" line bumped (CI docs-sync gate
  checks it, `check-versions.sh` does not); stale "no-op on Windows" skip reason
  in `advisory_lock_testsupport_test.go` corrected. **Local-only:** committed and
  merged on local `main`; tag/push/GitHub Release deferred until budget restored.
- ✅ **v1.14.0 stack integrated on local main** (`899b36b`). Merge order
  A(`fb2a19e`)→B(`e8ae18c`)→C1(`5badf10`)→C2(`427a957`)→D(`0894e7c`) +
  public-surface shipped-status follow-up. Full local gate suite green: golden
  parity, `go test ./...` (36 pkgs), `custos` race (9 pkgs), web vitest 58/58,
  lint, typecheck, `check-versions` (1.14.0), support-matrix, docs-sync greps.
  Pre-existing Trivy Docker CVE (Go 1.26.3 stdlib `CVE-2026-42504`) deferred to a
  Go 1.26.4 toolchain bump — not a stack blocker.
- ✅ **Map `docs/research/*` to roadmap** (`docs/research-to-roadmap`, merged) —
  added a "Research & design notes (forward-looking — not commitments)" section to
  `docs/roadmap.md` mapping the transparency-log, EU AI Act, and capture/scope
  notes to in-repo reference direction. Honest framing held: a single-operator
  transparency log is not equivocation-resistant without witness cosignatures; AI
  Act evidence is *support, not certification*; no hosted-service promises in this
  repo.

### Programme status — Phases 0–5 complete for in-repo scope

All six phases are done for what lives in `pcguest/atb`. Remaining ATB work is
**bugfixes, golden parity, and dependency hygiene only** (e.g. the deferred Go
1.26.4 toolchain bump for the pre-existing Trivy stdlib CVE; optional `atb
reconcile` per the narrowed #49). The v1.14.0 stack + D2 + C3 + E + narrative are
integrated on local `main`; **tag/push/GitHub Release are deferred until the
Actions budget is restored**. Net-new feature direction now belongs to the
**Custos product** (new repo / hosted), which depends on ATB as a frozen
contract — not to more code in the in-repo `custos/` stubs.

## Out of scope (not in the ATB repo)

Multi-tenant auth, billing, legal-hold UI, SIEM platform, KMS product,
compliance-certification claims, production transparency-log service.
