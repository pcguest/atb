# Research note — SDK auto-capture decision & Custos scope guardrails (Tiers 2–3)

Status: **decision + scoping note.** Unlike the two Tier-1 notes, the open
questions here are product/engineering judgement, not new research. The output is
therefore a *decision and a set of guardrails* rather than a design to build.

## Tier 2 — SDK-native auto-capture (Anthropic / OpenAI)

### What already exists (verified in the tree)

| Path | Covers | File |
|---|---|---|
| `atb intercept` proxy | **any** provider over HTTP; extracts tool calls for **Anthropic Messages, OpenAI Chat Completions, OpenAI Responses**; SSE streaming; privacy-safe body digests; secret-header stripping; optional `--custos` auto-push | `internal/proxy/*` (`toolcalls.go:28`) |
| LangChain callback | in-process LangChain runs (`on_llm_start/_end`, tool, chain) | `sdk/python/atb/langchain_callback.py` |
| Vercel AI middleware + tool gate | in-process Vercel AI SDK calls | `sdk/typescript/src/vercel-ai-middleware.ts`, `vercel-gate.ts` |
| `AutomationSession`, env live-capture (`ATB_BUNDLE_PATH`) | in-process session capture scaffolding | `sdk/typescript/src/automation-session.ts`, `capture.ts` |
| chatlog import | retrospective import (`generic-jsonl`, provider variants) | `internal/capture`, `sdk/*/capture.*` |

So the roadmap line *"wire automatic capture to Claude and OpenAI SDK callbacks"*
is **largely already satisfied for the HTTP surface** by the proxy, and for the
two dominant frameworks by the callback/middleware paths.

### The real question

The official Anthropic and OpenAI SDKs do **not** expose LangChain-style callback
hooks. "Wiring to their callbacks" therefore means one of:

- **(a) HTTP proxy (have it).** Out-of-process, provider-agnostic, zero app code
  change. Sees every request/response that flows through it, including tool calls.
  *Misses:* in-process calls that bypass the proxy (direct egress, a different
  base URL), and **app-level semantics** the wire doesn't carry — the logical
  "agent step / task" boundary, which local tool actually ran, retries collapsed
  into one HTTP call, etc. Needs cert trust + env config.
- **(b) In-process client wrapper.** Wrap/subclass the official client (or its
  `httpx`/`fetch` transport) so each `messages.create` / `responses.create` emits
  the canonical ATB events at the call site, with the surrounding app context.
  *Misses:* only the SDKs you wrap; per-SDK maintenance; risk of monkeypatch
  fragility across SDK versions.

### Decision / recommendation

**Do not build bespoke monkeypatch "callback" shims for the raw SDKs.** They have
no stable callback contract; such shims are fragile and quietly drift with SDK
releases — a poor fit for an evidence tool whose value is *trustworthy* capture.

Instead:

1. **Keep the proxy as the universal capture path.** It already covers Claude and
   OpenAI. Invest marginal effort here (more provider/endpoint coverage) before any
   per-SDK work — it has the best coverage-per-unit-maintenance.
2. **Offer one thin, explicit, supported in-process wrapper per official SDK** —
   an opt-in `atb.capture(client)` / context manager that records canonical events
   at the call boundary *with* app context — for users who need in-process
   semantics the wire can't provide. Explicit and supported beats implicit and
   monkeypatched.
