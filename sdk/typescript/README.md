# @pcguest/atb-sdk - ATB TypeScript SDK

The official TypeScript/JavaScript SDK for [ATB (Agent Trace Bundle)](https://github.com/pcguest/atb).

## Installation

```bash
npm install @pcguest/atb-sdk
# or
pnpm add @pcguest/atb-sdk
```

Use this package when you need to write or verify bundles from TypeScript or JavaScript code. The Go CLI remains the authoritative CLI path:

```bash
go install github.com/pcguest/atb/cmd/atb@latest
```

The package does not include a standalone ATB CLI. The installed `atb` command is a compatibility stub that prints Go CLI install guidance and will be removed in a future major release.

## Quick Start

```typescript
import {
  AI_MODEL_INVOKED_EVENT_TYPE,
  AI_MODEL_OUTPUT_EVENT_TYPE,
  AI_REQUEST_RECEIVED_EVENT_TYPE,
  Bundle,
} from "@pcguest/atb-sdk";

const bundle = new Bundle();

bundle.append(AI_REQUEST_RECEIVED_EVENT_TYPE, {
  request_id: "req-001",
  actor_id_hash: "sha256-actor-abc",
  purpose_tag: "rag_answer",
});
bundle.append(AI_MODEL_INVOKED_EVENT_TYPE, {
  model_provider: "openai",
  model_id: "gpt-4o",
  model_parameters_digest: "sha256-params-def",
  prompt_digest: "sha256-prompt-ghi",
});
bundle.append(AI_MODEL_OUTPUT_EVENT_TYPE, {
  output_digest: "sha256-output-jkl",
  output_format: "text/plain",
});

bundle.save();

const loaded = Bundle.load();
loaded.verify();
console.log(`Verified ${loaded.length} records (including manifest).`);
```

`new Bundle()` starts with an `atb.bundle.manifest` record at `seq = 0`. Appended events start at `seq = 1`.

## Licence

MIT
