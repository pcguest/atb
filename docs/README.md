# ATB documentation

ATB has one front door, this hub, and a small set of guided entry points.
Start with the [five-minute quickstart](./quickstart.md), then choose the path
that matches your role. Deeper pages are linked from these guides rather than
listed as an undifferentiated catalogue.

## 1. Use ATB today

- [Five-minute quickstart](./quickstart.md) — install the CLI, create a bundle,
  and verify it offline.
- [Developer capture guide](./guides/capture-quickstart.md) — instrument an AI
  or agent workflow without overstating capture coverage.
- [Auditor acceptance guide](./ciso-acceptance-guide.md) — review integrity,
  profile obligations, CAS, and residual risk.
- [Operator WORM guide](./integrations/worm-s3.md) — retain bundles in
  operator-controlled immutable storage.

## 2. Understand the contract

- [Bundle specification v1.0](./spec-v1.0.md) — append-only NDJSON, RFC 8785
  canonicalisation, and the SHA-256 chain.
- [Security model](./security.md) — guarantees, threats, and explicit limits.
- [Profiles and CAS](./profiles.md) — workflow obligations and evidence
  completeness signals.

## 3. Integrate

- [SDK index](../sdk/README.md) — Go, Python, and TypeScript integration
  surfaces with shared golden vectors.
- [AI integration guide](./ai-integration.md) — capture adapters, imports, and
  framework integration choices.
- [Custos companion product](https://github.com/pcguest/custos-product) —
  separate proprietary custody, receipts, transparency log, and auditor access.

## 4. Compliance mapping

- [Compliance mapping hub](./compliance/README.md) — technical evidence
  mappings only; not legal advice or certification.
- [Provability ladder](./provability-ladder.md) — what each evidence layer does
  and does not prove.

## 5. Maintainer

- [Contributing](../CONTRIBUTING.md) — development workflow and gates.
- [Maintenance index](./maintenance/README.md) — current operations,
  architecture guardrails, and archived programme records.
- [Submission / evaluation](https://github.com/pcguest/custos-product/blob/main/docs/SUBMISSION.md)
  — public evaluator path, release references, never-claims, and exclusions.
