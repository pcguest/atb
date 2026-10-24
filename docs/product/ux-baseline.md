# ATB viewer design rationale

This note records the public design choices for ATB's local evidence viewer.
It is not a record of another product's implementation or deployment.

## User task and evidence boundary

The viewer helps a person inspect one bundle: identify the incident, locate a
finding, follow recorded event order, inspect context and relationships, open
the exact evidence, and assess integrity and evidence coverage. The viewer
does not create an organisational case workflow or infer facts that are absent
from the bundle.

## Navigation

The viewer keeps these jobs distinct:

1. **Incident** — identify the bundle and show bounded verification and
   coverage status.
2. **Findings** — list derived findings with links to their supporting event
   sequence numbers.
3. **Timeline** — inspect recorded order; ordering does not assert causality.
4. **Context** — show recorded context or state that it is unavailable.
5. **Relationships** — show explicit identifiers and references without
   inventing edges.
6. **Evidence** — inspect the original event and its canonical identity.
7. **Trust** — separate integrity, coverage, and external corroboration.

A selected finding or event remains connected to the underlying evidence.
Missing information is shown as unknown or unavailable, not as zero or proof
that nothing happened. Detail is revealed on request; raw event data is not
the default summary.

## Public interoperability

ATB verification and the viewer work without an external service. Optional
custody handoff is described by the
[interoperability contract](../integrations/mortise.md); it does not change
bundle hashing, verification, or ATB's completeness limits.

## Design references

These sources informed general interaction conventions; the decisions above
remain specific to ATB's evidence model.

- Sentry, [Issue Details](https://docs.sentry.io/product/issues/issue-details/):
  findings can lead into detailed evidence.
- Elastic, [Timeline investigation](https://www.elastic.co/docs/solutions/security/investigate/timeline):
  temporal investigation is distinct from a graph of inferred relationships.
- Vanta, [Evidence collection](https://help.vanta.com/en/collections/193126-evidence):
  evidence status benefits from explicit gaps and provenance.

No third-party images, code, or product copy are reproduced here.
