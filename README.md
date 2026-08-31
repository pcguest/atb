# ATB — independently verifiable evidence for AI agents

[![CI](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml) ![Go version](https://img.shields.io/badge/go-1.26.7-blue) [![Licence](https://img.shields.io/badge/licence-MIT-green.svg)](LICENSE)

ATB is an open-source, local-first evidence system for AI agents. It captures
agent and tool activity into portable, tamper-evident bundles that can be
independently verified offline.

**ATB proves the integrity of what was recorded. It does not prove that every
relevant event was captured, that an actor was honest before capture, or that
the recorded activity was correct.**

When an agent incident occurs, ATB lets an investigator verify the evidence,
reconstruct recorded actions, and trace deterministic findings back to
hash-addressed bundle events without relying on the agent application's own
logs. It requires no service, cloud account, external database, or hosted
verifier.

Source version: [`v1.15.4`](CHANGELOG.md). A source tag does not imply that
every registry or GitHub release artefact has been published; check the
installed CLI/SDK version and the relevant registry before relying on a
particular release.

## Try ATB in five minutes

Install the CLI:

```bash
go install github.com/pcguest/atb/cmd/atb@latest
```

Create a bundle, record a decision, and verify its integrity and evidence
obligations:

```bash
atb bundle new
atb append ai.request.received --data='{"request_id":"req-1042","purpose_tag":"support-triage"}'
atb append ai.policy.decision --data='{"policy_id":"routing","policy_version":"1","decision":"deny","decision_reason_codes":["manual_review"],"subject_id_hash":"sha256-subject","action_id":"act-1042"}'
atb verify --profile atb.profile.policy_decision run.atb/bundle.atb
```

`go install` provides the CLI and a minimal install-guidance page for
`atb view`. The full embedded local viewer is included by a source build:

```bash
git clone https://github.com/pcguest/atb.git
cd atb
make build
./atb view run.atb/bundle.atb --profile atb.profile.policy_decision
```

See the [five-minute quickstart](./docs/getting-started/quickstart.md) for Python and TypeScript
SDK installation, capture paths, and the complete local review flow.

## What ATB records

```text
AI agent / application / framework
              │
              ▼
           CAPTURE
   ┌──────────┼──────────┐
   │          │          │
  SDK      intercept    import
   │          │          │
   └──────────┼──────────┘
              ▼
       canonical ATB event
              │
              ▼
       RFC 8785 canonicalise
              │
              ▼
         SHA-256 chain
              │
              ▼
          .atb bundle
              │
       ┌──────┼─────────────┐
       ▼      ▼             ▼
    verify  incident      view
              │
              ▼
        evidence pack
              │ optional
              ▼
           Mortise
        custody boundary
```

Optional Ed25519 or configured KMS signatures, RFC 3161 timestamp evidence,
and AES-256-GCM encryption add provenance, time evidence, and confidentiality.
They do not change the capture-completeness boundary.

## Incident forensics

`atb intercept` records routed provider traffic, tool calls, and failures. By
default it stores body digests and byte lengths rather than raw prompts or
completions, and strips credential headers. Raw capture is explicit and emits
a sensitive-content warning.

```bash
atb incident list --bundle <bundle-file>
atb incident report --bundle <bundle-file> --session <session-id>
atb incident export --bundle <bundle-file> --session <session-id> --out incident-evidence.zip
```

Findings such as `tool_without_approval` mean that no matching approval is
present in the recorded evidence before the located tool call. They do not
prove that no approval existed anywhere outside the capture boundary. Follow
the reproducible [incident-forensics walkthrough](./docs/investigate/incidents.md).

Run the complete offline workflow from a source checkout:

```bash
make demo-incident
```

It creates the same incident bundle twice and checks byte-for-byte determinism,
verifies the intact evidence, reconstructs the session, asserts the expected
finding, and proves content mutation, record reordering, and record removal all
fail verification.

## Independent implementations

Go, Python, and TypeScript independently implement the ATB bundle semantics.
They share schemas, event vocabulary, and deterministic golden vectors; the
Python and TypeScript SDKs do not call a hidden Go service.

| Language | Package |
| --- | --- |
| Go | [`pkg/api/v1`](./pkg/api/v1) |
| Python | [`sdk/python`](./sdk/python) |
| TypeScript | [`sdk/typescript`](./sdk/typescript) |

`make test-golden` blocks release unless all three implementations agree on
canonical bytes and hashes.

## Tenon, ATB, and Mortise

```text
Tenon
├── ATB      open-source, local-first evidence core
└── Mortise  optional commercial custody and organisational layer
```

Tenon is the umbrella product identity, not an ATB runtime component or hosted
dependency. ATB remains independently useful for capture, verification,
incident analysis, local review, profile evaluation, and evidence-pack
generation.

Mortise is an optional integration boundary for custody of record, WORM
retention, signed receipts, organisational evidence management, and enterprise
controls. ATB's optional Mortise client does not make Mortise part of the ATB
runtime. Operator-controlled S3/Object Lock export is also supported; ATB
records a successful retention request but does not prove continuing
storage-side enforcement.

## Evidence profiles and packs

ATB obligation profiles are declarative evidence models evaluated against the
records in a bundle. The Completeness Assurance Score (CAS) estimates
profile-scoped evidence coverage; it is not proof of universal capture and not
an external audit opinion.

`atb compliance pack` is the CLI name for a deterministic, mapping-oriented
evidence archive. It packages verification artefacts; it does not certify legal
or regulatory compliance.

## Documentation

| Read | Purpose |
| --- | --- |
| [Documentation hub](./docs/README.md) | Canonical map of current documentation |
| [Architecture](./docs/concepts/architecture.md) | Components, implementations, and trust boundaries |
| [Security model](./docs/concepts/trust-model.md) | Guarantees, threats, and explicit limits |
| [Incident forensics](./docs/investigate/incidents.md) | Reproducible post-incident workflow |
| [Profiles and CAS](./docs/evidence/profiles.md) | Evidence obligations and profile-scoped scoring |
| [Bundle specification](./docs/specification/bundle-v1.md) | Frozen format, hashing, and canonicalisation contract |
| [Glossary](./docs/concepts/glossary.md) | Canonical product and evidence terminology |
| [Contributing](./CONTRIBUTING.md) | Local development and review gates |
| [Versioning](./VERSIONING.md) | SemVer and compatibility rules |

## Licence

[MIT](./LICENSE)
