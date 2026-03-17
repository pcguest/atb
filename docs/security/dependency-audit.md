# Dependency Security Audit (v1.1.0-rc1)

## Go Dependencies
- Total: 6 modules (`go list -m all`)
- Vulnerabilities: 0 reachable vulnerabilities (`govulncheck ./...`)
- Outdated: 5 modules with updates available (`go list -m -u -json all`)
- Remediation: `go.mod` moved to `go 1.26.1` with `toolchain go1.26.1`

## NPM Dependencies
- Total: 984 dependencies (`npm audit --json`)
- High/Critical vulns: 1 (High: 1, Critical: 0)
- Remaining package: `next@14.2.35`
- Status: documented in [known-vulns.md](/Users/paddyguest/atb/docs/security/known-vulns.md) because the shipped product embeds static export output (`web/out`) instead of running a self-hosted Next.js server
- Remediation completed:
  - `next` bumped from `14.2.21` to `14.2.35`
  - `eslint-config-next` bumped from `14.2.5` to `14.2.35`
  - `glob` overridden to `10.5.0`
  - `minimatch` overridden to `9.0.7`

## GitHub Actions
- Workflow `uses:` entries scanned: 57
- SHA-pinned actions: 0
- Version-pinned actions: 57
- No branch pins (`main`/`master`/`latest`): ✅
- No tag-based pins: ❌ (all current pins are version tags, not immutable commit SHAs)

## Recommendation
**NEEDS REMEDIATION**  
Do not issue Gold security approval until:
1. Remaining GitHub Actions are pinned to immutable SHAs (or exception accepted in writing).
2. Remaining documented `next` advisory is either accepted for static-export usage or cleared by a future framework upgrade.
