# ATB Roadmap

ATB is a layered assurance framework for high-impact AI workflows. Its purpose
is to capture, bind, verify, score, and export evidence of what happened, when
it happened, and under what policy and control conditions it happened. The
design is integrity-first: every bundle carries explicit exclusions and known
blind spots. ATB does not claim to capture everything and does not pretend
otherwise.

| Phase | Focus | Status |
|-------|-------|--------|
| v1.10.0 (current) | Hash-chained NDJSON; Ed25519 and ECDSA-P256 signatures; RFC 3161 anchoring; six obligation profiles; WORM export; MCP transport; AES-256-GCM encryption at rest; Go, Python, and TypeScript SDKs | Shipped |
| Q2 2026 | Obligation-profile DSL v1 formalisation; verifier report v1 structured output; privileged-action and data-export profiles end-to-end | In progress |
| Q3 2026 | Remaining profiles; Completeness Assurance Score (CAS) v1 — diagnostic, not authoritative; source signatures for policy gate | Scoped |

The Completeness Assurance Score is a structural score over what a bundle
contains. It is not an audit opinion. A high CAS indicates that the expected
event shape is present and chained correctly; it does not prove model
correctness, actor identity, or that no relevant action occurred outside the
captured session. Obligations in a profile are the hard gate; CAS is the
explanatory signal.
