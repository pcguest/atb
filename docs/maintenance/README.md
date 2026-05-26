# Maintenance index

This directory contains maintainer-facing planning documents for turning ATB into a finalised developer-first automatic audit-capture toolkit.

These documents do not replace the product specifications. They translate the current repository state into execution goals, cleanup tracks, and acceptance criteria for maintainers and Codex sessions.

## Core documents

| Document | Purpose |
| --- | --- |
| [`repo-cleanup-roadmap.md`](repo-cleanup-roadmap.md) | Repository-wide cleanup plan covering architecture, CI/CD, security, docs, SDKs, release trust, and automatic audit capture. |
| [`automatic-audit-capture-definition.md`](automatic-audit-capture-definition.md) | Defines what automatic capture means, what ATB can prove, what it cannot prove, and how audit profiles should verify completeness. |
| [`codex-execution-brief.md`](codex-execution-brief.md) | Implementation brief for Codex or maintainers, including stream ordering, guardrails, and final definition of done. |
| [`codex-goals.md`](codex-goals.md) | Persistent Codex Goal templates for repo cleanup, trust hardening, automatic capture, release provenance, and docs alignment. |

## How to use this directory

1. Start with `repo-cleanup-roadmap.md` to understand the full current-to-final path.
2. Use `automatic-audit-capture-definition.md` before implementing any feature that claims automatic capture, audit coverage, or completeness.
3. Use `codex-execution-brief.md` when opening an implementation thread or planning a PR stack.
4. Use `codex-goals.md` to start Codex with a persistent objective rather than a one-off prompt.

## Current final-state thesis

ATB is finalised when it can:

- capture supported workflow activity through documented live intake routes;
- write that activity through the same append-only bundle boundary;
- verify bundle integrity and supported signer paths offline;
- distinguish live capture from retrospective import;
- report audit-profile completeness separately from tamper evidence;
- present evidence through CLI, SDK, viewer, and export flows;
- ship with a release trust story that matches the sensitivity of the product.

## Maintainer warning

Do not let cleanup work blur the product guarantee.

ATB proves the integrity of recorded evidence. Automatic capture and profile verification are the work needed to improve audit coverage. Even when those features are implemented, ATB should still report blind spots explicitly rather than implying omniscient capture.
