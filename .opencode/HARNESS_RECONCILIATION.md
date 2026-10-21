# HARNESS RECONCILIATION LEDGER

Date: 2026-09-23
Scope: LOCAL OpenCode development harness only. `.opencode/` is committed in
this branch as development harness configuration. ATB product code must never depend on this harness.

## Starting state (historical snapshot, 2026-09-23)

- At capture time on branch `feat/acquisition-continuity`, local HEAD was
  `d5bd611` and `origin/main` was `ebcb97d` (current repository HEAD: `fefba25`).
- Working tree: 11 modified product files + 3 untracked acquisition fixtures.
- `.opencode/` is committed in this branch as development harness configuration.
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
| `.git/info/exclude` | REVERT / NOT NEEDED | `.local/` remains excluded; `.opencode/` is now tracked as committed development tooling. |
| `~/.config/opencode/opencode.jsonc` | KEEP | Empty schema stub; no change needed. |

## Verification performed

- `opencode debug agent <name>` for all 6 agents → correct provider/model from config.
- `opencode debug skill` → all 7 project skills discovered at `.opencode/skills/`.
- `git ls-files .opencode/` → all harness files (config, agents, skills, this ledger) are tracked in this branch as committed development tooling. `.git/info/exclude` continues to exclude `.local/` only; `git check-ignore` reports already-tracked paths only for the untracked `.local/` tree, never for committed `.opencode/` files.

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
  current config. Both models are present in `opencode models` output. These
  one-shot runs use the default agent and inherit the global `edit`/`bash`
  permissions, not the read-only reviewer agents, so treat their output as
  advisory evidence only, not as a substitute for the configured reviewers.
- `repo-explorer` via the `task` tool worked (read-only).
- The `-f`/`--dir` flags plus a prompt file in `.tmp/` gave bounded, reviewable
  runs; capturing stdout to a file was necessary because tool-call output is
  verbose and can scroll past a `tail`.

Rule reinforced: registered agents and runtime-callable agents are not
automatically equivalent. Verify the actual runtime route; if stale, use fresh
explicit-model runs and record the substitution honestly.

Certification outcome for this run: ATB_MORTISE_INTEROP_GATE: GO.

Historical scope: this result belongs to the 2026-09-23 scenario and versions
recorded above (OpenCode 1.18.32; reviewer `opencode-go/qwen3.7-plus`,
implementer `opencode-go/kimi-k2.7-code`, adversarial reviewer
`opencode-go/minimax-m3`). It is preserved as historical evidence for that
scenario. It does not certify the 2026-10-21 routing or current ATB–Mortise
interoperability, and no new interoperability run was performed for this harness
change.

## Routing re-verification (2026-10-21, OpenCode 1.18.34)

### Configuration validation (on-disk)

This section validates the on-disk configuration only. It is not proof of what a
long-running server loaded or of live delegation.

Verified against `opencode models`, `opencode providers list` (names only; no
credentials read or recorded), `opencode debug config` and `opencode agent list`.

- Config key is `agent` (singular). No root `agents` key remains.
- `.opencode/permission.yaml` is absent; its permissions live in the
  `permission` block of `opencode.jsonc`. `.opencode/astra/` is absent and
  unreferenced.
- No model routing remains in agent frontmatter; `opencode.jsonc` is the single
  routing source. All six roles are native agents (`atb-engineer`,
  `repo-explorer`, `implementer`, `reviewer`, `adversarial-reviewer`,
  `practitioner`); no parallel agent layout.
- Current routes: atb-engineer `openai/gpt-6.1-sol-fast`; repo-explorer and
  implementer `opencode-go/deepseek-v4.1-flash`; reviewer
  `opencode-go/glm-5.3-flash`; adversarial-reviewer `openai/gpt-6.1-sol-fast`;
  practitioner `opencode-go/mimo-v2.5`.
- `openai` is the connected OAuth provider. The primary is not routed through
  `opencode-go`.
- Reviewers are read-only (`edit`/`bash`/`task` denied).
- `subagent_depth` is 1, but it limits delegation depth; it is not a writer lock.
  Single-writer discipline is an instructed behavioural rule, not a technical
  serialization mechanism. While an implementer task is active, avoid edits or
  tests against files, generated artefacts, or shared mutable resources that task
  is modifying. Test execution can touch shared caches/build outputs, so do not
  assume a nominally unrelated test is side-effect-free. Run heavy validation
  serially after writers stop. No locking framework is introduced.

### Runtime evidence (Phase 3B)

The Phase 3B checkpoint and local session records separate actual runtime use from
the on-disk validation above:

- The original interactive primary remained
  `opencode-go/deepseek-v4.1-flash` (`ses_ea4649ba7ffe7Hed0QM6wmAt0m`) and
  predates the repair. It did not observe the new OpenAI primary route.
- Fresh final review orchestration used `openai/gpt-6.1-sol-fast`
  (`ses_ea43e9fcbffe6hx1vloO6oHvP6`) after one transient 429 retry.
- Normal child route verified: `reviewer` / `opencode-go/glm-5.3-flash`
  (`ses_ea43e9203ffeQbrBbJ2mAehXKF`).
- Adversarial child route verified: `adversarial-reviewer` /
  `openai/gpt-6.1-sol-fast` (`ses_ea4380e11ffefiHlNzI6Z0MGsp`).
- Targeted evidence re-review used `reviewer` / `opencode-go/glm-5.3-flash`
  (`ses_ea42bf214ffe30mAJsXYVR3IPy`).
- Roles not invoked in Phase 3B: `repo-explorer`, `implementer`, `practitioner`.
  Runtime coverage for those routes remains unverified beyond config validation.
- No server restart was performed or observed. Live `task` delegation from the
  original interactive primary was not observed and is not runtime evidence for
  the new routes. The fresh-process child routes above are recorded in local
  session metadata, not inferred from the configuration.

### Residual coverage

Runtime evidence covers the fresh primary, normal reviewer, and adversarial
reviewer. It does not cover `repo-explorer`, `implementer`, or `practitioner`.
Configuration parsing, model inventory, agent registration, and permission
resolution are verified; live provider behaviour beyond the recorded sessions is
unverified.

