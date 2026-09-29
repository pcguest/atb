# ATB evidence attribution

> Status: **RATIFIED**
> Decision: **reuse existing primitives** — ATB does not add a canonical
> agent/run attribution model.

This document is the normative explanation of how ATB evidence answers
questions about *who or what produced a record*, and how records relate. It
records an architecture decision (a reuse-only attribution model) and the
attribution layer map that producers, SDKs, and the viewer follow.

It does not change the canonical event envelope, canonical hashing, the bundle
format, golden vectors, or the semantic evidence locator. Attribution is a
description of existing evidence, not a new class of evidence.

## Context

An ATB bundle records portable, independently verifiable AI-agent evidence.
The semantic evidence locator (`docs/specification/evidence-locator.md`)
answers *which record* a reference denotes. Investigators also ask *what
produced that record* and *how records relate* — who acted, which agent,
model, tool, run, and on whose behalf.

An architecture review proposed adding canonical attribution fields
(`agent_id`, `agent_type`, `agent_instance_id`, `run_id`, `parent_run_id`,
`model_version`, `toolset_digest`). The review found that ATB already carries
sufficient primitives for portable producer/run attribution, and that the
remaining weakness is documentation, producer coverage, and human-readable
presentation rather than missing canonical schema. This document ratifies that
finding.

## Forensic requirement

ATB must let an investigator answer, from the evidence alone:

- what a record reports about its producer and context;
- how records group and relate (execution and capture context);
- what supports an asserted identity; and
- which record a reference denotes.

ATB must **not** answer, from attribution:

- whether a producer was authorised;
- whether an assertion is true;
- whether a producer should be trusted; or
- whether anything may execute.

## Existing primitives

ATB's existing primitives already cover the forensic questions. They fall into
five categories that must not be blurred:

| Category | Location | Members |
| --- | --- | --- |
| **Canonical envelope** | top-level event fields (hashed) | `seq`, `prev_hash`, `type`, `hash_algo`, `data`, `actor_id`, `org_id`, `workspace_id`, `timestamp`, `trace_id`, `span_id`, `parent_span_id` |
| **Payload convention** | inside `event.data` | `principal{type,id_hash,on_behalf_of}`, `identity_evidence`, `model_id`/`model_provider`, `framework`/`framework_version`, `run_id`, `session_id`, `tool_name`/`tool_call_id` and digests |
| **Bundle metadata** | bundle manifest | `capture_run_id` (capture run that created the bundle) |
| **Import/acquisition metadata** | top-level `event.acquisition` (hashed envelope field) | `mode`, `source_system`, `source_record_id`, `source_timestamp`, `acquired_at`, `adapter`, `adapter_version`, `source_digest`, `checkpoint*` |
| **Derived information** | computed views | incident findings, session index, viewer summaries, `capabilities` |

Notes:

- **Canonical envelope** fields are hashed and are the portable identity of an
  event. `actor_id`, `org_id`, and `workspace_id` are *caller-asserted*
  context: ATB preserves them, it does not verify them.
- **Payload convention** fields live inside `event.data` and are hashed as part
  of the event, but they are conventions per event type, not envelope fields.
  `principal` is defined for `ai.action.precommit`
  (`schemas/event.v1.json`, `documented_event_types`).
- **Import/acquisition metadata** (`acquisition`) is an optional top-level
  canonical envelope field, declared in `schemas/event.v1.json` and included in
  canonical hashing when present. It records how evidence entered ATB
  (import/capture) and traces the original source representation. It is listed
  as its own category because it is provenance about acquisition, not about the
  acting producer: it is not producer identity, authority, or truth, and it does
  not alter hashing semantics here.
- **Derived information** is never canonical; it is a reading of canonical
  evidence and must be labelled as such.

## Attribution layer map

Attribution is layered. Each layer answers a different question, and a lower
layer is not evidence of an upper one:

```
session actor            (proxy-resolved actor: actor_id, data.actor)
    ↓
request actor            (application-asserted: actor_id_hash)
    ↓
acting principal         (gate-asserted: principal{type,id_hash,on_behalf_of})
    ↓
identity evidence        (external assertions: identity_evidence)
```

Execution context is a separate axis:

```
trace_id
    ↓
span_id
    ↓
parent_span_id
```

Capture context is a third axis:

```
bundle
    ↓
capture_run_id
```

Payload context (per event type) carries model, framework, run, session, and
tool detail where a producer recorded it.

## Forensic question → primitive map

| Forensic question | Primitive | Availability |
| --- | --- | --- |
| WHO performed the action? | `actor_id` (envelope) / request `actor_id_hash` / action `principal` — by layer | available; **producer-dependent** (see below) |
| WHICH logical producer? | `principal{type,id_hash}` where the producer supplied it | producer-dependent (currently `ai.action.precommit`) |
| WHICH execution context? | `trace_id` / `span_id` / `parent_span_id` (envelope) | available when the producer recorded trace context |
| WHICH capture operation? | `capture_run_id` (bundle manifest) | available; **bundle-level** |
| WHICH model? | `model_id` / `model_provider` (payload, e.g. `ai.model.invoked`) | payload-only |
| WHICH framework/harness? | `framework` / `framework_version` (payload, imported/runtime) | payload-only |
| WHICH tool? | tool invocation event/payload (`atb.tool.call`, `ai.tool.exec`) | available |
| ON WHOSE BEHALF? | `principal.on_behalf_of` | producer-dependent |
| WHAT supports identity? | `identity_evidence` (asserted; unverified by design) | producer-dependent |
| WHICH record? | semantic evidence locator | available |
| WHICH tools were available? | none recorded | **unavailable** |

