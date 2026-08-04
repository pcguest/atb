# ATB Python SDK

The ATB Python SDK writes tamper-evident audit bundles in the same format as the Go CLI; bundles written by the SDK are verifiable with `atb verify`.

## Installation

```bash
pip install atb-sdk
```

For a reproducible contributor checkout (CI uses the same pin file):

```bash
python -m pip install -r requirements-dev.txt
python -m pip install -e .[dev] --no-deps
```

Release tooling is intentionally separate in `requirements-release.txt`; it is
not required for normal SDK development. CI-only Bandit dependencies are kept
in `requirements-security.txt`.

## Quick example

```python
from atb import Bundle
from atb.event_types import (
    AI_MODEL_INVOKED_EVENT_TYPE,
    AI_REQUEST_RECEIVED_EVENT_TYPE,
)

bundle = Bundle()

bundle.append(
    AI_REQUEST_RECEIVED_EVENT_TYPE,
    {
        "request_id": "req-001",
        "actor_id_hash": "hash-user-01",
        "purpose_tag": "quickstart_demo",
    },
)

bundle.append(
    AI_MODEL_INVOKED_EVENT_TYPE,
    {
        "model_provider": "openai",
        "model_id": "gpt-4o-mini",
        "model_parameters_digest": "sha256-params-abc",
        "prompt_digest": "sha256-prompt-def",
    },
)

path = bundle.save("run.atb/bundle.atb")
print(path)
```

## Supported event types

| Event type constant name | Event type string |
| --- | --- |
| `BUNDLE_MANIFEST_EVENT_TYPE` | `atb.bundle.manifest` |
| `BUNDLE_ANCHOR_EVENT_TYPE` | `atb.bundle.anchor` |
| `BUNDLE_SIGNATURE_EVENT_TYPE` | `atb.bundle.signature` |
| `AI_REQUEST_RECEIVED_EVENT_TYPE` | `ai.request.received` |
| `AI_RESPONSE_SENT_EVENT_TYPE` | `ai.response.sent` |
| `AI_POLICY_DECISION_EVENT_TYPE` | `ai.policy.decision` |
| `AI_RETRIEVAL_EXECUTED_EVENT_TYPE` | `ai.retrieval.executed` |
| `AI_MODEL_INVOKED_EVENT_TYPE` | `ai.model.invoked` |
| `AI_MODEL_OUTPUT_EVENT_TYPE` | `ai.model.output` |
| `AI_ACTION_PRECOMMIT_EVENT_TYPE` | `ai.action.precommit` |
| `AI_ACTION_EXECUTED_EVENT_TYPE` | `ai.action.executed` |
| `AI_ACTION_COMMITTED_EVENT_TYPE` | `ai.action.committed` |
| `AI_HUMAN_APPROVAL_EVENT_TYPE` | `ai.human.approval` |
| `AI_JOB_SCHEDULED_EVENT_TYPE` | `ai.job.scheduled` |
| `AI_JOB_STARTED_EVENT_TYPE` | `ai.job.started` |
| `AI_JOB_STEP_EVENT_TYPE` | `ai.job.step` |
| `AI_JOB_COMPLETED_EVENT_TYPE` | `ai.job.completed` |
| `DATA_EXPORT_PRECOMMIT_EVENT_TYPE` | `data.export.precommit` |
| `DATA_EXPORT_EXECUTED_EVENT_TYPE` | `data.export.executed` |

## Profile support

To verify a bundle against a profile, use the Go CLI: `atb verify --bundle <path> --profile <profile-id>`. The Python SDK does not include a verifier.

## Optional reviewer identity evidence

Oversight gates accept advanced, digest-only identity context:

```python
from atb import ActionGateInput, IdentityEvidence

action = ActionGateInput(
    "deploy",
    "production/api",
    "release candidate",
    {"version": "2026.06"},
    identity_evidence=IdentityEvidence(
        identity_provider="https://idp.example",
        subject="reviewer@example",
        auth_context="mfa",
        assertion_type="jwt",
        assertion_digest="sha256:...",
    ),
)
```

Retain and verify the original assertion in your IdP/JWKS/PKI workflow. ATB
hash-chains the supplied digest but does not authenticate the subject.

See [`examples/python/agent_incident_demo.py`](../../examples/python/agent_incident_demo.py)
for an offline incident-forensics flow.

## Optional Mortise custody

Mortise is a separate service; the SDK remains fully local without it. To lodge
or remotely verify a completed bundle:

```python
import os
from pathlib import Path

from atb import MortiseClient

client = MortiseClient(
    "https://mortise.example",
    token=os.environ["ATB_MORTISE_TOKEN"],
)
bundle = Path("run.atb/bundle.atb").read_bytes()
verification = client.verify_bundle(bundle)  # does not persist
receipt = client.ingest_bundle(bundle)       # WORM custody + signed receipt
assert client.verify_receipt(receipt)["verified"] is True
matches = client.receipts_by_hash(receipt["bundle_hash"])
```

The client rejects credential-bearing or non-HTTP(S) URLs, does not follow
redirects, caps responses at 1 MiB, and validates the frozen
`custos.receipt.v1` wire identifier.

## LangChain integration

For LangChain middleware guidance, see [docs/integrations/](../../docs/integrations/README.md).

## Licence

MIT.
