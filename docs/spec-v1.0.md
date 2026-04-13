# ATB v1.0 specification

**ATB (Agent Trace Bundle)** is a file format and protocol for creating tamper-evident, verifiable audit trails of AI workflow events.

---

## 1. Overview

An ATB bundle is a newline-delimited JSON (NDJSON) file where each line contains a single JSON object, a **record**, consisting of an **event** and its **hash**. New bundles start with a manifest record at `seq = 0`; later records form a cryptographic chain on top of that manifest.

---

## 2. File Format

### 2.1 Bundle File

A bundle file uses the `.atb` extension and is stored in the `run.atb/` directory by default. Each line is a valid JSON object terminated by a newline character (`\n`).

```text
run.atb/
└── bundle.atb
```

### 2.2 Record Structure

Each line in a bundle file is a JSON object with the following schema:

```json
{
  "event": {
    "seq": 0,
    "prev_hash": "0000000000000000000000000000000000000000000000000000000000000000",
    "type": "atb.bundle.manifest",
    "hash_algo": "sha256",
    "data": "{\"version\":\"1\",\"created_at\":\"2026-04-01T00:00:00Z\",\"bundle_id\":\"00112233445566778899aabbccddeeff\"}",
    "timestamp": "2026-04-01T00:00:00Z"
  },
  "hash": "44dea4de15506571cc6c9006b250aab0e8b3bdcb8269bd5c02f39a34b6ee0586"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `event.seq` | integer | Sequence number within the bundle. New bundles reserve `0` for the manifest record; later records use `1..N`. |
| `event.prev_hash` | string | Hex-encoded SHA-256 hash of the preceding event |
| `event.type` | string | Dot-namespaced event type identifier |
| `event.hash_algo` | string (optional) | Hash algorithm identifier. Current runtimes emit `sha256`. |
| `event.data` | any JSON value | Arbitrary JSON-serialisable payload |
| `event.actor_id` | string (optional) | Actor identifier for multi-tenant attribution |
| `event.org_id` | string (optional) | Organization identifier for multi-tenant attribution |
| `event.workspace_id` | string (optional) | Workspace identifier for multi-tenant attribution |
| `event.timestamp` | string (optional) | RFC 3339 UTC event creation time |
| `event.trace_id` | string (optional) | W3C trace context trace identifier |
| `event.span_id` | string (optional) | W3C trace context span identifier |
| `event.parent_span_id` | string (optional) | W3C trace context parent span identifier |
| `hash` | string | Hex-encoded SHA-256 hash of this event |

---

## 3. Hash Algorithm

### 3.1 Genesis Hash

The manifest record in a new bundle uses a **genesis hash** as its `prev_hash`. Legacy manifest-less bundles also use this value on their first record:

```text
prev_hash = "0000000000000000000000000000000000000000000000000000000000000000"
```

### 3.2 Hash Computation

The hash for event `n` is computed as follows:

```text
hash(n) = SHA256( UTF8(hex(hash(n-1))) || RFC8785(event(n)) )
```

Where:
- `UTF8(prev_hash)` is the UTF-8 encoding of the previous event's hex hash string.
- `RFC8785(event(n))` is the RFC 8785 canonical JSON encoding of the event object (including `seq`, `prev_hash`, `type`, `data`, and any optional fields that are set).
- `||` denotes byte concatenation.

### 3.3 RFC 8785 Canonicalization

Before hashing, the event object is serialised using the [RFC 8785 JSON Canonicalization Scheme (JCS)](https://www.rfc-editor.org/rfc/rfc8785):

- Object keys are sorted by their UTF-16 code unit sequence.
- Numbers use ES6 serialisation (no trailing zeros, no unnecessary exponents).
- Strings use minimal escaping.
- No whitespace between tokens.

This ensures that the same event produces the same hash in every language and runtime.

Unset optional fields are omitted from canonicalisation output. For example, if `actor_id`, `org_id`, `workspace_id`, `timestamp`, `trace_id`, `span_id`, or `parent_span_id` are not set, they are excluded from the canonical JSON bytes before hashing.

---

## 4. Event Types

Event types use dot-namespaced identifiers. The following types are defined by the ATB standard:

### Reserved system event types

| Type | Description |
|------|-------------|
| `atb.bundle.manifest` | Bundle manifest record. First record in a new bundle (`seq = 0`). |
| `atb.bundle.anchor` | RFC 3161 TSA anchor record appended after anchoring. |
| `atb.bundle.signature` | Ed25519 bundle signature record. |
| `atb.snapshot` | Named bundle checkpoint appended by `atb snapshot`; `bundle_hash` commits to the serialised bundle prefix that existed immediately before the snapshot record was appended. |

The `bundle_hash` field is the SHA-256 (hex) of the serialised bundle
prefix that existed immediately before the snapshot record was appended,
using the same NDJSON serialisation as `atb snapshot`.

When `atb verify --with-snapshot-check` is used, the verifier recomputes
that prefix hash for each `atb.snapshot` record and compares it with the
recorded `bundle_hash`. On mismatch the verifier reports
`snapshot_hash_mismatch at seq N` and exits non-zero.

Without `--with-snapshot-check`, `bundle_hash` is not verified: ordinary
integrity checks still validate the hash chain, but they do not prove the
snapshot metadata matches the historical prefix.

### Legacy event types (v1.0, superseded)

These types are defined for backward compatibility with bundles created before the current AI trace specification. New integrations MUST use the types defined in `docs/spec-ai-traces.md` instead.

| Type | Description |
|------|-------------|
| `dev.session` | A development session |
| `decision` | An architectural or product decision |
| `release` | A software release |
| `langchain.llm.start` | LangChain LLM invocation started |
| `langchain.llm.end` | LangChain LLM invocation completed |
| `langchain.chain.start` | LangChain chain started |
| `langchain.chain.end` | LangChain chain completed |
| `langchain.tool.start` | LangChain tool invocation started |
| `langchain.tool.end` | LangChain tool invocation completed |
| `langchain.agent.action` | LangChain agent tool selection |
| `langchain.agent.finish` | LangChain agent completed |

Custom event types are permitted using reverse-domain notation (e.g., `com.example.custom_event`).

### Current AI integration lifecycle event types (current AI trace specification)

| Type | Description | Reference |
|------|-------------|-----------|
| `ai.llm.call` | Canonical LLM lifecycle event type for new integrations | `docs/spec-ai-traces.md` |
| `ai.tool.exec` | Canonical tool execution event type for new integrations | `docs/spec-ai-traces.md` |
| `ai.chain.run` | Canonical chain or step lifecycle event type for new integrations | `docs/spec-ai-traces.md` |

### Governance and control-plane event types (verification/profile flows)

The following canonical event families are used by obligation profiles and verification workflows:

- `ai.request.received`, `ai.response.sent`
- `ai.policy.decision`
- `ai.retrieval.executed`, `ai.model.invoked`, `ai.model.output`
- `ai.action.precommit`, `ai.action.executed`, `ai.action.committed`
- `ai.human.approval`, `ai.override.requested`
- `ai.job.scheduled`, `ai.job.started`, `ai.job.step`, `ai.job.completed`

The built-in profile templates currently evaluate the following required event sets:

- `atb.profile.privileged_tool_action` requires `atb.bundle.manifest`, `ai.request.received`, `ai.action.precommit`, `ai.policy.decision`, `ai.action.executed`, and `ai.action.committed`. `ai.human.approval` is warning-level evidence and is evaluated when actions execute.
- `atb.profile.rag_answer` requires `atb.bundle.manifest`, `ai.request.received`, `ai.model.invoked`, and `ai.model.output`. `ai.retrieval.executed`, `ai.policy.decision`, and `ai.response.sent` are warning-level evidence only.
- `atb.profile.data_export` requires `atb.bundle.manifest`, `ai.request.received`, `ai.policy.decision`, `ai.action.precommit`, `ai.action.executed`, and `ai.action.committed`. `ai.human.approval` is warning-level evidence and is evaluated when exports execute. `data.export.*` event types remain the target taxonomy for export-specific lifecycle records, but the current built-in template still evaluates the `ai.action.*` control-plane flow. Migration to `data.export.*` is planned for a future release.
- `atb.profile.policy_decision` requires `atb.bundle.manifest`, `ai.request.received`, and `ai.policy.decision`. `ai.action.precommit` is warning-level evidence only.
- `atb.profile.human_override` requires `atb.bundle.manifest`, `ai.request.received`, `ai.human.approval`, `ai.action.precommit`, and `ai.action.executed`. `ai.action.committed` is warning-level evidence only.
- `atb.profile.background_automation` requires `atb.bundle.manifest`, `ai.job.scheduled`, `ai.job.started`, and `ai.job.completed`. `ai.job.step` is warning-level evidence only. It does not require `ai.request.received`, and profile auto-detection should rely on recorded `ai.job.*` events rather than `purpose_tag` on request events.

For the machine-readable registry and profile-scoped criticality, see `atb events` and `internal/event/types.go`.

---

## 5. Verification

To verify a bundle, a verifier must:

1. Read all records in order.
2. Detect whether the first record is the manifest type `atb.bundle.manifest`.
3. For each record at position `i`:
   a. Set `event.prev_hash` to the hash of the preceding record (or the genesis hash for the first record).
   b. Set `event.seq` to `0` for the manifest record, `i` for later records in a manifest-first bundle, or `i + 1` for a legacy manifest-less bundle.
   c. Compute `hash = SHA256(UTF8(prev_hash) || RFC8785(event))`.
   d. Assert that the computed hash equals the stored `hash` field.
4. If any assertion fails, the bundle has been tampered with.

---

## 6. Storage

### 6.1 Local Storage

The default storage location is `run.atb/bundle.atb` relative to the current working directory. This directory should be excluded from version control (`.gitignore`).

### 6.2 Optional Encrypted Payloads

ATB supports optional client-side bundle encryption via `atb encrypt` / `atb decrypt`.

- Cipher: AES-256-GCM
- Wire format: `ATBE` magic + version + salt + nonce + auth tag + ciphertext
- Key derivation: PBKDF2-SHA256 with versioned parameters
  - `0x01`: PBKDF2-SHA256 (`100000` iterations). Retained for decrypt compatibility with existing encrypted bundles.
  - `0x02`: PBKDF2-SHA256 (`600000` iterations). Default for newly encrypted bundles.

`atb decrypt` accepts both wire-format versions. `atb encrypt` writes version `0x02`.

Optional encrypted handoff is being evaluated separately in [docs/spec/bundle-push.md](./spec/bundle-push.md). The bundle-push design-intent document describes the planned v1.6 `atb push` interface and is not part of the v1.0 local storage contract.

---

## 7. Schema Evolution (v1.0+)

ATB v1.0+ supports optional fields on events:

- `actor_id`
- `org_id`
- `workspace_id`
- `hash_algo`
- `timestamp`
- `trace_id`
- `span_id`
- `parent_span_id`

Compatibility rules:

- Old bundles (created before the manifest record or optional fields were added) verify unchanged with newer SDKs.
- New bundles that set optional fields produce different hashes (expected, because those fields are included in canonicalisation).
- Unset optional fields are omitted from canonicalisation (`omitempty` behaviour), preserving canonical byte compatibility with legacy events that did not define them.

### TypeScript SDK

Optional fields (`actor_id`, `org_id`, `workspace_id`) are omitted from canonicalisation when `undefined`. This ensures backward compatibility with bundles created before v1.0+.

```ts
// Omitted when undefined
const event: Event = {
  seq: 1,
  prev_hash: "0000...",
  type: "test",
  data: { x: 1 },
};

// Included when set
const eventWithActor: Event = {
  seq: 1,
  prev_hash: "0000...",
  type: "test",
  data: { x: 1 },
  actor_id: "user",
};
```

## 8. Versioning

This document describes ATB specification version **1.0**. Future versions will be backwards-compatible unless a major version increment is made.
