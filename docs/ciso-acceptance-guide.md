# CISO Acceptance Guide

## 1. Purpose

This guide explains how a security team can evaluate, accept, and rely on ATB bundle evidence during an audit, incident investigation, or regulatory submission. ATB bundles are designed as a tamper-evident evidence layer for recorded workflow activity. They are not a replacement for SIEM, endpoint telemetry, or general logging infrastructure.

## 2. What ATB produces

A security team reviewing an ATB submission should expect three artefacts.

- A bundle file (`.atb`). This is the primary evidence object: a hash-chained NDJSON event log whose integrity can be checked locally.
- A VerifierReport (JSON). This is the machine-readable verification result, including pass or fail status, matched profile id, critical failures, and, where supported by the profile, a CAS score and grade.
- A TrustReport (JSON or Markdown). This is the reviewer-facing evidence summary, organised into profile-specific sections that show which events were present, which fields were populated, and what the verifier concluded.

## 3. How to verify a bundle

1. Run the verifier in JSON mode:

   ```bash
   atb verify --bundle <path> --format json
   ```

   Review these VerifierReport fields first:

   - `pass`: whether the evaluated profile passed its required checks.
   - `cas_score`: the overall completeness score when the matched profile supports CAS.
   - `cas_grade`: the corresponding grade for `cas_score` (`High`, `Medium`, `Low`, or `Insufficient`).
   - `residual_risk`: the verifier's overall evidence risk summary after integrity and profile evaluation.
   - `critical_failures`: the blocking failures, with a `kind` and a `detail` for each missing event, missing field, relation violation, temporal violation, or signature failure.

2. Generate the human-readable trust report:

   ```bash
   atb trust-report --bundle <path> --format markdown
   ```

   This produces a profile-specific section breakdown intended for human review. Use it to inspect which evidence sections passed, which fields were present, which warnings were raised, and which declared blind spots remain outside the bundle.

3. If timestamp anchoring is in scope, verify anchoring as well:

   ```bash
   atb verify --bundle <path> --with-anchor --roots <pem>
   ```

   This adds RFC 3161 timestamp-token verification against the supplied PEM trust roots. Anchoring adds external time-bounding evidence that the recorded bundle hash existed before the TSA timestamp in the token. Anchoring is optional. An unanchored bundle can still be tamper-evident within its own hash chain; anchoring only adds an externally verifiable time bound.

## 4. Understanding the CAS score

The CAS score is a completeness measure for the matched profile, not a judgement on whether the underlying decision or action was correct. Security teams should read it as an evidence-quality signal. A low score means the bundle may still be intact, but material parts of the expected workflow evidence are missing or weak. ATB currently computes eight CAS sub-scores.

| Sub-score | What it measures | Low score means |
|-----------|------------------|-----------------|
| `EC` | Whether the required event types for the profile are present | One or more required lifecycle records are missing |
| `FC` | Whether required fields are populated on the recorded events | Events exist, but key identifiers, digests, or outcomes are absent |
| `RC` | Whether related events are consistently linked, for example by `action_id` or `request_id` | The bundle does not reliably show that records belong to the same workflow instance |
| `TC` | Whether the workflow is recorded in a causally credible order | Timing or sequence data leaves the chronology uncertain |
| `SC` | Whether the evidence is bound to its originating source and governance context | The bundle gives weaker proof of origin, signing, or provenance context |
| `XC` | Whether external corroboration is present beyond the bundle itself | The reviewer has weaker corroborating evidence outside the local record set |
| `AC` | Whether any RFC 3161 anchor covers the relevant bundle state | The reviewer cannot rely on the anchor as a complete time-bounding record |
| `GC` | Whether the gated control-plane path is fully represented from intent to committed effect | The reviewer cannot show that the expected gate operated end to end |

ATB emits CAS for all six built-in profiles. The sub-score set and weighting are profile-scoped; not all sub-scores are equally weighted across profiles.

## 5. Residual risk levels

Residual risk is the verifier's acceptance summary after integrity checks and profile evaluation.

- Critical: chain integrity failed; bundle must not be accepted as evidence without investigation.
- High: chain intact but significant evidence gaps remain; acceptable only with documented compensating controls.
- Medium: chain intact and the workflow is partially evidenced; acceptable for most internal audit purposes.
- Low: chain intact with strong evidence coverage; suitable for regulatory submission.

## 6. Regulatory alignment

EU AI Act Article 12. ATB bundle evidence is designed to assist with lifecycle logging and auditability expectations by preserving a tamper-evident record of recorded workflow events, control points, and evidence gaps. It supports review of what was logged and whether the evidence chain still verifies. It does not perform conformity assessment, and it does not itself determine whether the surrounding AI system satisfies Article 12 in full.

NIST AI RMF. ATB evidence is designed to assist with governance and operational oversight by supporting the `GOVERN` and `MANAGE` functions, particularly where an organisation needs to demonstrate that high-risk actions, policy decisions, or model outputs were recorded and can be reviewed later. It does not address `MAP` or `MEASURE` in full, because it does not by itself establish risk identification completeness, performance testing sufficiency, or model quality evaluation.

UK DSIT AI Code of Practice. ATB evidence is designed to assist with transparency and explainability duties by supporting later review of what inputs, decisions, approvals, and outputs were recorded for a workflow. It supports the production of explainability artefacts for audit and incident response, but it does not stand in for the wider governance, impact assessment, or documentation regime required around the system.

## 7. Key management requirements

Ed25519 bundle signing is optional, but it is strongly recommended for regulatory submissions and any evidence handoff that depends on organisational provenance. The security team should verify that the public key embedded in the signature record matches a key under the submitting organisation's control, and that the key was not generated after the events it signs. Signature validity shows that a holder of the private key signed the bundle state; it does not by itself prove authorisation. Key rotation workflow is documented separately in `docs/key-management.md`.

## 8. Limitations

- ATB records what the instrumented code emits; it cannot detect uninstrumented actions.
- CAS scoring reflects evidence completeness, not correctness of the AI decision.
- TSA anchoring depends on the availability and trustworthiness of the chosen TSA; DigiCert and Sectigo are tested.
- The dashboard preview (`atb view --ui-experimental`) is not yet a formally specified API surface; auditors should use the CLI or JSON outputs.
