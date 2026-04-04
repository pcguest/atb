# ATB Release Announcement Template

**Release Date:** YYYY-MM-DD
**Tag:** `vX.Y.Z`
**Download:** https://github.com/pcguest/atb/releases/tag/vX.Y.Z

ATB vX.Y.Z continues the local-first ATB runtime for teams that need to verify what an AI workflow did, keep trace data under their control, and produce a portable review artifact without default external trace storage.

## Highlights

- Local viewer and dashboard preview for private review workflows
- Hash-chained privacy reveal audit logging
- Token-based auth and rate limiting on sensitive reveal paths
- Deterministic SOC 2 and GDPR export workflows
- Client-side encryption support for protected sharing workflows

## Release Quality

Current release validation covers:

- Security gate checks for Go, Node, and Python
- Trivy filesystem and Docker image scanning
- Consolidated security scan
- Golden test validation
- CI on macOS, Ubuntu, and Windows

## Quick Start

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb init
atb append agent.run --data='{"workflow":"support-triage"}'
atb verify
atb view --ui-experimental
```

## Product Positioning

ATB is for teams that need verifiable audit trails for privacy-sensitive AI systems. The Go CLI is the primary distribution path. The Python and TypeScript packages are SDKs, not the primary CLI install path.

Best-fit workflows:

- incident review without default external trace storage
- customer handoff without platform lock-in
- internal audit and privacy review on a local bundle

## Links

- Docs: https://github.com/pcguest/atb/tree/main/docs
- Incident review workflow: https://github.com/pcguest/atb/blob/main/docs/use-cases/incident-review.md
- Internal audit and privacy review: https://github.com/pcguest/atb/blob/main/docs/use-cases/internal-audit-privacy-review.md
- Hosted observability comparison: https://github.com/pcguest/atb/blob/main/docs/comparisons/hosted-observability.md
- Security policy: https://github.com/pcguest/atb/blob/main/SECURITY.md
- Issues: https://github.com/pcguest/atb/issues

If you find a vulnerability, please report it privately to [patrickcguest@proton.me](mailto:patrickcguest@proton.me) before public disclosure.
