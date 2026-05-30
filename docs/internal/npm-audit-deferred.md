# Deferred npm audit advisories

This document records moderate-severity npm audit findings that are intentionally
deferred. Each entry states why the upgrade is blocked and what would unblock it.

Run `npm audit` in `web/` to refresh advisory metadata before release review.

## postcss (via Next.js)


| Field          | Value                                                                    |
| -------------- | ------------------------------------------------------------------------ |
| Package        | `postcss` (transitive via `next`)                                        |
| Advisory ID    | [GHSA-qx2v-qp2m-jg93](https://github.com/advisories/GHSA-qx2v-qp2m-jg93) |
| Severity       | Moderate                                                                 |
| Affected range | `<8.5.10` (bundled under `node_modules/next/node_modules/postcss`)       |


**Reason for deferral:** Next.js 16 pins an internal PostCSS version. Upgrading
PostCSS independently would require overriding Next's dependency tree or
downgrading Next, neither of which is acceptable for the viewer build.

**Unblock condition:** Next.js releases a patch that bundles PostCSS `>=8.5.10`,
or the project moves to a Next version whose transitive PostCSS satisfies the
advisory. Re-run `npm audit` in `web/` after the next Next.js bump.

## qs (via Cypress)


| Field          | Value                                                                    |
| -------------- | ------------------------------------------------------------------------ |
| Package        | `qs` (transitive via `@cypress/request` → `cypress`)                     |
| Advisory ID    | [GHSA-q8mj-m7cp-5q26](https://github.com/advisories/GHSA-q8mj-m7cp-5q26) |
| Severity       | Moderate                                                                 |
| Affected range | `6.11.1` – `6.15.1`                                                      |


**Reason for deferral:** Cypress 15 locks `@cypress/request`, which depends on the
affected `qs` range. npm audit's suggested fix is a major Cypress downgrade
(13.x), which would break the current E2E and accessibility gate configuration.

**Unblock condition:** Cypress 15 ships a patch that pulls `@cypress/request` with
`qs >=6.15.2`, or the project completes a planned Cypress major upgrade that
resolves the transitive chain. Re-run `npm audit` after the next Cypress bump.