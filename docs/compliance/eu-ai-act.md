# EU AI Act — ATB Coverage Map

Regulation (EU) 2024/1689 applies broadly from 2 August 2026, subject to phased and sector-specific provisions. ATB targets runtime evidence for high-risk AI system obligations under Title III by recording local audit events into verifiable bundles.

## Coverage summary

| Article | Requirement | ATB coverage | Status |
| --- | --- | --- | --- |
| Article 9 | Risk management system | Obligation profiles + CAS scoring | Partial |
| Article 12 | Automatic logging | Hash-chained bundles, Ed25519 signing, retention guard | Covered |
| Article 13 | Transparency to deployers | verify.report.v1, CAS residual-risk output | Partial |
| Article 14 | Human oversight | human_override + policy_decision profiles | Partial |
| Article 17 | Quality management documentation | docs/spec/, obligation profile specs | Partial |
| Article 20 | Automatically generated logs | capture run, import chatlog, OTel inbound (Phase 9) | Partial |
| Article 10 | Training data governance | Not applicable — runtime capture only | Out of scope |
| Article 11 | Technical documentation | docs/spec/ covers runtime; training docs not in scope | Out of scope |

## What ATB guarantees

- Every event appended to a bundle can be verified to be unmodified since capture, using only the `atb` CLI and the bundle file.
- Each record is linked to the previous record by the documented SHA-256 hash chain over RFC 8785 canonical JSON.
- Signed bundles can be checked against recorded Ed25519 signatures.
- The retention guard blocks configuration below the EU AI Act minimum unless the operator explicitly allows it.
- `verify.report.v1` records verifier results, CAS output, and profile gaps in a machine-readable custody report.
- Local-first operation avoids a mandatory hosted logging service or vendor custody path.

## Known gaps (honest)

- Bypass via direct provider API calls remains possible; no subprocess interception exists in ATB core, and the hard-boundary project handles that separately.
- Reviewer identity is not yet anchored (Article 14 gap — planned Phase 10).
- No automated evidence pack export exists for Articles 17–20 (planned Q4 2026).
- Completeness is structural (CAS), not proof of universal capture.

## Roadmap to full coverage

The Q3 2026 roadmap focuses on Phase 9 corroboration adapter wiring, OTel bundle event mapping, GitHub audit log corroboration, LangChain/LangGraph corroboration, and automatic capture callbacks. These milestones reduce Article 12 capture gaps and add external corroboration signals for recorded workflows. The Q4 2026 to Q1 2027 roadmap covers reviewer identity anchoring, retention enforcement access logging, automated evidence pack export, and CAS v1 formalisation. See [docs/roadmap.md](../roadmap.md) for the current milestone list and explicit out-of-scope boundaries.
