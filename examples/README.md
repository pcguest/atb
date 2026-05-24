# ATB examples

Use this directory for public, product-facing examples only.

## Quickstart

A minimal privileged_tool_action workflow that produces a verifiable bundle with a CAS score. Run with `bash run.sh`.

Path: [examples/quickstart/](./quickstart/)

## SDK examples

- [Python LangChain example](./python/langchain_bot.py)
- [TypeScript Vercel AI example](./typescript/vercel-chat-bot.ts)

## Bundle examples

- [Project bootstrap bundle](./bundles/project-bootstrap.atb) — valid fixture for `atb verify`
- [Tampered bootstrap bundle](./bundles/project-bootstrap-tampered.atb) — hash-chain failure demo; see [tamper demo guide](../docs/guides/tamper-demo.md)

The bundle examples are small verified ATB bundles that demonstrate the on-disk format without exposing internal project logs or operational notes.
