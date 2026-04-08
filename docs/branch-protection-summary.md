# Branch Protection Summary (Enterprise Pilot)

This document defines the minimum branch protection baseline for `main` during enterprise pilots.

## Required Settings

- Require pull requests before merge (no direct pushes to `main`)
- Require at least 1 approving review
- Dismiss stale approvals when new commits are pushed
- Require conversation resolution before merge
- Require status checks to pass before merge
- Restrict who can push to `main` (maintainers only)

## Required Status Checks

Use required checks from current workflows:

- `CI`
- `Security Gate`

If check names change, update branch protection to the exact new check names.

## Admin and Force-Push Policy

- Do not allow force pushes on `main`
- Do not allow branch deletion on `main`
- Apply restrictions to admins (no bypass in normal operation)

## Pilot Incident Handling

If a check flakes (for example Windows-only CI flake), follow `docs/ci-known-issues.md` and require a rerun result plus note in PR comments before merge.
