# Final pre-flight checklist (v1.15.0)

The last page to read before authorising live execution. Every item is either
proven here, or flagged as a decision or a gated step that is yours. Nothing
below has been pushed, published, deployed, or made public.

## Capability inventory (what this environment can actually do)

| Capability | State | Label |
| --- | --- | --- |
| GitHub auth (pcguest) | Logged in; scopes `repo`, `workflow`, `gist`, `read:org` | AGENT-capable, but every use is a gated irreversible step |
| Push branches | Both remotes reachable; `git push --dry-run` clean for both staged branches | Gated |
| Change repo visibility | `repo` scope allows it | Gated, HUMAN decision |
| Cut a GitHub release | `repo` scope allows it | Gated |
| Publish to PyPI | `twine` and `build` not installed; no credentials proven | HUMAN |
| Publish to npm | `npm whoami` returns 401, not logged in | HUMAN |
| Deploy Mortise site | No deploy target configured (only `ci.yml`); static files only | HUMAN |

Note: although the GitHub token could technically push, release, and flip
visibility from here, all of those are on the irreversible list and will not run
without your explicit "go".

## Decision resolved: GitHub private vulnerability reporting

- Security disclosure on both repos points to GitHub's private vulnerability
  reporting (the Security tab). No email is exposed on a security surface.
- The conduct report uses the same private channel. The Mortise site sales CTA
  uses the proton stopgap (it is a procurement contact, not a vuln report).
- New prerequisite before going public: enable Private vulnerability reporting
  in each repository's Settings (runbook section 5).
- The personal `proton.me` address otherwise stays only in SDK author metadata,
  by design.

## Proven, with evidence

| Brief requirement | Proven where |
| --- | --- |
| Cross-language verifiers agree | ACCEPTANCE.md, `make test-golden` (Go, Python 8, TypeScript 8) |
| Full Go suite green (ATB and Mortise) | ACCEPTANCE.md, 40 and 9 packages, 0 fail |
| Hygiene gate green | ACCEPTANCE.md, `make hygiene-quick` exit 0 |
| Version markers all 1.15.0 | ACCEPTANCE.md; the built binary prints `atb 1.15.0` |
| Clean clone builds (embed fix) | ACCEPTANCE.md; the regression is closed and re-proven |
| Reveal does not mutate evidence | ACCEPTANCE.md; sha256 identical before and after; sidecar verifies |
| Incident forensics from the bundle alone | ACCEPTANCE.md; `tool_without_approval` and action error fire |
| Tamper gives verify exit 2, intact exit 0 | ACCEPTANCE.md and re-proven this pass (SDK-built 3-record bundle) |
| Default-drift guard fires | ACCEPTANCE.md; drifting the Python default fails the golden test |
| `reviewer_identities` earned, additive, frozen | ACCEPTANCE.md; strict-additive vs v1.14.5, SHA-256 pinned |
| Mortise conforms and end-to-end works | ACCEPTANCE.md; e2e script, live `POST /ingest 201` receipt |
| History rewrite safe on a clone | `docs/maintenance/history-rewrite-plan.md`; 121 to 4 MiB, gitleaks clean |
| Article 12 mapping honest, no certification | `docs/compliance/article-12-mapping.md` |
| Mortise site self-contained, links resolve | Audited this pass: 0 external assets, 0 em dashes, all anchors resolve |

## Launch collateral ready (no live button)

- Release notes, both PR descriptions, GitHub release body, and two
  announcements: `LAUNCH-COLLATERAL.md`.
- Post-launch external verification script: `scripts/post-launch-verify.sh`
  (syntax and shellcheck clean; the tamper-to-exit-2 path proven locally).
- Per-commit review: `RELEASE-REVIEW-v1.15.0.md`. Ordered runbook:
  `LAUNCH-RUNBOOK.md`. Re-proved claims: `ACCEPTANCE.md`.

## Honesty flags to clear before you ship

- The CHANGELOG dates v1.15.0 as `2026-06-15`, two days before today and before
  it has shipped. Set it to the actual tag date.
- Decide Path A or Path B on the history rewrite (LAUNCH-RUNBOOK.md, Decision 0).
  Path A ships with signatures intact and defers the rewrite; recommended,
  because gitleaks is clean and the bloat is not a secret.

## Gated execution order (each waits for your "go")

Path A: push branches, review and fast-forward `main`, tag `v1.15.0` once, make
repos public, publish the SDKs (HUMAN: install `build`/`twine`, `npm login`),
deploy the Mortise site, then run `scripts/post-launch-verify.sh`.
