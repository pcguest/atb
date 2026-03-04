# @pcguest/atb-sdk — ATB TypeScript SDK

The official TypeScript/JavaScript SDK for [ATB (Agent Trace Bundle)](https://github.com/pcguest/atb).

## Installation

```bash
npm install @pcguest/atb-sdk
# or
pnpm add @pcguest/atb-sdk
```

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

// Later — reload and verify
const loaded = Bundle.load();
loaded.verify(); // Throws ATBVerificationError if tampered
console.log(`Verified ${loaded.length} events — chain intact.`);
```

## License

MIT
