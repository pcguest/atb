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
import { Bundle } from "@pcguest/atb-sdk";

const bundle = new Bundle();

bundle.append("dev.session", {
  date: "2025-01-15",
  featuresBuilt: ["hash chaining", "CLI init"],
});

bundle.append("decision", {
  choice: "Go over Rust for CLI",
  reason: "Solo founder velocity",
});

bundle.save(); // Writes to run.atb/bundle.atb

// Later - reload and verify
const loaded = Bundle.load();
loaded.verify(); // Throws ATBVerificationError if tampered
console.log(`Verified ${loaded.length} records (including manifest) - chain intact.`);
```

`new Bundle()` starts with an `atb.bundle.manifest` record at `seq = 0`. Appended events start at `seq = 1`.

## Licence

MIT
