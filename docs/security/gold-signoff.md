# ATB v1.1.0 Gold Release Security Sign-Off
**Date:** 2026-03-16  
**Auditor:** Security Agent  
**Version:** v1.1.0

## Executive Summary
ATB runtime controls (auth, rate limiting, audit chaining, CSP, and PII masking) passed validation.  
Gold release is **NOT APPROVED** at this time due to unresolved dependency critical/high findings and missing `v1.1.0-rc1` tag traceability.

## Critical/High Findings
- Runtime API/control findings from prior gate: ✅ resolved.
- Dependency scan findings: ❌ unresolved.
  - Go: 5 reachable standard-library vulnerabilities (fixed in Go 1.26.1).
  - NPM: 1 critical + 6 high vulnerabilities (`npm audit`).

## Medium/Low Findings
- G304 gosec findings in `export.go` / `config.go` (accepted pre-existing risk).
- Trivy local Docker fallback requirement (documented tooling constraint).
- GitHub Actions currently use version tags rather than immutable SHA pins.

## Test Coverage
- Auth/Rate-limit/Audit: ✅ Pass
- CSP/Headers: ✅ Pass
- PII Masking: ✅ Pass
- Dependency Scan: ❌ Fail (Go and NPM high/critical findings present)
- E2E Security Tests: Not executed in this audit pass

## Compliance Alignment
- SOC2 controls: Partially aligned (audit trail and access controls verified; dependency risk remains open).
- GDPR controls: Partially aligned (PII redaction verified; release blocked on dependency risk posture).

## Recommendation
**DO NOT PROCEED WITH GOLD RELEASE** until dependency critical/high findings are remediated and `v1.1.0-rc1` tag traceability is restored.
