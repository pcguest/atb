# @pcguest/atb-sdk

The ATB TypeScript SDK reads ATB bundles, verifies local hash chains, and lets Node.js callers iterate canonical bundle events.

It also creates in-memory bundles, appends events, and saves NDJSON bundle files.

## Installation

```bash
npm install @pcguest/atb-sdk
```

## Quickstart

```ts
import { Bundle } from "@pcguest/atb-sdk";

const bundle = Bundle.load("run.atb/bundle.atb");
const result = bundle.verify();

for (const record of bundle.records) {
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

To verify a bundle against a profile, use the Go CLI: `atb verify --bundle <path> --profile <profile-id>`. The TypeScript SDK does not include a verifier.

## Vercel AI SDK integration

For Vercel AI SDK middleware guidance, see [docs/integrations/](../../docs/integrations/README.md).

## Licence

MIT.
