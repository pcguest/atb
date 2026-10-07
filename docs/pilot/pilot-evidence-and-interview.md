# Pilot evidence collection, interview guide and security/privacy questionnaire

Companion to the [external design-partner pilot protocol](external-pilot-protocol.md).
It contains the blank evidence template, the champion interview guide, the
security/privacy questionnaire, the operating-burden measurements, and the
Mortise signal definitions. Do not translate participant answers into desired
product conclusions.

## 1. Pilot evidence template (blank)

Record exact commands and trimmed outputs. Exclude secrets.

### Environment

| Field | Value |
| --- | --- |
| ATB version | |
| Binary checksum (SHA-256) | |
| OS / platform | |
| Workload description | |
| Capture configuration (no secrets) | |
| Capture start boundary | |
| Source incarnation (label only) | |
| Relevant dependency versions | |

### Capture

| Field | Value |
| --- | --- |
| Capture start command | |
| `capture status` (text + json) | |
| Observation count | |
| Checkpoint / committed position | |
| Recovery / replay state | |
| Known gaps (`known_gap` / `possible_unknown_gap`) | |
| Degradation events | |
| Final bundle path and identity | |

### Incident

| Field | Value |
| --- | --- |
| Unexpected operation (seeded or natural) | |
| Exact evidence reference (`atb://evidence/...`) | |
| Relevant relationships and context | |
| Source / observation limitations stated | |

### Verification

| Field | Value |
| --- | --- |
| Offline `verify` result (integrity) | |
| Bundle identity | |
| Evidence locator | |
| `export` result | |
| Verified without capture process / network? | |

### Governance (where Mortise is in scope)

| Field | Value |
| --- | --- |
| Immutable `EvidenceReference` preserved | |
| Resolution result | |
| Capture boundary visible | |
| ATB evidence mutated? (must be no) | |

### Practitioner

| Field | Value |
| --- | --- |
| Tasks attempted | |
| Outcome per task | |
| Confusion / semantic mistakes | |
| Assistance required (record every intervention) | |
| Evidence relied upon | |

### Commercial

| Field | Value |
| --- | --- |
| Existing workflow | |
| Current alternative | |
| Perceived value | |
| Operating burden | |
| Security / privacy objections | |
| Deployment objections | |
| Integration objections | |
| Willingness to continue | |
| Willingness to champion | |
| Willingness to pay / procure | |

## 2. Semantic-understanding check

Record whether the participant correctly distinguished, with the evidence that
shows it:

- integrity;
- coverage;
- capture gaps;
- recorded sequence;
- relationships;
- evidence;
- interpretation;
- governance;
- execution.

Explicitly probe these false interpretations. Any documentation or interface that
materially encourages one is a product defect.

    integrity = truth
    integrity failure = malicious tampering
    absence = non-occurrence
    sequence = causality
    relationship = causation
    capture process alive = complete capture
    replay success = no observation loss
    digest = original source content
    source identity = authenticated real-world identity
    custody = truth
    interpretation = evidence
    decision = execution

## 3. Champion interview guide

Interview separately, after the practical exercise. Do not lead. Record answers
verbatim.

1. What problem did this solve for you?
2. What would you have done without ATB?
3. Which existing systems would you normally use?
4. What was easier?
5. What was harder?
6. What evidence would you still need during a real incident?
7. What would prevent deployment?
8. What would your security/privacy team object to?
9. Who would operate this?
10. Who would own it?
11. Who would pay for it?
12. Which budget would it come from?
13. Would you run another pilot?
14. Would you deploy it to a real workload?
15. Would you personally champion the purchase?
16. What would need to exist before you could?
17. Would you prefer the evidence to go into: Mortise, SIEM, ticketing, GRC,
    incident platform, object storage, something else?
18. Which parts would you expect to be free/open?
19. Which operational capabilities would you pay for?
20. If ATB disappeared tomorrow, is the retained evidence still useful?

## 4. Security / privacy questionnaire

The participant or their security team should be able to answer "where does every
relevant byte go?". If they cannot, stop.

- What data crosses the capture boundary (headers, bodies, query strings, tool
  arguments, model/usage metadata)?
- What data is retained, where, and in what form (raw vs digest)?
- What is deliberately excluded?
- What are the credential boundaries; can any credential material enter
  evidence, the journal, logs, or the checkpoint?
- What source privileges and network exposure does the proxy/CA require? Is it
  loopback-only? What is the CA key mode?
- Who can access the evidence; what are the deletion/retention expectations?
- How does export work, and what leaves in it?
- Which operations require the network, and what works fully offline?
- Can the evidence be verified without the vendor or the paid capture service?
- What failure behaviour could lose or leak data?

## 5. Operating-burden measurements

Measure; do not optimise prematurely.

| Measurement | Value |
| --- | --- |
| Installation time | |
| Configuration burden | |
| Manual recovery interventions | |
| Restart behaviour | |
| Storage growth | |
| Journal growth | |
| CPU / memory overhead (where practical) | |
| False alarms | |
| Capture degradation events | |
| Upgrade friction | |
| Evidence transfer effort | |
| Investigation effort | |

## 6. Competitive substitution test

Ask directly and record the answer; do not answer for them:

> Why would you not just use your existing logs, traces, SIEM, observability
> system and object storage?

Test whether ATB uniquely provides meaningful value through some combination of:
stable evidence identity; retained observations; provenance; explicit gaps;
reconciliation; portable evidence; independent verification; AI/tool/context
relationships; exact governance references. If the participant sees no meaningful
incremental value, record that.

## 7. Mortise signal

Observe where participants want ATB evidence to go. Use the pilot evidence to
determine which hypothesis is supported:

- **H1** — customers want a dedicated governance product.
- **H2** — customers primarily want ATB evidence integrated into existing SIEM,
  incident response, ticketing, GRC or audit systems.
- **H3** — different customers require both.

Report one of: `MORTISE_NATIVE_WORKFLOW`, `EXISTING_SYSTEM_INTEGRATION`, `BOTH`,
`INCONCLUSIVE`. Architecture alone does not prove Mortise deserves to be a
separately purchased product.