Availability terms:

- **available** — recorded by default for the relevant event path.
- **producer-dependent** — present only when the producer knows and records it;
  omission is valid and must not be read as "no producer".
- **payload-only** — a payload convention, not part of the canonical envelope.
- **bundle-level** — describes the bundle/capture, not an individual record.
- **unavailable** — not represented today; do not infer it.

Optional evidence is never mandatory. A missing attribution field means the
producer did not report it — not that the action lacked a producer.

## Identity ≠ authority

ATB records **assertions**. `actor_id`, `principal.id_hash`, and
`identity_evidence` are producer-supplied claims preserved in the hash chain;
`on_behalf_of` is a descriptive delegation hint, not an authorisation. A
signed bundle can establish tamper evidence *around* a recorded claim; it does
not establish the external truth of the claim.

ATB must not render, or imply, that a principal is verified, authorised,
trusted, or permitted to execute. Authority, policy, delegation validity,
approval, and trust are interpreted outside ATB (Mortise governance; Tenon
execution). `on_behalf_of` must never be used for gating or authorisation.

## Interoperability

Attribution maps onto existing conventions; ATB does not adopt their schemas
wholesale and does not become an observability backend:

| External | Concept | ATB equivalent |
| --- | --- | --- |
| W3C Trace Context | trace-id / span-id / parent-id | `trace_id` / `span_id` / `parent_span_id` |
| OpenTelemetry GenAI | `gen_ai.agent.name` | `actor_id` / `principal.id_hash` (partial) |
| OpenTelemetry GenAI | `gen_ai.tool.definitions` | none (deferred) |
| OpenInference | `openinference.span.kind` | `event.type` |
| OpenInference | `session.id` / `user.id` | `session_id` / `actor_id` |
| RFC 8693 (OAuth token exchange) | `act` / subject | `principal.id_hash` / `principal.on_behalf_of` (asserted only) |
| LangSmith / LangGraph | run tree (`run_id`, `parent_run_id`) | `span_id` / `parent_span_id` / `trace_id` |

Telemetry-only concepts (token counts, cost, latency, live spans, model scores,
ephemeral instance ids) are out of scope; ATB interoperates via import
adapters rather than recording them.

## Privacy

Reuse introduces no new correlation surface. Existing identifiers are kept in
their existing forms: hashed/opaque for principal identifiers, caller-supplied
for `actor_id`/`org_id`/`workspace_id`. Attribution must not convert a private
identifier into a more prominent or more raw form. Human-readable does not mean
more personal data.

Producers should be aware that `identity_evidence.subject` is caller-supplied
and may be plaintext (it is not hashed by ATB), and that `principal.id_hash`
and `principal.on_behalf_of` are producer-supplied identifier strings (hashed
by convention, not by ATB). Whether such values are masked in a viewer is
governed by operator-configured PII rules, not by attribution. Producers must
not place raw personal identifiers in these fields where a hashed or opaque
identifier would serve.

## Security

Attribution is producer-controlled and remains visibly an assertion. ATB adds
no identity authority, no trust scoring, no network/registry/identity
resolution, and no canonical producer object. Producer-kind and delegation are
recorded only where the producer reports them; ATB does not infer them.

## Human readability

The viewer uses progressive disclosure:

```
human-readable evidence (summary)
    ↓
attribution summary / detail (reported acting principal, identity evidence)
    ↓
raw evidence (canonical record)
```

Preferred language: *Acting principal*, *Reported by producer*, *Identity
evidence*, *On behalf of*, *Not reported*. Forbidden unless independently
established (which this model never does): *Verified agent*, *Trusted
identity*, *Authorised actor*.

## Compatibility

Additive and opt-in. No canonical schema change, no hashing change, no golden
vector change, no locator change, no SDK break. Producers that know a
principal may record it using the existing payload convention; producers that
do not must omit it.

## Consequences

- Investigators gain a consistent, documented reading of existing evidence.
- Coverage gaps remain for producers that do not record optional context; these
  are coverage, not schema, problems.
- ATB avoids an agent ontology and any governance/authority semantics.

## Non-goals

ATB is not an agent framework, orchestration runtime, policy engine, execution
engine, trust scorer, model evaluator, identity authority, telemetry backend,
or a replacement for distributed tracing.

## Deferred work

- Canonicalising any attribution field. No demonstrated forensic question
  currently requires a new canonical primitive.
- `toolset_digest` / available-tools inventory.
- `model_version` as a payload convenience (belongs inside the model payload).
- Recording `principal` beyond `ai.action.precommit` where producers genuinely
  know it (payload convention only; never synthetic).

None of the deferred items are forbidden; they are deferred because no
demonstrated forensic question presently requires them.
