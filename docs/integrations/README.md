# Integrations

ATB integrates with common AI stacks and operator storage. Each integration has
an explicit capture boundary described in the
[trust model](../concepts/trust-model.md).

| Integration | Doc |
| --- | --- |
| LangChain (Python / TS) | [langchain.md](./langchain.md) |
| Vercel AI SDK | [vercel-ai.md](./vercel-ai.md) |
| MCP | [mcp.md](./mcp.md) |
| Chatlog import | [chatlog-import.md](./chatlog-import.md) |
| WORM / S3 push | [bundle push contract](../specification/bundle-push.md) |
| Optional Mortise custody | [mortise.md](./mortise.md) |
| SIEM / GRC export | [siem-grc.md](./siem-grc.md) |

Custody, receipts, organisational controls, and transparency/witness
operations live in optional products such as Mortise, not in the ATB runtime.
