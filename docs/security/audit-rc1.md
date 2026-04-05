<!-- Archived release-planning draft. Not maintained. -->
# ATB v0.9.0-beta Security Audit
**Date:** 2026-03-16  
**Auditor:** Security review  
**Requested Tag:** v0.9.0-beta  
**Audited Revision:** main @ `6315310b0b880b01e9f91513516058427169b92d` (tag not found locally or on `origin`)

## Test Results
| Test | Status | Notes |
|------|--------|-------|
| Auth on `/privacy/reveal` | ✅ Pass | Returns 401 without token |
| Rate limiting (10/min) | ✅ Pass | Requests 1-10 return 200; request 11+ returns 429 |
| Audit chain to `bundle.atb` | ✅ Pass | `privacy_reveal` events present in `run.atb/bundle.atb` |
| CSP headers | ✅ Pass | `Content-Security-Policy` header present on `/view/` |
| PII masking | ✅ Pass | API responses include `[REDACTED]` markers for sensitive fields (`id`, `ip`); current dataset did not include `email` samples |

## Remaining Issues
- Release traceability gap: the expected beta tag is missing in this repository snapshot.
- G304 gosec findings in `export.go` / `config.go` (pre-existing; previously documented).
- Trivy local execution still depends on local binary or Docker fallback.

## Recommendation
Runtime security controls are working as expected. This rc1 review was an intermediate checkpoint before final release validation.
