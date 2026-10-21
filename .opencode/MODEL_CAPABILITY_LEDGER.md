# MODEL CAPABILITY LEDGER (local)

Discovered via `opencode models` (OpenCode 1.18.32) on 2026-09-23; re-verified
against `opencode models` on OpenCode 1.18.34 on 2026-10-21. No credentials are
recorded here. This ledger records owner-selected routing and observed inventory;
it is not a benchmark ranking or a price quote.

## Current routed models

Only configured agent and root-default routes are listed; the separate `small_model` route (`opencode/ling-3.0-flash-fin-free`) is documented in `.opencode/opencode.jsonc`.
are not routing decisions and are not implied fallbacks.

| Role(s) | Model ID | Provider | Family | Route class | Fallback |
| --- | --- | --- | --- | --- | --- |
| Primary orchestrator; adversarial reviewer | `openai/gpt-6.1-sol-fast` | openai (OAuth) | OpenAI GPT | OAuth route | Global fallback only; no per-role fallback configured |
| Repo explorer; bounded implementer | `opencode-go/deepseek-v4.1-flash` | opencode-go | DeepSeek | Go allowance | Global fallback only; no per-role fallback configured |
| Normal reviewer | `opencode-go/glm-5.3-flash` | opencode-go | GLM | Go allowance | Global fallback only; no per-role fallback configured |
| Synthetic practitioner A | `opencode-go/mimo-v2.5` | opencode-go | MiMo | Go allowance | Global fallback only; no per-role fallback configured |
| Global fallback | `opencode/nemotron-3-ultra-free` | opencode (Zen) | NVIDIA | Free | Configured global fallback |

## Routing rationale and historical notes

- The owner selected `openai/gpt-6.1-sol-fast` for the primary and adversarial
  roles. This is not an escalation-only route, and no cost ranking is asserted.
- The owner selected `opencode-go/deepseek-v4.1-flash` for repo exploration and
  bounded implementation. The implementer's scope is an explicit task packet:
  smallest coherent patch plus focused validation, not architecture selection or
  self-certification. This is a routing choice, not a benchmark result or a claim
  that DeepSeek is superior to another model. Kimi K2.7 Code remains available in
  inventory but is not routed here.
- The normal reviewer uses a different model family from the implementer. The
  adversarial reviewer does not: it shares the primary's OpenAI model. See the
  diversity limits below.
- The former `opencode-go/deepseek-v4-flash` ID is absent from the current
  inventory; the current DeepSeek ID is `opencode-go/deepseek-v4.1-flash`. Other
  inventory entries are not fallbacks just because they are listed by
  `opencode models`.
- Routing lives ONLY in `.opencode/opencode.jsonc` (`agent.<name>.model`). No ATB
  product code contains model names.

## Final routing (2026-10-21)

| Role | Agent | Model |
| --- | --- | --- |
| Primary orchestrator | atb-engineer | openai/gpt-6.1-sol-fast |
| Repo explorer | repo-explorer | opencode-go/deepseek-v4.1-flash |
| Bounded implementer | implementer | opencode-go/deepseek-v4.1-flash |
| Independent reviewer | reviewer | opencode-go/glm-5.3-flash |
| Adversarial reviewer | adversarial-reviewer | openai/gpt-6.1-sol-fast |
| Synthetic practitioner A | practitioner | opencode-go/mimo-v2.5 |
| Synthetic practitioner B | (task-scoped, not a configured agent) | opencode-go/deepseek-v4.1-flash |
| Free overflow | global fallback | opencode/nemotron-3-ultra-free |

## Model-family diversity and limits

- Different families: implementer (DeepSeek) and normal reviewer (GLM).
- The adversarial reviewer uses the same OpenAI GPT-6.1 Sol Fast model as the
  primary. It is not an independent model family.
- Its independence is procedural only: a fresh read-only context, hostile-input
  review role, and no edit/bash/task/external-directory permissions.
- This is an accepted owner-selected trade-off. It does not provide independent
  model diversity from the primary and does not eliminate correlated model
  errors. No immunity to correlated mistakes is claimed.

`openai` is the connected OAuth provider. `console-openai` is a separate
console/API provider and is not used for routine routing.

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
