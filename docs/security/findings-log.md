# Security Findings Resolution Log (v1.1.0)

## Finding #1: Reveal Auth Missing
- **Severity:** Critical
- **Resolved In:** Commit `5a13183` (~2026-03-17)
- **Fix Summary:** Added token check via `X-ATB-Viewer-Token` header in `pkg/api/v1/handlers.go:425`
- **Test Added:** `TestRevealHandlerAuth` in `pkg/api/v1/handlers_test.go`
- **Verification:** `curl -X POST /api/v1/privacy/reveal` returns 401 without token

## Finding #2: Audit Chain Sidecar
- **Severity:** Medium
- **Resolved In:** Commit `5a13183` (~2026-03-17)
- **Fix Summary:** Route audit events to `bundle.Append()` for hash-chaining in `handlers.go:478`
- **Verification:** `strings run.atb/bundle.atb | grep privacy_reveal` outputs event JSON

## Finding #3: Rate Limit Threshold
- **Severity:** High
- **Resolved In:** Commit `5a13183` (~2026-03-17)
- **Fix Summary:** Set limiter to 10/min burst in `handlers.go:32`
- **Verification:** 11th request returns 429
