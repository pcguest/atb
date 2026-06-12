# ATB maintainer agent handoff

> Verifiable AI incident evidence — capture with ATB, custody with Custos, review without exposing prompts.

Public state verified 2026-06-13. ATB is **product finalized** at tagged release
`v1.14.3`; `main` may carry post-tag fixes queued for `v1.14.4`.

## Current versions

| Component | Version | Notes |
| --- | --- | --- |
| ATB (tagged release) | [`v1.14.3`](https://github.com/pcguest/atb/releases/tag/v1.14.3) | `go install github.com/pcguest/atb/cmd/atb@v1.14.3` |
| ATB (`main`, post-tag) | unreleased | Intercept shutdown, profile inference, doc honesty — see CHANGELOG `[Unreleased]` |
| Custos | [`v0.5.0`](https://github.com/pcguest/custos-product/releases/tag/v0.5.0) | `go.mod` pins ATB `v1.14.3`; evaluator docs updated to match |

## What “finalized” means

Shipped: hash-chained capture, offline verify, intercept proxy, incident
reporting, evidence packs, SDK parity, public MIT licence.

Open **operational** items (not product scope): cut `v1.14.4` when post-tag fixes
merge; re-run `Release` workflow on the next green tag so npm/PyPI advance;
Custos licence evaluation grant (P0, Custos repo only).

## Release pipeline status

| Gate | Status |
| --- | --- |
| `version-gate` on `main` | Green |
| `Gold Release Gate` on `main` | Green (workflow_dispatch after coverage tests) |
| `Gold Release Gate` on tag `v1.14.3` | Failed historically (77% coverage); fixed on `main` ≈91% |
| `Docker Publish` on tag `v1.14.3` | Green (operator PAT rotation succeeded) |
| `Release` (npm/PyPI) on tag `v1.14.3` | Failed (blocked by gold gate at tag time); registries lag until next green tag |

See [`docs/release.md`](../release.md) for the runbook.

## Practitioner review resolution (P0/P1)

| Finding | Resolution |
| --- | --- |
| **P0** Custos no public evaluation licence | **Custos scope** — accepted limit; ATB docs never claim Custos procurement readiness |
| **P1** Intercept shutdown skips session close | **Fixed on `main`** — single shutdown path in `internal/proxy` |
| **P1** Receipt signature field coverage | **Custos scope** — document in Custos capability boundary |
| **P1** Infer `background_automation` for proxy traffic | **Fixed on `main`** — infer only when `ai.job.*` present |
| **P1** Custos docs pin ATB v1.14.2 | **Fixed in custos-product** — SUBMISSION/HANDOFF/e2e aligned to v1.14.3 |
| **P1** README “no trust in operator” | **Fixed** — integrity vs capture completeness stated |
| **P1** CAS Medium/Low audit claims | **Fixed** — `docs/ciso-acceptance-guide.md` softened |

**P2 deferred:** `critical_failures` on integrity failure; signed fixture in guide (unsigned example now documented).

## Orthodox operator setup

- Run gate scripts with **`/bin/bash scripts/*.sh`** (not env `bash` on operator Mac).
- Go work: `GOCACHE=$(pwd)/.gocache GOTOOLCHAIN=go1.26.4`.
- Intercept capture: `HTTPS_PROXY`, local CA trust — see [`docs/guides/agent-incident-forensics.md`](../guides/agent-incident-forensics.md).

## Out of scope for next agent

Ring 4 hosted product, `atb.human.*` schema merge, demo videos, hard-boundary
enforcement code, compliance certification language, re-tagging or moving
existing release tags.

## Starter commands

```bash
cd /path/to/atb
GOCACHE=$(pwd)/.gocache GOTOOLCHAIN=go1.26.4 /bin/bash -c 'make test-golden'
GOCACHE=$(pwd)/.gocache GOTOOLCHAIN=go1.26.4 go test ./... -count=1
/bin/bash scripts/check-versions.sh && /bin/bash scripts/check-support-matrix.sh
gh run list -R pcguest/atb -b main -L 5
```
