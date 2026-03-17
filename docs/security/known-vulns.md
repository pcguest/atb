# Known Dependency Vulnerabilities (v1.1.0-rc1)

## NPM: next (14.2.35)
- Vuln IDs: `GHSA-h25m-26qc-wcjf`, `GHSA-9g9p-9gw9-jx7f`
- Severity: High / Moderate
- Path: `atb-web > next`
- Status: No non-breaking fix available in current major as of 2026-03-16. `npm audit` only offers `next@16.1.6` as a semver-major upgrade.
- Risk Assessment:
  - Used in: build/export tooling, not the shipped ATB runtime
  - Attack surface: local/dev or CI if `next dev` is run directly; not exposed in the production Go-served UI path
  - Exploitability: low for released ATB binaries, because the product ships static assets from `web/out` embedded via `uiembed.go`
- Mitigation:
  - `web/next.config.js` sets `output: "export"` and `images.unoptimized: true`
  - Production serving path is the Go binary, not a self-hosted Next.js server
  - Keep `next dev` bound to localhost for developer use
- Plan: reassess major upgrade to Next 16 in `v1.1.1` or `v1.2.0` after compatibility validation
