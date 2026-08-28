# Roadmap

ATB's stable product is local capture, tamper-evident bundle construction,
offline verification, incident reconstruction, obligation profiles, CAS, and
portable review/export. This roadmap records direction, not release promises.

## Now: product hardening

- Keep the incident-forensics workflow deterministic and runnable from a fresh
  checkout.
- Maintain byte-identical Go, Python, and TypeScript hashing behaviour.
- Bound untrusted inputs and preserve explicit privacy defaults.
- Keep release artefacts reproducible, attributable, and independently
  verifiable.

## Next: evidence interoperability

- Broaden OpenTelemetry GenAI mappings without weakening the canonical event
  model.
- Add database and independent-system reconciliation evidence.
- Refine CAS and provability guidance from real incident reviews.

## Later: optional integrations

- Additional framework adapters when they can remain thin and testable.
- Further independently operated witness or custody interoperability through
  stable public contracts.

## Not planned for ATB core

- Hosted tracing, telemetry collection, or multi-tenant review.
- Organisational SSO, billing, legal hold, or custodian-of-record operations.
- Real-time universal prevention of AI actions.
- Claims of capture completeness, model correctness, actor identity, or
  regulatory certification.
- Training-data governance.

Those organisational and hosted concerns belong in Mortise or another external
system. Each ATB integration remains bounded by the calls and records that
actually cross it.
