# ATB Release Announcement Template

**Release Date:** YYYY-MM-DD
**Tag:** `vX.Y.Z`
**Download:** https://github.com/pcguest/atb/releases/tag/vX.Y.Z

ATB vX.Y.Z is a local-first bundle runtime for recording tamper-evident workflow evidence. This release [state the concrete improvement in one sentence — what workflow or verification path this release advances].

## Highlights

[Replace each bullet with a concrete, shipped item from this release. Do not carry forward
placeholder bullets. Each bullet should state what changed and what it enables for the operator.
Remove this block if the change is adequately covered by the release notes.]

## Release Quality

Current release validation covers:

- Security gate checks for Go, Node, and Python
- Trivy filesystem and Docker image scanning
- Consolidated security scan
- Golden test validation
- CI on macOS, Ubuntu, and Windows

## Quick Start

```bash
go install github.com/pcguest/atb/cmd/atb@vX.Y.Z
atb bundle new
atb append ai.action.precommit --data='{"action_id":"act-1","action":"escalate"}'
atb verify
```

## Use cases

ATB is suited for:

- incident review without default external trace storage
- customer handoff without platform lock-in
- internal audit and privacy review on a local bundle

The Go CLI is the primary distribution path. The Python and TypeScript packages are SDKs, not the
primary CLI install path.

## Links

- Docs: https://github.com/pcguest/atb/tree/main/docs
- Compliance mappings: https://github.com/pcguest/atb/tree/main/docs/compliance
- Security policy: https://github.com/pcguest/atb/blob/main/SECURITY.md
- Issues: https://github.com/pcguest/atb/issues

If you find a vulnerability, report it privately to [patrickcguest@proton.me](mailto:patrickcguest@proton.me) before public disclosure.
