# ATB Roadmap

ATB is a layered assurance framework for high-impact AI workflows. Its purpose
is to capture, bind, verify, score, and export evidence of what happened, when
it happened, and under what policy and control conditions it happened. The
design is integrity-first: every bundle carries explicit exclusions and known
blind spots. ATB does not claim to capture everything and does not pretend
otherwise.

| Phase | Focus | Status |
|-------|-------|--------|
| v1.12.0 (current) | Hash-chained NDJSON; Ed25519 and ECDSA-P256 signatures; RFC 3161 anchoring; six obligation profiles; Capture v1 (`capture run`, `import chatlog`); corroboration events; WORM export and queue push; MCP transport; AES-256-GCM encryption; `verify.report.v1` custody contract; Agent capture/workspace APIs; profile workflow helpers across Go, Python, and TypeScript SDKs | Shipped |
| Q2 2026 | Obligation-profile DSL v1 formalisation; verifier report v1 structured output; privileged-action and data-export profiles end-to-end | Shipped |
| Q3 2026 | Temporal ordering gates; CAS v1 formalisation; source signatures for policy gate; provability ladder in verifier output | Scoped |
| Custos (separate product) | Hosted custodian-of-record: ingest, WORM receipts, auditor portal, retention | Planned — see [custos-handoff.md](./custos-handoff.md) |

The Completeness Assurance Score is a structural score over what a bundle
contains. It is not an audit opinion. A high CAS indicates that the expected
event shape is present and chained correctly; it does not prove model
correctness, actor identity, or that no relevant action occurred outside the
captured session. Obligations in a profile are the hard gate; CAS is the
explanatory signal.

For the provability model and how to shrink recorded blind spots, see
[provability-ladder.md](./provability-ladder.md).
