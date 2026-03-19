# ATB v1.1.0: Trust Dashboard

**Release Date:** 2026-03-12  
**Tag:** `v1.1.0`  
**Download:** https://github.com/pcguest/atb/releases/tag/v1.1.0

ATB v1.1.0 sharpens ATB into a local-first trust dashboard for teams that need to verify what an AI agent did, keep trace data under their control, and export deterministic evidence for review.

## Highlights

- Local trust dashboard with engineer, auditor, and executive views
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
pip install atb-sdk
atb init
atb view
atb verify
```

## Core Positioning

ATB is for teams that need verifiable audit trails for privacy-sensitive AI agents. It is not positioned as a generic hosted observability platform.

## Links

- Docs: https://github.com/pcguest/atb/tree/main/docs
- Security policy: https://github.com/pcguest/atb/blob/main/SECURITY.md
- Issues: https://github.com/pcguest/atb/issues

If you find a vulnerability, please report it privately to [security@pcguest.dev](mailto:security@pcguest.dev) before public disclosure.
