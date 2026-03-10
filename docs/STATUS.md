# ATB Status

Last updated: 2026-03-10

## Release Status

- ✅ Public release: `v1.0.2`
- ✅ CI and security gates passing
- ✅ Documentation aligned to release capabilities

## v1.0.2 Capability Summary

The following capability areas match the public README feature set.

- **Encryption and Tamper Evidence**
  - SHA-256 hash chaining and RFC 8785 canonical JSON
  - Optional client-side bundle encryption (`atb encrypt` / `atb decrypt`)
- **SOC2/GDPR Exports**
  - `atb export --format soc2`
  - `atb export --format gdpr --type dsr|ropa`
  - Deterministic export outputs for audit repeatability
- **AI Integrations**
  - `ATBCallbackHandler` for LangChain (Python)
  - `atbMiddleware` for Vercel AI SDK (TypeScript)
  - Streaming delta events and trace/span linking
- **Local Dashboard**
  - `atb view` local-first dashboard flow
  - Tamper-detected blocking state
  - Privacy reveal auditing with `--log-reveals`

## Release Milestones

- **Milestone 1:** Encryption and parity validation across SDKs
- **Milestone 2:** Event schema evolution with backward compatibility
- **Milestone 3:** Retention, archive ledger, and compliance export foundation
- **Milestone 4:** SOC2/GDPR export specifications and output contracts
- **Milestone 5:** AI integration middleware and streaming trace support
- **Milestone 6:** Local dashboard with graph, inspector, and privacy audit controls

## Quality Gates

- Multi-OS CI matrix for Go, Python, and TypeScript
- Golden cross-language canonicalization and parity tests
- Security pipeline: gosec, Bandit, npm audit, Trivy FS/image scans
- Secret scanning before release publication

## Current Focus

- Maintain release quality for v1.0.x patch line
- Keep docs, examples, and integration guides synchronized with shipped behavior

## Canonical References

- [README](../README.md)
- [Changelog](../CHANGELOG.md)
- [Versioning Policy](../VERSIONING.md)
- [Security Policy](../SECURITY.md)
