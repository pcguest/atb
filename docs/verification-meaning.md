# What ATB Verification Means (and What It Doesn't)

## The Short Answer

**ATB verification proves the integrity and order of the records presented in a bundle.**

It does **not** prove:
- That every relevant event was captured
- That the recorded events are true or correct
- That the agent behaved safely or correctly
- That no unrecorded events exist outside the bundle

---

## What Verification Proves

When a verification result is described as **PASS**, read the label that carries that word. `Integrity: PASS` is the hash chain. `Profile: PASS` is a selected profile. The stable JSON field `pass` is a combined gate and stays false when no profile was evaluated. See [Machine-readable results](#machine-readable-results).

| Property | What It Means |
|----------|---------------|
| **Hash-chain integrity** | Every present record's hash matches the computed hash of its canonical JSON + previous hash. A change that is inconsistent with the recorded hashes (a modified, reordered, or inserted record; a non-tail removal) is detected. A fully recomputed chain is not detected without an independent head commitment — see [Limitation: recomputed chains and unsigned tail truncation](#limitation-recomputed-chains-and-unsigned-tail-truncation). |
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
| **External authority** | ATB doesn't verify who had permission to act; that requires an independent identity and policy authority. |
| **Objective continuity** | ATB doesn't determine whether an agent's objective drifted; that requires application-specific policy and review outside ATB. |
| **External custody** | A local integrity result does not show that a bundle was retained by anyone else. Custody is a separate check of a receipt, store, and signer. |
| **Legal or regulatory compliance** | Verification is not a certification, conformity assessment, or compliance determination. |

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

## Source Change vs. Tampering

When an import records acquisition provenance, each record carries a **source digest** of the external source it was read from. A later re-import that observes the same source identity with a **different** source digest produces a bounded `source_record_changed` finding.

| Question | ATB Answers |
|----------|-------------|
| "Did the external source change since it was last observed?" | Yes/No (deterministic, per recorded observation) |
| "Was the change malicious, unauthorised, or a correction?" | **ATB cannot determine this** |
| "Was the source complete when read?" | **ATB cannot prove this** |

> **A changed source digest proves the observed representation changed. It does not prove tampering, and it does not prove the previous or current content is true.**

A `source_record_changed` finding is a **bounded anomaly conclusion**: at least one observed record of the same source identity differed between two acquisitions. The finding states what ATB can and cannot conclude; it never asserts truth, intent, or causation. Source identity, digest, and reproduction rules are defined in [Chatlog import](./integrations/chatlog-import.md).

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

## Detecting modification

Verification recomputes each present record's hash and compares it with the recorded hash. It reports failure, deterministically, when a change to the records **presented** is inconsistent with the recorded hashes:

| Change | Detection |
|--------|-----------|
| **Content mutation** | Record hash mismatch at the modified event |
| **Reordering** | Sequence number mismatch |
| **Removal (non-tail)** | Sequence gap detected |

> Verification **FAIL** means the presented records **could not be verified against their recorded hash chain**. It is not, by itself, proof of tampering, and it does not establish cause or intent.

### Limitation: recomputed chains and unsigned tail truncation

Verification checks the internal consistency of the records presented to it
against their recorded hashes. It does **not**, on its own, establish that those
hashes are the originals. Two cases are therefore not detected by verification
alone:

- **A fully recomputed chain.** A party who can rewrite a record and recompute
  every subsequent recorded hash produces a different but self-consistent chain
  that verifies. Without an independent commitment to the expected content/head,
  the rewrite is indistinguishable from the original.
- **Unsigned tail truncation.** Removing trailing records from an unsigned bundle
  yields a shorter but internally consistent prefix.

Detecting either requires an independent commitment to the expected content/head
that the rewriter cannot recompute — for example an RFC 3161 anchor, a signature
over the head, or a custody receipt that records the expected head hash. To bind
the head, supply the expected head hash out of band and compare it against the
bundle's head hash.

---

## Machine-readable results

`atb verify` keeps four results separate. A false combined gate does not
rewrite a chain result that was reported separately.

| Result | Where it appears | Intact unsigned bundle, no profile, no anchor |
| --- | --- | --- |
| Hash-chain integrity | Text `Integrity: PASS`. Stable JSON `gate_result.chain_valid` and `integrity_valid`. Diagnostic `--json` field `integrity.chain_valid`. | Passes. The presented records match their recorded hashes. |
| Profile evaluation | Header `Profile: none matched` whenever no profile matched. Summary `Profile: NOT EVALUATED` in that same case, including when the chain or a signature failed. `Verification: NOT EVALUATED` only when the chain is valid and any present signature verifies. Stable JSON `profile_id` and `gate_result.profile_pass`. | Not evaluated. `profile_id` is `""` and `profile_pass` is false. That false value is neither a successful profile assessment nor a hash-chain failure. |
| Bundle signature | Text `Signature present` and, when a signature record exists, `Signature verified`. | `Signature present: no`. No bundle signature was checked. |
| Process exit | The process status code. | `0` for this intact unsigned bundle. Exit `2` is a broken chain, a present signature that does not verify, or a requested `--with-anchor` or `--with-snapshot-check` failure. Exit `3` is a selected profile whose obligations failed; the same code is also used for some system errors, so read the report with the code. No profile result by itself is not exit `3`. |

### No profile selected

Stable `atb verify --format json` (`verify.report.v1`) for that bundle keeps:

- `pass: false`
- `gate_result.pass: false`
- `gate_result.chain_valid: true`
- `gate_result.profile_pass: false`
- `profile_id: ""`
- `critical_failures: []`

`pass` is true only when the chain is valid, a profile result exists and
passed, and any required anchor passed. With no profile result, profile
assurance was not established, so the combined gate stays false. Read the
chain from `chain_valid` or the text `Integrity` line. An empty
`critical_failures` array means no profile obligation failed. It does not mean
a profile passed.

`residual_risk.level` may be `High` with a driver such as `No matching
obligation profile evaluated.` That is a statement that no profile ran. It is
not a finding that the hash chain failed.

These booleans, the schema, and the exit codes are unchanged by this
explanation. Callers that already treat an empty `profile_id` as unpassed
should keep doing so, and should not treat `pass: false` alone as a broken
chain.

### Absent optional anchor

Text prints `anchor: not present` when the bundle has no anchor record.

Diagnostic `atb verify --json` can still contain:

- `anchoring.anchor_present: false`
- `anchoring.anchor_required: false`
- `anchoring.status: "failed"`
- summary `anchor: failed (anchor record not present)`

Read `status` with the two booleans. When `anchor_present` and
`anchor_required` are both false, no anchor record exists and no timestamp
token was checked. The `failed` status is a legacy label for that absence. It
is not evidence that an RFC 3161 verification was attempted, and it is not a
clear way to say the anchor is absent. The stable `--format json` gate does
not use `anchoring.status`. A present anchor keeps its own `verified`,
`partial`, or `failed` result.

### Signing and other commitments

An intact chain shows that the records presented match the hashes recorded in
that bundle. It does not show that those hashes are the original head. A
bundle signature, an RFC 3161 anchor, or a custody receipt can commit to a
head that a recomputed chain or an unsigned tail truncation would otherwise
hide. Each commitment has its own verifier and its own trusted key or
authority. None of them proves that every relevant event was captured, that
recorded events are true, that the agent behaved safely or with authority,
that a third party retained the bundle, or that a legal or regulatory duty was
met.

### Trust-report file headings

`atb trust-report --format markdown` prints headings
`## Operational Safety (PASS)` and `## Test Coverage (PASS)`. The text format
prints the same categories as `operational_safety: PASS` and
`test_coverage: PASS`. Those two categories do not read the workflow inside
the bundle.

`Operational Safety` passes when `SECURITY.md` and
`docs/concepts/trust-model.md` are found in the current working directory or
embedded in the CLI binary. `SECURITY.md` is checked twice, once as a security
policy and once as an incident-response document.

`Test Coverage` passes when `cmd/atb/main_test.go`,
`test/golden/golden_test.go`, and `sdk/python/tests/test_properties.py` are
found in the same way.

PASS on either category means the named file was present. It does not mean
the recorded agent was safe, that any test ran, or that a coverage percentage
was measured. The bundle's chain result in the same report is
`Cryptographic Integrity`. The markdown `Verdict`, and the text `Status`,
aggregate every category in the report, including these file checks, the
canonicalization-spec check, the documentation checks, and the chain. A
missing advisory file makes that aggregate `WARN`. On an intact bundle with
no profile the aggregate can still read `PASS` even though stable
`atb verify --format json` keeps `pass: false`. That aggregate is not a
profile assessment. `atb trust-report --format json` is a different document
and does not use these headings.

## Quick Reference

The terminal report keeps these results separate:

- `Integrity: PASS` means the presented hash chain is internally consistent.
- `Signature present: no` means no bundle-signature event was found.
  `Signature present: yes` reports a signature record; only
  `Signature verified: yes` affirms successful cryptographic verification.
  An invalid signature still fails verification and returns exit code 2.
- Profile obligation lines describe recorded events and required fields. A
  recorded `atb.bundle.signature` event alone does not establish a valid
  cryptographic signature; consult the Bundle Signature section.
- Source Commitment (SC) is diagnostic evidence coverage. A manifest or request
  can contribute to SC without any cryptographic bundle signature.
- `anchor: not present` means no anchor was recorded, rather than a
  failed verification attempt. `anchor: not present (required)` identifies an
  unmet anchor requirement. A present anchor retains its `verified`, `partial`,
  or `failed` verification result and explanation.

Missing mandatory profile evidence still fails the applicable profile. With no
matching profile, `Verification: NOT EVALUATED` describes profile evaluation;
inspect integrity and signature results separately. The text wording above
does not change the JSON values or exit codes described in
[Machine-readable results](#machine-readable-results). Neither form establishes
capture completeness, agent truthfulness, safe or authorised behaviour,
external custody, or legal or regulatory compliance.

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

- [Quickstart Guide](./getting-started/quickstart.md) — Create and verify your first bundle
- [Viewer Specification](./specification/viewer.md) — Investigation UI details
- [Trust Model](./concepts/trust-model.md) — ATB's trust boundaries
- [CAS Scoring](./evidence/cas.md) — Completeness Assurance Score details
- [Incident Forensics](./investigate/incidents.md) — Incident investigation workflow
