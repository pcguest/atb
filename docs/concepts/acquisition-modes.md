# Automated acquisition: modes, contract, and next phase

Status: **direction / roadmap** — not shipped. This document records the bounded
design direction for automated acquisition in ATB. It describes no connector
implementation and promises no provider capability that a provider API may not
expose. It complements, and does not replace, the existing acquisition
specification in [`bundle-v1.md`](../specification/bundle-v1.md),
[`events.md`](../specification/events.md),
[`attribution.md`](../specification/attribution.md), the chatlog source-digest
contract in [`chatlog-import.md`](../integrations/chatlog-import.md), and the
acquisition-continuity invariants in [`verification-meaning.md`](../verification-meaning.md).

## Principle

> Automated acquisition is an evolution of ATB's existing acquisition subsystem,
> not a fourth product and not a replacement evidence model.

ATB already records *how* an event entered the system (the hashed `acquisition`
envelope) and *how far* an acquisition has progressed (checkpoints and
reconciliation) — today on the **import** path only. Automated acquisition
generalises that same evidence model across four user experiences. It does not
introduce a new trust primitive, and it does not change what ATB proves:
**integrity of what was recorded, never completeness, truth, or authority.**

## The four acquisition experiences

These are **user experiences**, not wire modes. They describe when observation
begins; the wire field is `event.acquisition.mode`, whose vocabulary is `live`,
`retrospective`, `replayed`, `derived`, `manual` (defined with the canonical
`acquisition` envelope in
[`bundle-v1.md`](../specification/bundle-v1.md) §3.4 and
`schemas/event.v1.json`). The names below describe provenance, not trust, and
are not scored.

> **Shipped vs proposed.** Today only **IMPORT** is implemented, and the only
> mode ATB writes is `retrospective` (the chatlog importer). `atb capture run`
> and `atb intercept` exist but currently write **no** `acquisition` envelope.
> START, ATTACH, and CONTINUOUS are direction only; their `live` mode is a
> proposal, not current behaviour, and how the existing capture surfaces map to
> these experiences is an archaeology-phase question (see *Open questions*).

| Experience | When it begins | Proposed `mode` | New work |
|------------|----------------|-----------------|----------|
| **START**  | before the interaction | `live` (proposed) | session lifecycle; observation start boundary |
| **ATTACH** | during an existing interaction | `live` + `retrospective` (proposed) | explicit observation-start time; history-recovery honesty |
| **IMPORT** | after the interaction | `retrospective` (**shipped**) | promote the existing chatlog/OTel paths to the common adapter contract |
| **CONTINUOUS** | long-running workloads | `live` (proposed) | checkpointed, unattended capture for organisations |

### START — before the interaction

The user deliberately begins an AI interaction under ATB observation.
Conceptually (exact surface to be determined later — **do not standardise the
CLI in this phase**):

```text
atb start -- <provider/tool/application> <context>
```

ATB establishes an **acquisition session**: source identity, observation start,
an initial checkpoint, and relevant context. Supported source observations are
then recorded as the interaction occurs.

### ATTACH — during an existing interaction

ATB begins observing an already-running source/session. The provenance must be
honest and explicit:

- observation began at time T;
- earlier activity was **not** observed live;
- historical recovery may or may not be available from the source.

If earlier history is recovered, records recovered before T are recorded as
`retrospective` acquisition and records observed after T as `live`. These two
are **epistemically distinct observations** and must never be collapsed into a
single "live" claim.

### IMPORT — after the interaction

The existing, mature path:

```text
export/log/file → source adapter → retrospective acquisition → ATB evidence
```

`atb import chatlog` and `atb import otel` already implement this and already
carry source identity, source digest, adapter/version, checkpoint, and
reconciliation. The chatlog source-digest contract is documented in
[`chatlog-import.md`](../integrations/chatlog-import.md); the OTel import path
shares the acquisition envelope but has no dedicated specification yet. The next
step is to express both through the **common adapter contract**, not to replace
them. Existing chatlog behaviour and its documented source-digest
equality/inequality semantics are unchanged.

### CONTINUOUS — long-running workloads

For applications and organisations:

```text
AI workload → OTel / SDK / native adapter → checkpointed acquisition → ATB evidence
```

No user should have to manually begin every production interaction. This is
where checkpointing, resumability, and unattended operation matter most. It
remains local-first: ATB records and verifies; hosted multi-tenant collection
stays a Mortise concern (see *Non-goals*).

## Common acquisition contract

Investigate a common `SourceAdapter` / `SourceObservation` contract. **Do not
fix the exact Go interfaces until repository archaeology is complete** — the
existing `internal/acquisition`, `internal/capture`, and `internal/proxy`
packages must be mapped first so the contract absorbs them rather than
duplicating them.

Conceptual responsibilities:

```text
source identity        stable id for a source record (existing event.SourceIdentity)
capabilities           what this adapter can actually do (declared, not assumed)
observe / read         produce observations (historical or live)
checkpoint             persist / resume a stream position (existing acquisition.Checkpoint)
acquisition metadata   mode, adapter, adapter_version, source_digest, timestamps
reconciliation         new / unchanged / changed / unknown (existing reconcile.go)
```

Candidate capability vocabulary (declarative; an adapter advertises only what it
can honour):

