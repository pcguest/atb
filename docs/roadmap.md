# ATB roadmap

This document summarises shipped and planned work from the v1.8.x/v1.9.x
baseline. It distinguishes shipped capabilities from forward-looking
items and should be read as sequencing guidance rather than a delivery
commitment. A roadmap entry means the work is tracked; it does not mean
it is committed to a specific release.

## Summary

ATB today gives you a record of an AI workflow that you can verify later — you know whether
the sequence of events was altered after the fact, without needing a server or a third-party
service. The next two quarters are focused on corroboration adapters and queue and storage
gateway integrations, which extend the verifiable record to cover evidence held in external
systems alongside the local bundle. Together those two pieces close the main evidence gap in
asynchronous workflows where the triggering event and the recorded event happen in different
systems. The longer-term objective is a published, independently-implementable specification:
a precise enough description of how ATB constructs and verifies a bundle that a second
implementation, written without access to this codebase, would produce bundles that verify
correctly against it.

ATB remains a narrow-scope product. Planned work should strengthen the integrity and evidence
model around that scope rather than broaden ATB into a general AI operations platform.

## Prioritisation notes

Current roadmap sequencing is driven by the local-first evidence model.
The immediate focus is hardening the local bundle, verifier, and export
path before adding broader external integrations.

Corroboration adapters (`#47`) and queue, gateway, and storage
integration (`#48`) are next because they address the main evidence gap
in asynchronous workflows: the hand-off between the local bundle and the
external systems where work is actually dequeued, routed, or persisted.
Exportable assurance packs (`#49`) follow because they reduce friction
when handing bundles to auditors, customers, or incident reviewers.

Roadmap items are tracked work, not committed delivery in a specific
release. This document is updated with each minor release so evaluators
can see current sequencing rather than stale intent.

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

## Shipped

### Profile DSL v1 hardening

Shipped: v1.8.0

Tightened validation, loading, and reporting around built-in and
file-defined profiles. Misconfiguration is caught early; every surface
consumes the same profile metadata. `atb profiles validate` validates
all built-in profiles and any additional profiles supplied via `--file`
or `--dir`, checking required fields, duplicate IDs, and CAS
weight-vector sums.

### Verifier and report v1 consolidation

Shipped: v1.8.0

`EvaluateBundle` in `internal/verify/evaluate.go` centralises bundle
loading, hash-chain integrity, RFC 3161 anchor verification, CAS
normalisation, profile stamping, residual risk, and post-profile
transformations. The CLI, viewer, dashboard, MCP bridge, and tests all
derive reports from this function.

### CAS v1 corroboration bonus

Shipped: v1.9.0

`CorroborationPolicy` struct (`AnchorBonus` 0.05, `SignatureBonus` 0.03,
`SnapshotBonus` 0.02, `MaxBonus` 0.10) with `Validate()` and
`DefaultCorroborationPolicy()`. `EvaluateBundle` accepts
`WithCorroborationPolicy`; when set, `CASResult` gains
`corroboration_bonus` and `effective_score` fields and grade derives from
`effective_score`. Nil policy produces output identical to v1.8.0.
`atb verify` applies the default policy automatically when `--with-anchor`
is present; `--corroboration-policy <path>` overrides the defaults.

### Source signatures for policy gates

Shipped: v1.9.0

`--policy-doc <path>` flag on `atb append` (`ai.policy.decision` events
only): reads the file, computes `SHA-256(contents)` hex, and embeds it as
`policy_doc_hash`. When `--sign-policy` is also set, stores a compound
Ed25519 `policy_doc_signature` over `SHA-256(canonical payload) ||
SHA-256(doc bytes)`. `VerifyPolicyDocSignature` in `internal/sign`
verifies the signature. `policy_doc_signature_valid` is surfaced in
`TrustReport`: nil when no `policy_doc_hash` is present, true or false
otherwise.

## Longer term

### Corroboration adapters

Status: Planned

Target: Q4 2026

Description: add adapters that can pull or compare corroborating
evidence from external systems such as gateways, ticketing layers, or
execution receipts.

Intended benefit: better support for workflows where the strongest
evidence spans more than one system boundary.

Sequencing: after the v1.9.x line. This is a planning target, not a promise.

### Queue, gateway, and storage integration

Status: Planned

Target: Q4 2026

Description: add focused integration points for queue consumers,
workflow gateways, and storage systems so key hand-off events can be
captured or checked more directly.

Intended benefit: stronger evidence for queued or asynchronous systems
where the local bundle should be paired with queue dequeue, gateway, or
storage-side facts.

Sequencing: after corroboration adapters begin to land. This is a planning target, not a promise.

### Reconciliation against underlying systems

Status: Planned

Target: Q1 2027

Description: provide workflows that compare a recorded bundle with the
state of the underlying system of record and report mismatches.

Intended benefit: clearer detection of gaps between what ATB recorded
and what the downstream system says actually happened.

Sequencing: after queue, gateway, and storage integration. This is a planning target, not a promise.

### Exportable assurance packs

Status: Planned

Target: Q1 2027

Description: package verification output, supporting artefacts, and
selected corroboration evidence into portable review bundles for handoff
to customers, auditors, or incident reviewers.

Intended benefit: simpler external review without overstating what ATB
guarantees on its own.

Sequencing: after reconciliation and corroboration foundations are in
place.

## Issue handling

- Use issues to track concrete work, gaps, or questions.
- Keep issue language specific about what is missing today.
- Avoid phrasing an issue as an implied promise of delivery unless a release decision has already been made.
- When work ships, move the user-facing statement to `CHANGELOG.md` and keep this roadmap focused on direction.

## Long-term objective

The end goal for ATB is a published specification — a sufficiently precise
and self-contained description of tamper-evident AI audit trail construction
that can be implemented independently, reference-audited, and cited by
compliance teams without relying on this repository as the sole authority.

The current implementation is the reference implementation of that
specification. The `docs/compliance/` folder is the evolving body of work
that maps the specification to regulatory frameworks: EU AI Act Article 12,
NIST AI RMF, GDPR Article 22, and SOC2. None of those mappings claim
ATB satisfies a framework on its own — they record what ATB contributes and
what operators must still provide.

Getting to a publishable specification means the format and verification
semantics need to be stable enough that a second implementation, written
without access to this codebase, would produce bundles that verify
correctly against this one. The hash-chaining scheme and canonical JSON
encoding (`SHA-256(prev_hash || RFC8785(event_json))`) are already at
that level. CAS normalisation and obligation profiles are not yet stable
enough to specify independently. That is the gap the v1.9.x and later
work is closing.

## Tracking

For shipped changes and release-by-release detail, see
[CHANGELOG.md](../CHANGELOG.md).
