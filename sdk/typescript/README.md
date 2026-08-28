# @pcguest/atb-sdk

The ATB TypeScript SDK creates and reads ATB bundles, verifies local hash
chains, and lets Node.js callers iterate canonical bundle events.

It targets Node.js 18 or newer because file-backed bundles, encryption, and the
optional loopback Agent transport use Node APIs. It is not a browser SDK.

## Installation

```bash
npm install @pcguest/atb-sdk
```

## Quickstart

```ts
import {
  AI_REQUEST_RECEIVED_EVENT_TYPE,
  Bundle,
} from "@pcguest/atb-sdk";

const bundle = new Bundle();
bundle.append(AI_REQUEST_RECEIVED_EVENT_TYPE, {
  request_id: "req-001",
  actor_id_hash: "sha256:actor-001",
  purpose_tag: "rag_answer",
});
bundle.save("run.atb/bundle.atb");

const received = Bundle.load("run.atb/bundle.atb");
const result = received.verify();

for (const record of received.records) {
  console.log(record.event.type);
}

console.log(result.signatures);
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

`Bundle.verify()` checks local hash-chain integrity and supported signature
records. To evaluate ATB obligation profiles, incident findings, anchors, and
the full `verify.report.v1` contract, hand the same file to the Go CLI:

```bash
atb verify --bundle run.atb/bundle.atb --profile atb.profile.rag_answer
```

`Bundle.load()` rejects records above 16 MiB and defaults to 512 MiB and one
million records per bundle. Pass `{ maxBytes, maxRecords }` as the second
argument to choose lower limits for untrusted inputs.

## Optional reviewer identity evidence

Oversight gates accept advanced, digest-only identity context:

```ts
import type { ActionGateInput } from "@pcguest/atb-sdk";

const action: ActionGateInput = {
  actionType: "deploy",
  targetResourceId: "production/api",
  intendedEffect: "release candidate",
  actionParameters: { version: "2026.06" },
  identityEvidence: {
    identityProvider: "https://idp.example",
    subject: "reviewer@example",
    authContext: "mfa",
    assertionType: "jwt",
    assertionDigest: "sha256:...",
  },
};
```

Retain and verify the original assertion in your IdP/JWKS/PKI workflow. ATB
hash-chains the supplied digest but does not authenticate the subject.

## Optional Mortise custody

Mortise is a separate service; the SDK remains fully local without it:

```ts
import { readFile } from "node:fs/promises";
import { MortiseClient } from "@pcguest/atb-sdk";

const client = new MortiseClient("https://mortise.example", {
  token: process.env.ATB_MORTISE_TOKEN,
});
const bundle = new Uint8Array(await readFile("run.atb/bundle.atb"));
const verification = await client.verifyBundle(bundle); // does not persist
const receipt = await client.ingestBundle(bundle);       // custody + receipt
await client.verifyReceipt(receipt);
await client.receiptsByHash(receipt.bundle_hash);
```

The client rejects credential-bearing or non-HTTP(S) URLs, disables redirects,
caps responses at 1 MiB, and validates the frozen `custos.receipt.v1` wire
identifier.

## Vercel AI SDK integration

For Vercel AI SDK middleware guidance, see [docs/integrations/](../../docs/integrations/README.md).

## Licence

MIT.
