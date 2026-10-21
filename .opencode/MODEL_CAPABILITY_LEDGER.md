# MODEL CAPABILITY LEDGER (local)

Discovered via `opencode models` (OpenCode 1.18.32) on 2026-09-23; re-verified
against `opencode models` on OpenCode 1.18.34 on 2026-10-21. No credentials are
recorded here. This ledger records owner-selected routing and observed inventory;
it is not a benchmark ranking or a price quote.

## Agent routing (explicit assignments)

These are explicit `agent.<name>.model` assignments. None has a configured
per-role failover; an unavailable explicit assignment is not replaced
automatically by the root default or another inventory model.

| Role(s) | Model ID | Provider | Family | Route class | Per-role failover |
| --- | --- | --- | --- | --- | --- |
| Primary orchestrator; adversarial reviewer | `openai/gpt-6.1-sol-fast` | openai (OAuth) | OpenAI GPT | OAuth route | None configured |
| Repo explorer; bounded implementer | `opencode-go/deepseek-v4.1-flash` | opencode-go | DeepSeek | Go allowance | None configured |
| Normal reviewer | `opencode-go/glm-5.3-flash` | opencode-go | GLM | Go allowance | None configured |
| Synthetic practitioner A | `opencode-go/mimo-v2.5` | opencode-go | MiMo | Go allowance | None configured |

## Root default and auxiliary small model

These settings are not agent routes and are not failover mechanisms.

| Setting | Model ID | Provider | Meaning |
| --- | --- | --- | --- |
| Root default | `opencode/nemotron-3-ultra-free` | opencode (Zen) | Root `model` default used when no explicit agent model applies. It is not an automatic fallback/failover, and no recovery behaviour is asserted. |
| Auxiliary small model | `opencode/mimo-v2.6-flash-free` | opencode (Zen) | Auxiliary `small_model` setting. Replaces the unavailable `opencode/ling-3.0-flash-fin-free` with an active, zero-cost, text-capable model (200k context / 32k output) from the already connected `opencode` provider. It is not an agent route, not the root default, and not a failover. |

Selection note: the previous `small_model` ID is absent from refreshed inventory.
The replacement uses the existing `opencode` provider rather than adding a
provider; no recovery or failover behaviour is implied.

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
  inventory entries are not failovers just because they are listed by
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

## Model-family diversity and limits

Both the normal reviewer (GLM) and adversarial reviewer (OpenAI) differ from the
implementer (DeepSeek). The adversarial reviewer shares the primary's OpenAI
model; its separation from the primary is procedural, not model-family diversity.

Its independence is procedural only: a fresh read-only context, hostile-input
review role, and no edit/bash/task/external-directory permissions. This is an
accepted owner-selected trade-off. It does not eliminate correlated model errors;
no immunity to correlated mistakes is claimed.

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
