# TypeScript SDK Quickstart

This example demonstrates a minimal bundle creation workflow using the TypeScript SDK that produces a verifiable ATB bundle.

## Prerequisites

- Node.js 18 or newer
- TypeScript execution (tsx or similar)

## Installation

The TypeScript SDK is available on npm. Use npx to run the example without installation:

```bash
npx -y tsx quickstart.ts
```

Or install globally:

```bash
npm install -g tsx @pcguest/atb-sdk
```

## How to run

Using tsx (recommended):

```bash
npm install -g tsx
tsx quickstart.ts
```

Or using npx:

```bash
npx tsx quickstart.ts
```

## What a passing run looks like

```bash
Creating ATB bundle...
✓ Appended event #1 [ai.request.received]
✓ Appended event #2 [ai.model.invoked]

Bundle saved to: run.atb/bundle.atb

Verification complete
Signatures found: 0
Records in bundle: 3

For full profile evaluation, run:
  atb verify --bundle run.atb/bundle.atb --profile atb.profile.rag_answer
```

## What to do next

- [Vercel AI SDK integration](./vercel-chat-bot.ts)
- [Integrations](../../docs/integrations/README.md)
- [TypeScript SDK README](../../sdk/typescript/README.md)
