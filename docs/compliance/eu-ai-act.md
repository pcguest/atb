# EU AI Act — ATB Coverage Map

Regulation (EU) 2024/1689 applies broadly from 2 August 2026, subject to phased and sector-specific provisions. ATB targets runtime evidence for high-risk AI system obligations under Title III by recording local audit events into verifiable bundles.

This page is a technical evidence map, not a conformity assessment, legal
opinion, or certification claim. ATB proves integrity of recorded bundle
contents. It does not prove capture completeness, actor identity,
provider-side behaviour, or regulatory compliance by itself.

## Coverage summary

| Article | Requirement | ATB coverage | Status |
| --- | --- | --- | --- |
| Article 9 | Risk management system | Obligation profiles + CAS scoring | Partial |
| Article 12 | Automatic logging | Hash-chained bundles, capture-scope attestations, SDK/proxy capture, retention audit events | Supporting evidence |
| Article 13 | Transparency to deployers | verify.report.v1, CAS residual-risk output | Partial |
| Article 14 | Human oversight | Oversight profiles plus optional reviewer identity evidence anchored to an external IdP assertion digest | Supporting evidence |
| Article 17 | Quality management documentation | Specs, obligation profiles, deterministic compliance evidence pack | Partial |
| Article 20 | Automatically generated logs | capture run, import chatlog, OTel span mapping | Partial |
| Article 10 | Training data governance | Not applicable — runtime capture only | Out of scope |
| Article 11 | Technical documentation | docs/spec/ covers runtime; training docs not in scope | Out of scope |

## What ATB records and verifies

- Every event appended to a bundle can be verified to be unmodified since capture, using only the `atb` CLI and the bundle file.
- Each record is linked to the previous record by the documented SHA-256 hash chain over RFC 8785 canonical JSON.
- Signed bundles can be checked against recorded Ed25519 signatures.
- The retention guard blocks local retention configuration below the EU AI Act minimum unless the operator explicitly allows it.
- Retention policy changes, local archive operations, and accepted S3 Object Lock requests are recorded in `.atb/operations.atb`.
- `verify.report.v1` records verifier results, CAS output, and profile gaps in a machine-readable custody report.
- `atb compliance pack` combines the authoritative bundle, profile report, CAS, obligations, incident reports, mappings, and relevant retention evidence without network access.
- Local-first operation avoids a mandatory hosted logging service or vendor custody path.

## Article 14 reviewer identity evidence

Deployments with an existing IdP, SSO layer, signed JWTs, or PKI can attach an
optional `identity_evidence` object to policy, approval, action, and override
events. The object records:

- identity provider and subject;
- authentication context;
- assertion type (`jwt`, `saml`, `x509`, or `opaque`);
- a digest of the assertion and, optionally, a digest of separately retained raw evidence.

ATB proves the recorded object was not altered after capture. It does not
authenticate the reviewer itself or validate the original assertion. For a
review, verify the retained assertion against the deployment IdP/JWKS/PKI and
use external corroboration where an independently retrieved identity-system
record is required.

Trust reports, verifier reports, incident reports, and compliance packs label
this data as caller-provided and unverified by ATB.

## Generate an EU AI Act evidence pack

```bash
atb compliance pack \
  --bundle run.atb/bundle.atb \
  --profile atb.profile.policy_decision \
  --regime eu-ai-act \
  --out eu-ai-act-pack.zip
```

The deterministic pack contains the bundle, `verify.report.v1`, trust reports,
CAS and obligation results, incident artifacts, reference mappings, and
relevant retention operations when `.atb/operations.atb` exists.

## Known gaps (honest)

- Bypass via direct provider API calls remains possible; no subprocess interception exists in ATB core, and the hard-boundary project handles that separately.
- Identity truth remains dependent on independent IdP/JWKS/PKI verification.
- Retention events prove recorded local operations or API acceptance, not continuing remote enforcement.
- Completeness is structural (CAS), not proof of universal capture.
- Custodian-of-record workflows, legal hold, auditor access, and hosted custody belong outside the ATB runtime.

## Roadmap

OTLP/protobuf transport, database reconciliation packs, and CAS v1
formalisation remain future work. See [docs/roadmap.md](../roadmap.md) for the
current milestone list and explicit out-of-scope boundaries.
