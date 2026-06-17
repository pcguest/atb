# EU AI Act Article 12: ATB evidence mapping

Regulation (EU) 2024/1689, Article 12, requires high-risk AI systems to allow
the automatic recording of events over the system lifetime, with traceability
appropriate to the intended purpose. This page maps each Article 12 obligation
to the ATB primitive that produces the evidence, and states plainly what ATB
does not prove.

This is a technical evidence map. It is not a conformity assessment, a legal
opinion, or a certification claim. ATB proves the integrity of recorded bundle
contents. It does not prove capture completeness, actor identity, provider-side
behaviour, or regulatory compliance by itself. Wording such as "aligns with
Article 12 logging requirements" is accurate. "Certified" or "compliant" is not.

## Per-obligation mapping

| Article 12 obligation | What it requires | ATB primitive providing the evidence | What ATB does not prove |
| --- | --- | --- | --- |
| 12(1) Automatic recording of events (logs) over the lifetime | The system technically allows automatic logging of events. | Append-only, hash-chained bundles over RFC 8785 canonical JSON. Events arrive from the SDKs, the `atb intercept` proxy, `atb capture run`, and OTel or chatlog import. Each record links to the previous by SHA-256. | That every relevant event was recorded at source. ATB cannot prove nothing was omitted before capture. |
| 12(2)(a) Events relevant to identifying risk situations or substantial modification | Logs enable identification of situations that may present a risk under Article 79(1), or a substantial modification. | `ai.policy.decision` (deny), `ai.action.error`, and the incident anomalies provable from the bundle alone: `tool_without_approval`, `policy_denied_executed`, `action_failed`, `unresolved_identity`, `session_not_closed`. Obligation profiles and the Capture Assurance Score grade structural coverage. | That the risk taxonomy is complete, or that the model or agent behaved correctly. ATB records what happened; it does not judge it. |
| 12(2)(b) Events facilitating post-market monitoring (Article 72) | Logs support the provider's post-market monitoring. | Deterministic `atb compliance pack` export, `verify.report.v1` machine-readable custody report, and retention audit events. Bundles are portable and verify offline with only the `atb` CLI. | Continuing remote enforcement, or legal compliance of the monitoring programme. |
| 12(2)(c) Events for monitoring operation under Article 26(5) | Logs support deployer monitoring and human oversight of operation. | Oversight profiles, `ai.human.approval` and override events, optional reviewer `identity_evidence`, and per-session anomaly flags surfaced by `atb incident` and the viewer. | The asserted reviewer identity. Identity evidence is caller-provided and is labelled unverified until checked against the deployment IdP, JWKS, or PKI. |
| Record-keeping duty (Articles 19 and 26(6)) | Providers and deployers keep the automatically generated logs for the required period. | Bundles are the retained artefact. The retention guard blocks local retention below the EU AI Act minimum unless the operator explicitly allows it. Retention policy changes, local archive operations, and accepted S3 Object Lock requests are recorded in `.atb/operations.atb`. | That storage-side retention is actually enforced. The `independently_verified` flag distinguishes API acceptance from proof of enforcement. WORM custody is the Custos layer, outside ATB core. |

## What ATB proves

- Every event in a bundle can be verified unmodified since capture, using only the `atb` CLI and the bundle file.
- The hash chain links each record to the previous one over canonical JSON.
- Signed bundles can be checked against recorded Ed25519 signatures.
- An unapproved or failed privileged action is provable from the bundle alone.

## What ATB does not prove

- That capture was complete. The Capture Assurance Score grades structural coverage, not universal capture.
- That the actor or reviewer is who the record claims. Identity evidence is caller-provided unless independently verified.
- That the model or agent behaved correctly.
- That storage-side retention continues to be enforced after the recorded operation.

## Produce the evidence

```bash
atb verify --bundle run.atb/bundle.atb --format json
atb compliance pack \
  --bundle run.atb/bundle.atb \
  --profile atb.profile.policy_decision \
  --regime eu-ai-act \
  --out eu-ai-act-pack.zip
```

The deterministic pack contains the bundle, `verify.report.v1`, trust reports,
CAS and obligation results, incident artifacts, reference mappings, and relevant
retention operations when `.atb/operations.atb` exists.

See [EU AI Act coverage map](./eu-ai-act.md) for the broader Article 9 to 20
view and the full known-gaps list.
