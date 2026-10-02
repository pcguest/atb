# HARNESS RECONCILIATION LEDGER

Date: 2026-09-23
Scope: LOCAL OpenCode development harness only. `.opencode/` is excluded from
git via `.git/info/exclude`. ATB product code must never depend on this harness.

## Starting state (verified, not assumed)

- Branch `feat/acquisition-continuity`, HEAD `d5bd611`, `origin/main` `ebcb97d`.
- Working tree: 11 modified product files + 3 untracked acquisition fixtures.
- `.opencode/` was entirely untracked and NOT git-excluded (commit hazard).
- OpenCode 1.18.32. Providers: `openai` (oauth), `google` (api),
  `opencode` (OpenCode Zen api), plus `opencode-go`.

## Inventory and classification

| Item | Class | Decision |
| --- | --- | --- |
| `.opencode/opencode.jsonc` | UPDATE | Rewritten. `agents` key was INVALID; now correct `agent`/`permission`/`mcp`. Single routing source. |
| `.opencode/permission.yaml` | SUPERSEDE / DELETE | NOT read by OpenCode (config uses `permission`). Replaced by the `permission` block in `opencode.jsonc`. |
| `.opencode/agent/atb-engineer.md` | UPDATE | Model removed from frontmatter; config is routing source. |
| `.opencode/agent/repo-explorer.md` | UPDATE | Same. |
| `.opencode/agent/implementer.md` | UPDATE | Same. |
| `.opencode/agent/reviewer.md` | UPDATE | Same; documents independent-family requirement. |
| `.opencode/agent/adversarial-reviewer.md` | ADD | New read-only security/trust reviewer. |
| `.opencode/agent/practitioner.md` | ADD | New read-only synthetic-practitioner agent. |
| `.opencode/astra/` (framework) | SUPERSEDE / DELETE | Abandoned, non-functional parallel router (`astra.sh` printed "would run"). No active reference outside itself (proven by grep). Concepts folded into routing ledger + `jev-decision-review` skill. |
| `.opencode/astra/state/*` | STALE / DELETE | Generated example envelopes/results, unreferenced. |
| `.opencode/skills/*` (7) | ADD | On-demand procedural context. |
| `.opencode/.gitignore`, `node_modules`, `package*.json` | LOCAL_ONLY / KEEP | Unchanged. |
| `.local/dev/state/*` (run_*.json, jev_learning_*.json) | TEMPORAL | Retained; superseded by a new consolidated continuation record. |
| `.local/dev/run.py` | STALE / SUPERSEDE | Points at the old single-review Jev harness; superseded by `jev_decision_plane.py`. Kept (local) but no longer the entry point. |
| `.devin/audit_code.*` | UPDATE / KEEP | Correct execution-status contract retained; extended by `jev_decision_plane.py`. |
| `.git/info/exclude` | UPDATE | Added `.opencode/` (local-only, never commit). |
| `~/.config/opencode/opencode.jsonc` | KEEP | Empty schema stub; no change needed. |

## Verification performed

- `opencode debug agent <name>` for all 6 agents → correct provider/model from config.
- `opencode debug skill` → all 7 project skills discovered at `.opencode/skills/`.
- `git check-ignore` → `.opencode/` now locally excluded.

## Context architecture (target)

- DURABLE: `AGENTS.md` + `docs/` canonical references.
- REUSABLE PROCEDURE: `.opencode/skills/*` (loaded on demand only).
- CURRENT: git state + current task.
- TEMPORAL: `.local/dev/state/` short-lived continuation.
- EXTERNAL: fetch current docs on demand (no historical transcripts in context).

## Not changed / deliberately avoided

- No orchestration framework introduced.
- No MCP servers enabled (`mcp: {}`).
- No ATB product file depends on the harness.

## Operational note (2026-09-23)

The running OpenCode server loads `.opencode/opencode.jsonc` at session start and
does NOT hot-reload it. A long-running session was observed holding a stale
routing (one agent resolving to a non-existent model), while a fresh
`opencode debug agent` read the current config correctly.

- After editing routing, restart the OpenCode session/server before relying on
  `@agent` delegation; verify with `opencode debug agent <name>`.
- For one-shot delegation on an explicit model without restarting, use
  `opencode run --model <provider/model> "<prompt>"` (fresh process, current
  config). `--agent <subagent>` falls back to the default agent, but the model
  override is honoured.
- Non-interactive `opencode run` auto-rejects permissions set to `ask`
  (`external_directory`), which aborts the run. Point bounded scratch work at an
  in-repo gitignored dir (e.g. `.tmp/`) instead of an external temp dir, or pass
  `--auto` (it does not override explicitly `deny`d commands).

## Operational note (2026-09-23, Prompt 94629-C)

Confirmed again during post-merge certification:

- The `task` tool in the running session did NOT honour on-disk routing:
  a `reviewer` delegation attempted the non-existent `opencode/mimo-v2.5-free`,
  and `adversarial-reviewer` was rejected as "not a valid agent type". The
  on-disk `opencode.jsonc` routes reviewer to `opencode-go/qwen3.7-plus` and
  defines `adversarial-reviewer` on `opencode-go/minimax-m3`.
- Recovery that worked: fresh `opencode run --model opencode-go/qwen3.7-plus`
  and `opencode run --model opencode-go/minimax-m3`, each a new process reading
  current config. Both models are present in `opencode models` output.
- `repo-explorer` via the `task` tool worked (read-only).
- The `-f`/`--dir` flags plus a prompt file in `.tmp/` gave bounded, reviewable
  runs; capturing stdout to a file was necessary because tool-call output is
  verbose and can scroll past a `tail`.

Rule reinforced: registered agents and runtime-callable agents are not
automatically equivalent. Verify the actual runtime route; if stale, use fresh
explicit-model runs and record the substitution honestly.

Certification outcome for this run: MORTISE_TRANSITION_GATE: GO.

