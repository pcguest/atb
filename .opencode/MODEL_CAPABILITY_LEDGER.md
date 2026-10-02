# MODEL CAPABILITY LEDGER (local)

Discovered via `opencode models` (OpenCode 1.18.32) on 2026-09-23. No credentials
are recorded here. Cost class is a routing heuristic, not a price quote.

## Available families relevant to ATB development

| Model ID | Provider | Family | Cost class | Best role | Poor-fit roles | Fallback |
| --- | --- | --- | --- | --- | --- | --- |
| `opencode-go/gpt-5.6-luna` | opencode-go | OpenAI GPT | go allowance | Orchestration, architecture, final integration | Bulk search, mechanical edits | `opencode/gpt-5.6-luna` |
| `opencode-go/kimi-k2.7-code` | opencode-go | Moonshot Kimi | go allowance | Bounded code implementation | Orchestration, security review | `opencode-go/kimi-k2.6` |
| `opencode-go/deepseek-v4-flash` | opencode-go | DeepSeek | go allowance | Repo exploration, high-volume routine work | Final semantic review | `opencode/deepseek-v4-flash` |
| `opencode-go/mimo-v2.5` | opencode-go | MiMo | go allowance | High-volume routine worker, practitioner A | Security review | `opencode-go/mimo-v2.6-flash` |
| `opencode-go/qwen3.7-plus` | opencode-go | Qwen | go allowance | Independent reviewer | Hostile-input fuzzing | `opencode-go/qwen3.7-max` |
| `opencode-go/minimax-m3` | opencode-go | MiniMax | go allowance | Adversarial / security review | Routine docs | `opencode-go/minimax-m2.7` |
| `openai/gpt-5.6-sol` | openai | OpenAI GPT | expensive escalation | Consequential architecture ambiguity, hard security/debug | Routine work | `openai/gpt-5.6-terra` |
| `opencode/nemotron-3-ultra-free` | opencode (Zen) | NVIDIA | free | Free overflow / fallback | Primary reasoning | `opencode/ling-3.0-flash-fin-free` |
| `opencode/ling-3.0-flash-fin-free` | opencode (Zen) | Ling | free | Small/title/summarise tasks | Implementation | `opencode/mimo-v2.6-flash-free` |
| `opencode/mimo-v2.6-flash-free` | opencode (Zen) | MiMo | free | Free overflow | Security review | `opencode/nemotron-3-ultra-free` |
| `opencode-go/qwen3.7-max` | opencode-go | Qwen | go allowance | Escalated review | Routine work | `opencode-go/qwen3.7-plus` |

## Notes / deviations from the starting hypothesis

- `MiMo-V2.5` and `Qwen3.7 Plus` and `MiniMax M3` exist only under the
  `opencode-go/` provider (not `opencode/`), so routing uses `opencode-go/*`.
- Kimi K3 is intentionally NOT the default; Kimi K2.7 Code is the bounded implementer.
- The expensive OpenAI model is an escalation class only, not a routine worker.
- Routing lives ONLY in `.opencode/opencode.jsonc` (`agent.<name>.model`).
  No ATB product code contains model names.

## Final routing

| Role | Agent | Model |
| --- | --- | --- |
| Primary orchestrator | atb-engineer | opencode-go/gpt-5.6-luna |
| Repo explorer | repo-explorer | opencode-go/deepseek-v4-flash |
| Bounded implementer | implementer | opencode-go/kimi-k2.7-code |
| Independent reviewer | reviewer | opencode-go/qwen3.7-plus |
| Adversarial reviewer | adversarial-reviewer | opencode-go/minimax-m3 |
| Synthetic practitioner A | practitioner | opencode-go/mimo-v2.5 |
| Synthetic practitioner B | (task-scoped) | opencode-go/deepseek-v4-flash |
| Free overflow | global fallback | opencode/nemotron-3-ultra-free |
| Escalation | (manual) | openai/gpt-5.6-sol |

Heterogeneity guarantee: implementer (Kimi) ≠ reviewer (Qwen) ≠ adversarial (MiniMax).

## MCP / plugin ledger

| Item | Class | Decision |
| --- | --- | --- |
| Native read/edit/bash/grep/glob/task/webfetch/websearch | ESSENTIAL | Enabled |
| MCP servers | none configured | `mcp: {}` — prefer native tools |
| Context7 | ON_DEMAND (candidate) | Not enabled; add only if a measured doc gap appears |
| GitHub MCP | ON_DEMAND | Disabled; remote work not started |

## Skill ledger

KEEP/ADD: `atb-trust-boundary`, `canonical-parity`, `acquisition-evidence`,
`investigation-ux`, `security-review`, `release-validation`, `jev-decision-review`.
SUPERSEDE: none required (no prior project skills existed). Global `golang-*`
and `supabase*` skills remain as unrelated on-demand context.
