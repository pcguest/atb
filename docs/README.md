# ATB documentation

Start with the [five-minute quickstart](./quickstart.md), then follow the path
that matches your role. This hub is the only doc map — deeper pages are linked
from here, not scattered across the tree.

ATB is the open-source, local-first evidence core for independently verifiable
records of AI-agent behaviour. It proves the integrity of recorded evidence,
not the completeness or truth of capture. No hosted service, account, or
external database is required.

## Use ATB

| Doc | For |
| --- | --- |
| [Quickstart](./quickstart.md) | Install, create a bundle, verify offline |
| [Capture guide](./guides/capture.md) | Import, wrapper, intercept, integrations |
| [Incident forensics](./guides/incident-forensics.md) | Capture → discover → review after an agent incident |
| [Tamper demo](./guides/tamper-demo.md) | One-byte mutation and verify failure |
| [WORM / S3](./integrations/worm-s3.md) | Request operator-controlled retention and verify the external boundary |
| [Mortise handoff](./mortise-handoff.md) | Optional commercial custody integration boundary |

## Review evidence

| Doc | For |
| --- | --- |
| [Auditor acceptance guide](./ciso-acceptance-guide.md) | Integrity, profiles, CAS, residual risk |
| [Security model](./security.md) | Threats, guarantees, explicit limits |
| [Public surface](./public-surface.md) | Shipped boundary and never-claims |
| [Provability ladder](./provability-ladder.md) | Integrity vs completeness |
| [CAS guide](./cas-guide.md) | Profile-scoped evidence coverage detail |
| [Compliance hub](./compliance/README.md) | Generate a deterministic mapping-oriented evidence pack |
| [Glossary](./glossary.md) | Canonical ATB, Tenon, Mortise, bundle, and evidence terms |

## Contract

| Doc | For |
| --- | --- |
| [Bundle spec v1.0](./spec-v1.0.md) | NDJSON format, hashing, canonicalisation |
| [AI traces spec](./spec-ai-traces.md) | Event types and workflow semantics |
| [Profiles](./profiles.md) | Obligation profiles and evidence rules |
| [Verify JSON schema](./api/verify-schema.md) | `VerifierReport` contract |
| [Support matrix](./support-matrix.md) | Supported Go, Python, Node, OS versions |

## Integrate

| Doc | For |
| --- | --- |
| [SDK index](../sdk/README.md) | Go, Python, TypeScript |
| [AI integration](./ai-integration.md) | CLI JSON contract, exit codes, CI patterns |
| [Integrations index](./integrations/README.md) | LangChain, MCP, Vercel AI, SIEM export |

## Compliance mapping

Technical evidence mappings only — not legal advice or certification.

| Doc | For |
| --- | --- |
| [Compliance hub](./compliance/README.md) | How to use mappings in a review |
| [Article 12 evidence mapping](./compliance/article-12-mapping.md) | Per-obligation map of Article 12 to ATB primitives |
| [EU AI Act mapping](./compliance/eu-ai-act.md) | Article 9 to 20 coverage map |

## Contribute and release

| Doc | For |
| --- | --- |
| [Contributing](../CONTRIBUTING.md) | Development workflow and maintainer rules |
| [Versioning](../VERSIONING.md) | SemVer, manifest version, breaking changes |
| [Release runbook](./release.md) | Tag, publish, and pipeline gates |
| [Roadmap](./roadmap.md) | Shipped vs planned ATB work |
| [Disaster recovery](./maintenance/disaster-recovery.md) | Source, secrets, and release recovery |
| [Manual test playbook](./maintenance/manual-test-playbook.md) | Historical private-trunk ATB/Mortise test procedure |
| [Public MIT extraction checklist](./maintenance/public-mit-extraction-checklist.md) | Manual allowlist, exclusion, and validation steps for public ATB extraction |
| [History rewrite plan](./maintenance/history-rewrite-plan.md) | Provenance-preserving history rewrite procedure and decision record |

Tenon is the umbrella product identity. Mortise is the optional commercial
custody and organisational layer for ATB evidence; it is not required for any
core workflow. See [Mortise handoff](./mortise-handoff.md).
