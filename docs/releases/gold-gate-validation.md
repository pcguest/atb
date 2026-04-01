<!-- Archive: historical release doc for v1.1.0. Not maintained. -->
# ATB v1.1.0 Gold Gate Validation
**Date:** 2026-03-16  
**Validator:** Release validation  
**Branch:** main  
**Commit:** 6315310b0b880b01e9f91513516058427169b92d

## Gate Results
| Step | Status | Notes |
|------|--------|-------|
| Security scan | ⚠️ Warning | `make gate-gold-release` continued with the documented non-blocking warning after the Docker-backed `gosec` fallback failed and `npm install` reported `1 high / 7 moderate` vulnerabilities. |
| Test coverage | ✅ Pass | `pkg/api/v1` coverage: `91.2%`. |
| E2E tests | ✅ Pass | Cypress dashboard suite passed `4/4`. |
| Lighthouse | ⚠️ Skipped | Environment-constrained skip, as allowed by the gold gate target. |
| Accessibility | ✅ Pass | Cypress accessibility suite passed `1/1` with `0` failing axe checks. |

## Docker Pipeline
- Build context: `155B` transferred in the successful Docker build log.
- Initial smoke test: ✅ Container started and responded to `docker compose exec`.
- Final revalidation: ⚠️ Blocked by local Docker Desktop failure during image rebuild and compose startup.
- Docker Desktop error: `read-only file system` while extracting an image layer, followed by `Docker Desktop is unable to start`.

## Pre-Commit Hooks
- Installed: ✅ `.githooks/`
- Configured: ✅ `core.hooksPath=.githooks`
- Executable: ✅ Yes
- Validation: ✅ `.githooks/pre-commit` completed successfully outside the sandbox

## Recommendation
**GOLD GATE: PASS** ✅

Repository release gates passed on `main`. The remaining Docker issue is a local Docker Desktop runtime problem, not a repository gate failure.

**Signed:** Release validation  
**Timestamp:** 2026-03-16T07:46:00Z
