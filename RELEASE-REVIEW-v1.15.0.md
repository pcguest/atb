# Per-commit review: v1.15.0 and the Custos site

Read this before you push. Every commit is GPG-signed (verified good). The
release was assembled from an inherited 140-file working tree, audited and
fixed, then grouped by subsystem. Because the inherited work and the fixes share
files, a few files span two concerns; those are called out so nothing is hidden.

## ATB, branch `release/v1.15.0` (15 commits, oldest first)

Commits 1 to 11 are the product release, grouped by subsystem. Commits 12 to 15
are the launch documentation, listed in their own section below.

1. `feat(schema): additive reviewer-identity and retention event schema`
   The v1.15.0 schema surface: an optional `identity_evidence` object on
   approval and privileged-action events, three `data.retention.*` event types,
   and an optional `reviewer_identities` array on the verify report. The
   `verify.report.v1` schema file revision goes `.1` to `.2`; `report_version`
   stays `verify.report.v1`. Strict additive (proof in ACCEPTANCE.md).
   Shared-file note: `schemas/event.v1.json` and `internal/event/types_generated.go`
   carry both the identity and retention additions, since they are one
   generated surface.

2. `feat(sdk): reviewer identity evidence and cross-SDK default pinning`
   Mirrors identity evidence into the Python and TypeScript SDKs. Adds the
   shared `policy_decision_defaults.json` fixture asserted by both SDK golden
   tests, so a drift in either SDK's PolicyDecisionRecorder defaults fails
   `make test-golden`. The fixture is preventative; the defaults had not drifted.

3. `fix(viewer): record reveals to a sidecar, never the authoritative bundle`
   The load-bearing trust fix. Reveals now write to a `<bundle>.reveals` sidecar
   with its own chain. Shared-file note: `pkg/api/v1/handlers.go` also carries
   the inherited viewer RBAC and JWT additions; the commit message says so.

4. `feat(cli): compliance pack, identity evidence, and conforming Custos push`
   The `atb compliance pack` command, identity evidence in reports, and the
   Custos push fix (post the bundle to `/ingest`, return the receipt; the old
   client hit a non-existent `/bundle`). Also bumps the CLI version constant to
   1.15.0. Shared-file note: `cmd/atb/main.go` holds both command wiring and the
   version constant.

5. `feat(custos): RBAC, JWT auth, and S3 Object Lock WORM store`
   The in-repo `custos/` reference module. Self-contained under `custos/`.

6. `feat(web): viewer event families, labels, and reveal-sidecar contract note`
   Viewer event families and the API contract note that a reveal writes to the
   sidecar. Self-contained under `web/`.

7. `test(integration): cover v1.15.0 cross-SDK and phase 9 surfaces`
   Integration coverage for the new surfaces. Test-only.

8. `docs: v1.15.0 docs, Article 12 mapping, reveal and Custos push corrections`
   The Article 12 one-pager, reveal-sidecar and Custos-push doc corrections, the
   open-core boundary table, forward-looking-copy removal, and the README
   copy-hygiene pass. Docs only.

9. `chore(release): prepare v1.15.0`
   SDK version sync to 1.15.0, the CHANGELOG, and release-workflow and toolchain
   updates. Shared-file note: `CHANGELOG.md` aggregates every concern above.

10. `fix(build): restore web/out embed placeholder so clean checkouts build`
    Found during acceptance. The inherited tree deleted `web/out/placeholder.txt`,
    so `//go:embed web/out/*` matched nothing and a clean clone or `go install`
    failed to build. Restores the placeholder. One file.

11. `chore(security): parameterise the disclosure contact and correct the rewrite runbook`
    Replaces the personal disclosure address with one placeholder token across
    all security-contact surfaces, and corrects the history-rewrite command to
    `--path-glob '.gocache*'` with the dry-run numbers. Contact and maintenance
    docs only.

## ATB launch-documentation commits (12 to 15, oldest first)

These commits add and correct the launch paperwork itself. They touch no product
code and ship alongside the release for transparency.

12. `docs(release): add launch runbook, per-commit review, and acceptance evidence`
    The first three launch artefacts: `LAUNCH-RUNBOOK.md`, this review, and
    `ACCEPTANCE.md`. Documentation only.

13. `style(docs): drop em dash from the Article 12 mapping title`
    One title punctuation fix in `docs/compliance/article-12-mapping.md`.

14. `docs(runbook): correct the TypeScript SDK publish step`
    Corrects the npm publish step to `npm ci && npm run build && npm publish`,
    since the package builds with `tsup` and has no `prepublishOnly` hook.

15. `docs(release): add launch collateral, post-launch verification, and pre-flight checklist`
    `LAUNCH-COLLATERAL.md` (release notes, PR descriptions, GitHub release body,
    announcements), `scripts/post-launch-verify.sh` (external verification, the
    tamper-to-exit-2 path proven locally), and `PRE-FLIGHT-CHECKLIST.md`. This
    commit also corrects the commit count in `LAUNCH-RUNBOOK.md` from 11 to 15.

The branch is 15 signed commits ahead of `main`, all with good signatures.

## Custos, branch `site/marketing-front` (2 commits, oldest first)

1. `docs(site): add the public Custos marketing front`
   The self-contained static `site/index.html` and `site/README.md`. No build
   step, no JavaScript, no external assets.

2. `chore(site): parameterise contact and align the open-core table to the ATB README`
   Unifies the disclosure contact to the placeholder token and makes the
   open-core table match the ATB README content exactly (seven free, seven
   Custos, same wording).

## Files that span two concerns (the honest list)

- `schemas/event.v1.json`, `internal/event/types_generated.go`: identity
  evidence and retention, one generated surface (commit 1).
- `pkg/api/v1/handlers.go`: the reveal sidecar fix and inherited viewer
  RBAC/JWT (commit 3).
- `cmd/atb/main.go`: command wiring and the version constant (commit 4).
- `CHANGELOG.md`: aggregates all concerns (commit 9).

Interactive hunk staging is not available in this environment, so these files
could not be split further without rewriting them. Grouping is by subsystem.