```text
HISTORICAL_IMPORT   POLL   STREAM   WEBHOOK   OTEL   SDK
```

Adapters produce **observations**. Adapters NEVER determine truth, trust,
compliance, approval, policy, or authority. Those boundaries are unchanged from
the rest of ATB.

## Control plane vs evidence plane

A hard distinction, preserved in code and in schemas:

| Control plane (NOT evidence) | Evidence plane (recorded, hashed) |
|------------------------------|-----------------------------------|
| provider credentials, OAuth/API tokens | `SourceIdentity` |
| connector configuration | `AcquisitionInfo` |
| schedules | observation / acquisition mode |
| operational checkpoints | source digest |
| connection health | adapter + adapter version |
|  | relevant checkpoint/provenance representation |
|  | canonical event, hash chain, relationships |

**Provider secrets must never enter evidence.** This restates the canonical rule
in [`bundle-v1.md`](../specification/bundle-v1.md) §3.4: adapters and producers
MUST NOT place secrets, tokens, or unnecessary personal data in acquisition
fields. Control-plane state may live in operator-local configuration (the
checkpoint path is the precedent: an operator-selected local path, explicitly
not remotely request-controlled evidence). Note the checkpoint distinction: a
checkpoint **file** on disk is operational state; the checkpoint **fields**
inside a hashed `acquisition` envelope are evidence.

## Context

A project/repository/file may be useful acquisition **context** but must not
automatically become evidence identity. For development agents, preserve
relationships such as:

```text
AI session → repository → commit/branch where observed → relevant files/artefacts → tool calls → source observations
```

Do not assume filesystem paths are permanent identities; treat them as observed
context attached to a session/repository identity.

## Provenance is not trust

`START` / `ATTACH` / `IMPORT` / `CONTINUOUS` describe how evidence was acquired.
They are NOT trust scores. Never encode `live = trusted` or
`retrospective = untrusted`. ATB records what it observed and how. Mortise may
later apply an explicit policy requiring a particular acquisition mode; that is
a governed product decision, not an ATB verdict.

## Interaction with the manifest-version profile

Automated acquisition does not add a second acquisition encoding. All four
experiences record the same hashed `acquisition` envelope, so they inherit the
manifest-version rules already specified in
[`bundle-v1.md`](../specification/bundle-v1.md) §3.4 and §9: acquisition-bearing
bundles declare manifest version 3, a pre-v3 reader fails loudly rather than
mis-hashing, and the non-acquisition default writer stays v1. Those rules are
single-sourced there and are not restated here.

## Dogfood target

The first serious live-acquisition pilot uses **OpenCode** (an interactive
coding agent that works in a repository) as the source:

```text
OpenCode works on ATB
      ↓
ATB automatically records supported OpenCode development observations
      ↓
user investigates them in ATB
      ↓
selected evidence enters Mortise
      ↓
governance review / decision
```

This is a **future implementation target, not part of the current publication
run**. What OpenCode exposes for observation has not yet been inventoried, so
the concrete observation set is a phase-1 archaeology output; provider
capabilities that OpenCode does not expose must not be promised. The minimum
prerequisites are phases 1–4 below: a mapped `SourceAdapter` seam, IMPORT
unification, session lifecycle, and a durable checkpointer.

## Phased plan (proposal, bounded)

1. **Foundation / archaeology.** Map `internal/acquisition`, `internal/capture`,
   `internal/proxy`; identify the seams for a `SourceAdapter` contract without
   changing existing import semantics or canonical hashing. *Output: a written
   seam map stating what is shipped today, how `capture run`/`intercept` relate
   to the four experiences, and the proposed adapter responsibilities. No
   user-facing change.*
2. **IMPORT unification.** Express chatlog and OTel importers as adapters behind
   the contract; keep byte-identical behaviour and golden vectors.
3. **START/ATTACH session lifecycle.** Observation-start boundary, session
   identity, honest live-vs-retrospective split on ATTACH.
4. **CONTINUOUS checkpointer.** Long-running, resumable, unattended capture;
   durability and failure semantics for checkpoints.
5. **OpenCode dogfood.** A native/OTel/SDK adapter for OpenCode development
   observations; end-to-end into the existing ATB→Mortise governance path.
6. **Context relationships.** Session→repository→commit→files→tool-call
   relationships, as observed context — not identity.

## Non-goals

- Hosted tracing, telemetry collection, or multi-tenant review (roadmap:
  "not planned for ATB core").
- A fourth product identity, or a replacement evidence model.
- Provider connectors in this phase.
- Encoding trust, compliance, approval, or authority in acquisition metadata.
- Claims of capture completeness or provider-side correctness.
- Real-time universal prevention of AI actions.

## Open questions (for the archaeology phase)

- What is the minimal `SourceAdapter`/`SourceObservation` interface that absorbs
  import, capture, and intercept without new canonical fields?
- How is an acquisition **session** identified across START/ATTACH/CONTINUOUS,
  and how does it relate to `capture_run_id` and `SourceIdentity`?
- Which capabilities can each real source honestly declare, and how is an
  unavailable capability surfaced rather than implied?
- What checkpoint durability guarantees are required for unattended
  CONTINUOUS operation, and how do they interact with the manifest-v3 floor?
- Where does operator-local control-plane configuration live, and how do we
  prove no secret can reach the evidence plane?
