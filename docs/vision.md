# Vision — the end state for ATB and Custos

This document states where ATB and Custos are going and why, grounded in the
market research that chose the wedge. It is a north star, not a release plan; the
[roadmap](./roadmap.md) tracks what is actually shipped. It deliberately
separates what is built today from where the product is headed.

## The market, in one paragraph

Through 2025–2026 organisations moved agents from demos into production, and the
incidents followed: agents deleting data, fabricating logs to cover their tracks,
running privileged tools without approval, and failing silently. Fewer than one
in five organisations have any AI-specific incident tooling, while the EU AI Act
(Article 12 logging, Article 26(6) six-month retention, Annex III from August
2026) is turning "keep tamper-evident records of what the AI did" from good
practice into a legal obligation. The existing tools answer the wrong question:
observability platforms (LangSmith, Langfuse) show you what happened but their
records are mutable; governance platforms (Vanta, Credo) produce documents about
your process, not cryptographic evidence of each execution. The closest analogue
is software supply-chain provenance — Sigstore, Rekor, in-toto, SLSA — but those
attest to artifacts and builds, not AI runtime behaviour.

**The wedge: agent incident forensics.** A local-first flight recorder for AI
agents that you can trust *even when the agent's own logs cannot be trusted*,
plus a custody layer that holds that evidence and lets a third party verify it
independently.

## The two products and the boundary between them

ATB and Custos are deliberately separate, with a hard boundary (see `AGENTS.md`).
The boundary is the product strategy, not an accident of packaging.

**ATB — the recorder and verifier. Local-first, open source, no account.**
ATB records what an agent did into append-only, hash-chained, optionally signed
`.atb` bundles and verifies them later. It runs at the edge — in the agent
process, behind a capture proxy, or in CI — with no network dependency. Its
claim is narrow and strong: *the recorded sequence was not altered since
capture*. It does not claim completeness, identity, or compliance by itself, and
it never becomes a hosted platform. ATB is the thing a developer can adopt in an
afternoon without asking anyone's permission.

**Custos — the custody and attestation layer. The independent third party.**
Custos ingests ATB bundles, holds them in write-once storage, and issues signed
receipts attesting *that it received this exact evidence at this time*. It is the
"Rekor for AI execution evidence": the neutral party whose receipt means
something precisely because it is not the agent, not the developer, and not the
recorder. Custos is reference infrastructure in this repo; the hosted,
multi-tenant operation of it (auditor portals, billing, SSO/RBAC, legal hold,
custodian-of-record) lives outside the ATB runtime.

The value compounds only when both exist: ATB makes evidence that is hard to
forge; Custos makes that evidence hard to suppress or backdate. Integrity (ATB)
plus custody (Custos) is the full chain a regulator, auditor, or court needs.

## End state — what "done" looks like

### ATB (the recorder) should be

- **Capture you can trust the boundary of.** Every bundle states what the
  recorder could and could not see (`atb.capture.scope`), so a reviewer knows the
  evidence's edges. Capture spans the in-process SDK, the `atb intercept` proxy,
  and CI, with the same canonical event contract across all three.
- **An incident report, not a log dump.** From any captured bundle, a reviewer
  gets a session-scoped forensic report: integrity status, signature provenance,
  an ordered event timeline, and **explained findings** — each anomaly named,
  rated, and pinned to the event that triggered it. Exportable as a
  self-contained, independently re-verifiable evidence package, and as NDJSON for
  a SIEM. *(Shipped today.)*
- **Accountable.** Every privileged action carries who acted (principal:
  human/agent/tool, on behalf of whom), under what scope, and whether it
  succeeded, was denied, or failed — with dedicated error events, not overloaded
  success records. *(Shipped today.)*
- **Honestly bounded.** ATB keeps proving integrity without ever pretending to
  prove completeness, identity, or regulatory compliance. The
  [provability ladder](./provability-ladder.md) stays the public, honest framing.

### Custos (the custody layer) should be

- **An auditable custody log, not a write-only receipt printer.** You can submit
  a bundle, get a signed receipt, enumerate everything held, fetch the signing
  key, and independently verify both the bundle's integrity and Custos's
  attestation over it. *(Ingest, signed receipts, enumeration, and server-side
  attestation verification are shipped; independent key discovery and inclusion
  proofs are next.)*
- **Transparency-log shaped.** Following Rekor's lesson: append-only, publicly
  verifiable, with inclusion proofs, so trust in a receipt does not require trust
  in the operator. A holder verifies offline against a published key.
- **Independently re-verifiable over time.** Anyone holding a receipt can confirm,
  at any later date, that the bundle still verifies and that Custos attested it —
  without trusting the store. *(Shipped: `GET /receipts/:id/verify` and
  `/attestation`.)*
- **Anchored to independent time.** Receipts chain to an RFC 3161 / eIDAS-class
  timestamp so "when" carries legal weight, not just the operator's clock.

### What stays explicitly out of scope

Managed storage as a feature of ATB, hosted tracing/telemetry collection,
real-time prevention or blocking of agent actions, model evaluation, training-data
governance, and universal capture-completeness guarantees. These are either other
products' jobs or claims ATB refuses to make.

## How adoption is meant to flow

1. A developer drops in ATB (SDK or `atb intercept`) to record one risky agent
   path locally — no account, no network.
2. An incident happens; `atb incident report` / `export` turns the bundle into a
   reviewable, independently verifiable evidence package. This is the "aha".
3. The organisation needs those packages held somewhere neutral and tamper-proof
   for retention and audit; they stand up Custos (or a hosted operator of it).
4. Compliance and audit consume the same evidence — receipts, attestations,
   re-verification, NDJSON into the SIEM — without anyone reconstructing the
   story from mutable dashboards.

Each step is independently useful, which is the point: the recorder earns trust
locally before custody is ever introduced.

## The one-line positioning

- **ATB** — a local-first flight recorder for AI agents: portable, hash-chained,
  signed evidence bundles you can verify yourself, even when the agent's own logs
  cannot be trusted.
- **Custos** — the custody and attestation layer for that evidence: it ingests
  ATB bundles, issues independently verifiable signed receipts, and anchors them
  to trusted time. Rekor for AI execution evidence.