3. **Make the coverage boundary first-class, not a footnote.** Whatever path is
   used, the `atb.capture.scope` attestation must state what that path can and
   cannot see (proxy: "only traffic through the proxy"; wrapper: "only calls via
   this client"). This is the L5 (capture boundary) honesty contract and it is what
   lets a reviewer trust the *edges* of the evidence.

This is a Tier-2 item precisely because the design is settled; it needs a product
decision (above) and incremental engineering, not research. The honest framing for
the roadmap: *"capture for Claude/OpenAI is delivered via the universal proxy plus
opt-in in-process wrappers; we do not claim universal completeness — coverage is
bounded by what flows through the chosen recorder, and every bundle states that
boundary."* (This is already the roadmap's "Out of scope: universal completeness
guarantees" stance — the note just makes the capture strategy match it.)

## Tier 3 — Custos scope guardrails (drift risk)

These two Custos stubs are where the product is most likely to drift away from its
deterministic, evidence-first, honestly-bounded identity. The research output is a
**boundary**, set deliberately *before* anyone implements them.

### `insights` / pitfall detection — keep it deterministic, or don't ship it

The stub (`custos/insights/insights.go`) defines `Analyser.Analyse(bundleID)` with
a TODO for "analysis inputs, evidence limits, and **output confidence semantics**."
"Confidence" strongly implies probabilistic / LLM-style judgement. That collides
head-on with the core identity:

- An LLM "insight" is **not evidence**. It is a generated opinion about evidence.
  Putting it inside Custos — the custody layer whose entire value is being the
  *neutral, deterministic* holder — would poison the chain-of-custody claim and
  invite exactly the "documents about your process, not cryptographic evidence"
  critique the vision levels at governance platforms.

**Guardrail:** `insights` must be a **deterministic findings rollup** over facts
already in the bundle — never a generative judgement. Allowed:

- counts/sequences over existing anomaly flags (e.g. "`tool_without_approval` ×3",
  "`policy_denied_executed` then `data.export` — denied-then-exported pattern");
- deterministic cross-session aggregation (this actor's flag frequencies).

Forbidden: LLM summarisation, risk scores that imply model judgement, any field
called "confidence" that isn't a deterministic count/ratio with a stated formula.

**Concrete recommendation:** rescope `insights` to reuse the **incident findings
machinery already built** (`internal/incident/findings.go`) — promote it to a
cross-bundle deterministic rollup. If a genuinely AI-generated narrative is ever
wanted, it belongs in a *separate, clearly-labelled, non-custody* tool, never
inside Custos, and never presented as evidence.

### `oversight` review queue — a review *record*, not a workflow engine

The stub (`custos/oversight/oversight.go`) sketches `Queue/Approve/Reject` with a
TODO for "queue storage, assignment, and SLA semantics." Queues, assignment, and
SLAs are **hosted multi-tenant workflow** — outside the runtime per `AGENTS.md`.

**Guardrail:** the in-repo reference should demonstrate the **evidence shape of a
human review decision**, not the workflow that produces it. Concretely: a signed,
append-only `review.requested` / `review.resolved` annotation referencing a
receipt or bundle, carrying *who* (verifiable principal — ties to the Art 14
reviewer-identity-anchoring work), *when*, *outcome*, and *reason*. That is
verifiable custody evidence of oversight. The queue/assignment/SLA *system* is a
hosted concern and stays out of the repo.

### `onboarding` — out of runtime scope

`Provision` / `Connect` with "credential storage" side effects is multi-tenant
account provisioning — outside the runtime per `AGENTS.md`. The reference layer
already has what it needs to demonstrate per-org binding: the signing
`PolicyStore` (key/TSA/rotation per org). No credential-provisioning side effects
should land in-repo. Leave the stub as an interface that documents the boundary.

### `discovery` — defer until there is a real consumer

The *tool-signature* discovery scaffold is speculative without a real discovery
source feeding it. Implementing a known-tool lookup that nothing populates is
scaffolding for its own sake. Defer until a concrete capture/corroboration
consumer needs it; revisit then.

> **Update (2026-06-05):** `registry` was originally a sibling tool-signature
> scaffold and is no longer covered by this guardrail. Per
> `docs/custos-handoff.md`, the registry is the **receipt + digest registry** —
> a lookup index over the (real, populated) receipt store, answering "which
> receipts custody this bundle hash?". That has a real consumer (the daemon's
> read API) and is implemented in `custos/registry/`. Only the tool-signature
> `discovery` scaffold remains deferred.

## One-paragraph summary

Tier 2 is a **decision, not research**: keep the proxy as the universal Claude/OpenAI
capture path, add opt-in in-process wrappers only where app-context is needed, and
make the capture boundary explicit in every bundle — do not build fragile
monkeypatch callback shims. Tier 3 is about **holding the line**: `insights` may
only ever be a deterministic rollup of facts already in evidence (reuse the
incident findings engine); `oversight` is a signed review *record*, not a queue;
`onboarding`/`discovery`/`registry` stay boundary-documenting stubs until a real
consumer or an explicit hosted-layer decision exists. The throughline is the same
discipline the rest of ATB holds: record and verify facts; never generate, assert,
or host beyond what the evidence supports.
