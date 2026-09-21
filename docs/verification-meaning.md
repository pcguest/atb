# What ATB Verification Means (and What It Doesn't)

## The Short Answer

**ATB verification proves the integrity and order of records in a bundle.**

It does **not** prove:
- That every relevant event was captured
- That the recorded events are true or correct
- That the agent behaved safely or correctly
- That no unrecorded events exist outside the bundle

---

## What Verification Proves

When `atb verify` (or `bundle.verify()`) returns **PASS**:

| Property | What It Means |
|----------|---------------|
| **Hash-chain integrity** | Every record's hash matches the computed hash of its canonical JSON + previous hash. No record was modified, reordered, inserted, or deleted without detection. |
| **Event order** | The sequence numbers (1, 2, 3...) match the actual order records were appended. |
| **Profile obligations** | The required event types for the selected profile are present. |
| **Relation consistency** | Declared relations between events (e.g., policy decision → action) are satisfied. |

---

## What Verification Does NOT Prove

| Not Proven | Why |
|------------|-----|
| **Capture completeness** | ATB only verifies what's in the bundle. Events outside the capture boundary are invisible. |
| **Truth or correctness** | A record says "action succeeded" — verification confirms the record wasn't tampered with, not that the action actually succeeded. |
| **Policy correctness** | A recorded policy decision "deny" is verified as recorded; ATB doesn't judge if the policy logic was correct. |
| **Safety** | No claim about whether the agent's behaviour was safe or appropriate. |
| **Causality** | Event A before Event B doesn't prove A caused B. |
| **External authority** | ATB doesn't verify who had permission to act; that's Mortise's domain. |
| **Objective continuity** | ATB doesn't track whether an agent's objective drifted; that's Tenon's domain. |

---

## The "Run" Concept

A **Run** is a **human-facing projection** of the captured records in a loaded bundle:

- Starts at the first captured record, ends at the last
- The bundle path identifies this projection
- A recorded `session_id` may relate records to a session, but is not required
- **Does not claim** one bundle = one complete session, lifecycle, or all activity

---

## Evidence Status

The **Evidence Status** surface answers three independent questions:

1. **Integrity** — Is the recorded evidence intact? (Hash chain verified/failed)
2. **Coverage** — Does the selected evidence profile pass? (Profile PASS/FAIL)
3. **Corroboration** — Is external/organisational custody recorded? (Yes/No)

> **Evidence Status is not a trust score, health score, or safety rating.**

---

## Findings

Findings are **bounded anomaly conclusions** derived from captured evidence:

- Each finding has a **flag** (e.g., `tool_without_approval`), **severity**, and **supporting event sequences**
- Findings state what ATB **can** and **cannot** conclude
- **Empty findings ≠ no problems** — it means no anomalies in captured evidence

Example:
> `tool_without_approval` (HIGH) — "No matching earlier human approval exists in the captured session evidence for this tool call."
> **What ATB cannot conclude:** "ATB cannot conclude that no approval existed outside the captured evidence."

---

## Evidence Integrity vs Capture Completeness

| Concept | Question | ATB Answers |
|---------|----------|-------------|
| **Integrity** | "Is this exact bundle intact?" | Yes/No (deterministic) |
| **Completeness** | "Was everything captured?" | Unknown (not provable) |

> **Integrity** = "The presented records match their hash chain and recorded order."
> **Completeness** = "All relevant events were captured." — **ATB cannot prove this.**

---

## CAS (Completeness Assurance Score)

The CAS score estimates **profile-scoped evidence coverage**, not universal truth:

| Sub-score | What It Measures |
|-----------|------------------|
| EC (Event Coverage) | Required event types present |
| FC (Field Completeness) | Required fields populated |
| RC (Relation Consistency) | Declared relations satisfied |
| TC (Temporal Consistency) | Timestamps ordered correctly |
| SC (Source Commitment) | Policy signatures, identities |
| XC (External Corroboration) | External receipts, anchors |
| AC (Anchor Coverage) | RFC 3161 timestamps |
| GC (Gating Completeness) | Profile gates satisfied |

> **CAS is an estimate, not a proof.** A score of 1.0 doesn't mean complete capture.

---

## Tamper Detection

ATB detects three tamper classes deterministically:

| Tamper Type | Detection |
|-------------|-----------|
| **Content mutation** | Hash mismatch at modified event |
| **Reordering** | Sequence number mismatch |
| **Removal** | Sequence gap detected |

> Verification **FAIL** on tampered bundle is definitive proof of tampering.

---

## Quick Reference

| If you see... | It means... |
|---------------|-------------|
| `Integrity: PASS` | Bundle hash chain is intact |
| `Profile: PASS` | Required events for profile present |
| `CAS: 0.70 (Medium)` | Estimated coverage is 70% (not 70% truth) |
| `Finding: tool_without_approval` | No approval in **captured** evidence |
| `Verified` | Hash chain + profile obligations met |
| `Not Verified` | Hash chain broken OR profile obligations not met |

---

## Further Reading

- [Quickstart Guide](../getting-started/quickstart.md) — Create and verify your first bundle
- [Viewer Specification](../specification/viewer.md) — Investigation UI details
- [Trust Model](../concepts/trust-model.md) — ATB's trust boundaries
- [CAS Scoring](../evidence/cas.md) — Completeness Assurance Score details
- [Incident Forensics](../investigate/incidents.md) — Incident investigation workflow