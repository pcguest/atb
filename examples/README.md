# ATB examples

Use this directory for public, product-facing examples only.

## Quickstart

A minimal privileged_tool_action workflow that produces a verifiable bundle with a CAS score. Run with `bash run.sh`.

Path: [examples/quickstart/](./quickstart/)

## SDK examples

- [Python LangChain example](./python/langchain_bot.py)
- [Python LangGraph example](./python/langgraph_demo.py)
- [TypeScript Vercel AI example](./typescript/vercel-chat-bot.ts)

## Flagship incident workflow

Run `make demo-incident` for the deterministic, application-boundary incident
workflow. It verifies intact evidence, reconstructs the session, asserts the
`tool_without_approval` finding, and confirms tamper detection. See
[examples/incident-demo/](./incident-demo/).

## Quickstart variants

- [README quickstart](../README.md#quickstart) — `atb.profile.policy_decision` (support triage deny)
- [examples/quickstart/run.sh](./quickstart/run.sh) — `atb.profile.privileged_tool_action` (six-event privileged action)

## Generated profile matrix

- Generated profile fixtures — pass/fail bundle pairs for each of the six
  built-in obligation profiles, created by
  [`generate_profile_fixtures.go`](../scripts/generate_profile_fixtures.go)

These bundles are generated test inputs, not a second demo. The flagship
incident workflow above is the sole end-to-end product narrative.

The `.atb` files are gitignored and absent from a fresh clone. Regenerate the
profile matrix with:

```bash
make goldens
```

The generator asserts the expected profile outcome for every bundle.
