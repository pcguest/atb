<!-- Archive: historical release doc for v1.1.0. Not maintained. -->
# Known Dependency Vulnerabilities

This file retains rc1 dependency notes for traceability.

## Historical rc1 note

The `v1.1.0-rc1` review tracked advisory exposure on `next@14.2.35` in the web build toolchain.

- That note applied to the rc1 dependency set, not the current checked-in `web/package.json`
- The current release status should be read from `docs/security/gold-signoff.md` and the live GitHub security checks
- Production ATB binaries serve exported static assets through the Go runtime rather than a self-hosted Next.js server
