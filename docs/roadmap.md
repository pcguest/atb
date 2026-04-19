# ATB roadmap

This document summarises planned work that follows the current v1.7.x
baseline. It distinguishes shipped capabilities from forward-looking
items and should be read as sequencing guidance rather than a delivery
commitment.

## Current baseline

The current line provides:

- Local-first SHA-256 hash chaining over RFC 8785 canonical JSON.
- Six schema-locked obligation profiles with CAS support.
- Profile DSL v1 for additional profile definitions.
- RFC 3161 anchoring, bundle signing, WORM export, local viewer, and
  MCP stdio support.

ATB proves the integrity of what was recorded. It does not prove
recording completeness, actor identity, workflow correctness, or a
compliance verdict.

## Short term

### Profile DSL v1 hardening

Status: In progress across v1.7.x and v1.8.x

Description: tighten validation, loading, and reporting around built-in
and file-defined profiles so misconfiguration is caught early and every
surface consumes the same profile metadata.

Intended benefit: fewer silent profile errors, more predictable
verification output, and simpler maintenance of custom workflow
definitions.

Sequencing: current work through early v1.8.x.

### Verifier and report v1 consolidation

Status: Largely complete

Description: keep bundle evaluation, CAS normalisation, and report
shaping on one internal path shared by the CLI, viewer, dashboard, MCP
bridge, and tests.

Intended benefit: one source of truth for integrity checks, obligation
results, residual risk, and profile stamping.

Sequencing: current work through early v1.8.x.

## Medium term

### CAS v1 refinements

Status: Planned

Description: refine sub-score guidance, strengthen normalisation rules,
and make profile-specific weighting clearer where recorded evidence is
stronger or weaker than the current defaults suggest.

Intended benefit: more consistent completeness scoring across different
workflow classes without weakening the existing integrity-first model.

Sequencing: after v1.8.x.

### Source signatures for policy gates

Status: Planned

Description: extend source-signature handling around policy gates and
authorisation points so recorded decisions can be corroborated by signed
inputs from the controlling system.

Intended benefit: stronger evidence that a recorded allow or deny event
originated from the expected gate, not only that it was recorded
unchanged afterwards.

Sequencing: after v1.8.x, alongside CAS refinements.

## Longer term

### Corroboration adapters

Status: Planned

Description: add adapters that can pull or compare corroborating
evidence from external systems such as gateways, ticketing layers, or
execution receipts.

Intended benefit: better support for workflows where the strongest
evidence spans more than one system boundary.

Sequencing: after the v1.8.x line.

### Queue, gateway, and storage integration

Status: Planned

Description: add focused integration points for queue consumers,
workflow gateways, and storage systems so key hand-off events can be
captured or checked more directly.

Intended benefit: stronger evidence for queued or asynchronous systems
where the local bundle should be paired with queue dequeue, gateway, or
storage-side facts.

Sequencing: after corroboration adapters begin to land.

### Reconciliation against underlying systems

Status: Planned

Description: provide workflows that compare a recorded bundle with the
state of the underlying system of record and report mismatches.

Intended benefit: clearer detection of gaps between what ATB recorded
and what the downstream system says actually happened.

Sequencing: after queue, gateway, and storage integration.

### Exportable assurance packs

Status: Planned

Description: package verification output, supporting artefacts, and
selected corroboration evidence into portable review bundles for handoff
to customers, auditors, or incident reviewers.

Intended benefit: simpler external review without overstating what ATB
guarantees on its own.

Sequencing: after reconciliation and corroboration foundations are in
place.

## Tracking

For shipped changes and release-by-release detail, see
[CHANGELOG.md](../CHANGELOG.md).
