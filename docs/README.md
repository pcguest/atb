# ATB documentation

ATB is local-first evidence infrastructure for AI-agent incidents and audit.
It produces portable, independently verifiable bundles. It proves the
integrity of recorded evidence, not that capture was complete or the recorded
claims were true.

## Start and capture

- [Five-minute quickstart](./getting-started/quickstart.md)
- [Configuration](./getting-started/configuration.md)
- [Capture paths and boundaries](./capture/overview.md)

## Understand the evidence

- [Architecture](./concepts/architecture.md)
- [Evidence model](./concepts/evidence-model.md)
- [Trust and security model](./concepts/trust-model.md)
- [Glossary](./concepts/glossary.md)
- [Profiles](./evidence/profiles.md) and [CAS](./evidence/cas.md)
- [Key management](./evidence/key-management.md)

## Investigate and verify

- [Offline verification](./investigate/verify.md)
- [Incident reconstruction](./investigate/incidents.md)
- [Tamper demonstration](./investigate/tampering.md)

## Integrate

- [SDK index](../sdk/README.md)
- [Integration index](./integrations/README.md)
- [Automation contract](./maintainers/automation-contract.md)
- [Mortise boundary](./integrations/mortise.md)

## Public contracts

- [Bundle format v1](./specification/bundle-v1.md)
- [Event semantics](./specification/events.md)
- [Profile DSL v1](./specification/profile-dsl-v1.md)
- [Verifier report](./specification/verify-report.md)
- [Local viewer](./specification/viewer.md)
- [Bundle push](./specification/bundle-push.md)

## Compliance evidence

- [Compliance evidence overview](./compliance/README.md)
- [EU AI Act evidence mapping](./compliance/article-12-mapping.md)

These are technical mappings, not legal advice, certification, or conformity
assessment.

## Maintain

- [Contributing](../CONTRIBUTING.md)
- [Versioning](../VERSIONING.md)
- [Release runbook](./maintainers/release.md)
- [Support matrix](./maintainers/support-matrix.md)
- [Performance](./maintainers/performance.md)
- [Lint suppressions](./maintainers/lint-suppressions.md)
- [Security scanner suppressions](./maintainers/security-suppressions.md)
- [Roadmap](./roadmap.md)

Mortise is an optional external custody and organisational layer. ATB remains
fully useful without it.
