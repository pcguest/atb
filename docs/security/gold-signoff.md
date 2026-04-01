<!-- Archive: historical release doc for v1.1.0. Not maintained. -->
# ATB v1.1.0 Gold Release Security Sign-Off
**Date:** 2026-03-19  
**Review:** Security release validation  
**Version:** v1.1.0

## Executive Summary
ATB runtime controls and release security gates passed validation.  
Gold release is **APPROVED FOR GOLD RELEASE** based on the current passing GitHub checks for Go, Node, Python, Trivy image/filesystem, the consolidated security scan, and cross-platform CI.

## Critical/High Findings
- Runtime API/control findings from the prior gate: ✅ resolved.
- Dependency gate findings: ✅ no blocking High/Critical failures in the current release checks.
- Container/image scanning: ✅ pass.

## Medium/Low Findings
- G304 gosec findings in `export.go` / `config.go` remain documented as accepted low-risk follow-up work.
- GitHub Actions SHA pinning remains a hardening item, but is not blocking this release.

## Validation Snapshot
- Go security gate: ✅ Pass
- Node security gate: ✅ Pass
- Python security gate: ✅ Pass
- Trivy Docker image gate: ✅ Pass
- Trivy filesystem gate: ✅ Pass
- Consolidated security scan: ✅ Pass
- Golden test: ✅ Pass
- Cross-platform CI (macOS, Ubuntu, Windows): ✅ Pass

## Compliance Alignment
- SOC2 evidence posture: aligned for release based on passing trace integrity, access control, and release security gates.
- GDPR evidence posture: aligned for release based on PII masking controls and passing security gates.

## Recommendation
**PROCEED WITH GOLD RELEASE.** Maintain the local-first security model and continue hardening follow-up items in the normal release cycle.
