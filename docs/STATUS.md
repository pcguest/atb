# ATB status

Last updated: 2026-04-12

## Release status

- ✅ Public release: `v1.4.0`
- ✅ CI and security gates passing
- ✅ Documentation and release metadata aligned to the current stable baseline

## Current capability summary

The following capability areas match the public README feature set.

- **Encryption and tamper evidence**
  - SHA-256 hash chaining and RFC 8785 canonical JSON
  - Optional client-side bundle encryption (`atb encrypt` / `atb decrypt`)
  - RFC 3161 anchoring: request, token storage, genTime extraction, digest match, cert chain, and signature verification
- **SOC 2 and GDPR exports**
  - `atb export --format soc2`
  - `atb export --format gdpr --type dsr|ropa`
  - Deterministic export outputs for audit repeatability
- **AI integrations**
  - `ATBCallbackHandler` for LangChain (Python)
  - `atbMiddleware` for Vercel AI SDK (TypeScript)
  - Streaming delta events and trace/span linking
- **Local dashboard**
  - `atb view` local-first dashboard flow
  - Tamper-detected blocking state
  - Privacy reveal auditing appended to `bundle.atb`

## Release milestones

- **Milestone 1:** Encryption and parity validation across SDKs
- **Milestone 2:** Event schema evolution with backward compatibility
- **Milestone 3:** Retention, archive ledger, and compliance export foundation
- **Milestone 4:** SOC2/GDPR export specifications and output contracts
- **Milestone 5:** AI integration middleware and streaming trace support
- **Milestone 6:** Local dashboard with graph, inspector, and privacy audit controls

## Quality gates

- Multi-OS CI matrix for Go, Python, and TypeScript
- Golden cross-language canonicalization and parity tests
- Security pipeline: gosec, Bandit, npm audit, plus scheduled/manual Trivy FS/image scans
- Tag-triggered release validation plus separate Docker image publication

## Current focus

- Keep docs, examples, and integration guides synchronized with shipped behaviour
- Preserve the local-first trust and evidence wedge

## Canonical references

- [README](../README.md)
- [Changelog](../CHANGELOG.md)
- [Versioning Policy](../VERSIONING.md)
- [Security Policy](../SECURITY.md)
