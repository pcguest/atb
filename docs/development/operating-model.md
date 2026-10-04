# ATB development operating model

This is the authoritative description of how development on ATB (and its
downstream Mortise/Tenon work) uses OpenCode, subagents, skills, harnesses, and
continuation state. It exists so a future session can reproduce the process from
the repository instead of reconstructing it from chat history.

It is development tooling. **No ATB product code depends on it.**

## 1. Layers of context

| Layer | Lives in | Committed? | Purpose |
| --- | --- | --- | --- |
| Durable instruction | `AGENTS.md`, `docs/` | yes | product identity, invariants, canonical references |
| Reusable procedure | `.opencode/skills/` | yes | on-demand workflows (trust boundary, canonical parity, acquisition, UX, security, release) |
| Role contract | `.opencode/agent/` | yes | bounded responsibilities and output formats per agent |
| Routing | `.opencode/opencode.jsonc` | yes (no secrets) | model routing and least-privilege permissions; single source of truth |
| Executable validation | `harnesses/` | yes | deterministic product harnesses (end-user, interoperability, security, release) |
| Run state | `.local/dev/` | no (ignored via `.gitignore`) | continuation records, ledgers, ephemeral reports |
| Raw source material | `.local/` or scratch | no (ignored via `.gitignore`) | chat exports / fixtures under review; never committed |

Rule of thumb: **architecture invariant → durable instruction; repeated
procedure → skill; temporary state → local continuation record; changing
external fact → retrieve; secret → never durable context.**

## 2. Agent roles

Model routing lives only in `.opencode/opencode.jsonc`. Roles:

| Role | Agent file | Independence requirement |
| --- | --- | --- |
| Orchestrator | `agent/atb-engineer.md` | owns sequencing, gates, final synthesis |
| Repo explorer | `agent/repo-explorer.md` | read-only archaeology |
| Implementer | `agent/implementer.md` | bounded writes only |
| Correctness reviewer | `agent/reviewer.md` | different model family from implementer |
| Adversarial reviewer | `agent/adversarial-reviewer.md` | hostile-input / trust-boundary |
| Practitioner | `agent/practitioner.md` | unfamiliar-user simulation, advisory only |

Hard rule: `implementer != reviewer != adversarial-reviewer != practitioner`.
An implementation agent never certifies its own work. Subagent output is compact
(`STATUS / FINDINGS / EVIDENCE_REFS / BLOCKERS / FOLLOW_UP`) and chain-of-thought
is not retained.

## 3. Delegation pattern

```
EXPLORE (repo-explorer) → PLAN (orchestrator) → IMPLEMENT (implementer)
→ REVIEW (reviewer) → ADVERSARIAL (adversarial-reviewer) → PRACTITIONER
→ DETERMINISTIC VALIDATION → DECIDE (orchestrator)
```

Task packets are bounded: `OBJECTIVE / ALLOWED AREA / INVARIANTS /
ACCEPTANCE CRITERIA / REQUIRED VALIDATION / PROHIBITED SCOPE`.

## 4. Harnesses

Committed, deterministic, safe to re-run. They use **public/documented product
surfaces** wherever practical and classify every step as:

- `PRODUCT_PATH` — documented public CLI/API/UI surface;
- `TEST_ONLY_PATH` — a fixture/convenience path not offered to end users;
- `INTERNAL_SHORTCUT` — an internal call that stands in for a missing product
  surface. Any shortcut is itself a usability/productisation finding.

| Harness | Location | Purpose |
| --- | --- | --- |
| End-user (ATB) | `harnesses/end-user/` | import → verify → inspect → locator → failure states |
| Interoperability | (Mortise) | the same evidence identity survives ATB→Mortise |
| Security / release | `test/` + `make` gates | existing deterministic suites |

Harness reports conform to `harnesses/report-schema.json`. The report goes to
the caller-selected `--out` path and transient work files go to a temp directory;
keep reports under the gitignored `.local/dev/`. They are development evidence
about the product experience — never ATB evidence.

## 5. Fixtures and privacy

- Prefer a real local export when one is safe; otherwise use a clearly labelled
  `REPRESENTATIVE_FIXTURE`.
- Review every fixture for credentials/secrets before use.
- Record honestly whether redaction or synthesis occurred.
- Never commit raw private chat history, tokens, or personal data.
- Fixtures that are safe and intentionally versioned live in `harnesses/*/fixtures/`.

## 6. Review workflow

1. Implementer makes the smallest coherent patch.
2. Correctness reviewer attempts falsification (different family).
3. Adversarial reviewer probes hostile inputs and trust boundaries.
4. Practitioner re-tests through the product surface.
5. Deterministic gates run (`make hygiene-quick`, `make test-golden`; hygiene-quick already runs `make test-go`).
6. Orchestrator decides; no auto-merge. Commit/push/merge/tag are human-authorised
   and denied by the local permission config.

## 7. Continuation state

A run leaves a short continuation record under `.local/dev/state/` capturing:
live HEAD, what was done, what is active, what is blocked, and the next move. It
is local and disposable; the durable process lives in this document and the
harness code.

## 8. Product gates

- **ATB_PRODUCT_GATE** — a real end user can ingest representative AI-interaction
  evidence, verify it, inspect provenance, obtain/use a semantic locator, and
  understand what verification does and does not mean.
- **ATB_MORTISE_INTEROP_GATE** — the same evidence identity survives the full
  cross-product journey without mutation or semantic collapse.
- **TENON_IMPLEMENTATION_GATE** — `NO_GO` unless a separate, human-authorised
  Tenon implementation task begins.
