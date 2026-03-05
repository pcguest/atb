# ATB v1.0 Specification

**ATB (Agent Trace Bundle)** is a file format and protocol for creating tamper-evident, replayable audit trails of AI agent workflow events.

---

## 1. Overview

An ATB bundle is a newline-delimited JSON (NDJSON) file where each line contains a single JSON object — a **record** — consisting of an **event** and its **hash**. Records are ordered by sequence number and form a cryptographic chain.

---

## 2. File Format

### 2.1 Bundle File

A bundle file uses the `.atb` extension and is stored in the `run.atb/` directory by default. Each line is a valid JSON object terminated by a newline character (`\n`).

```
run.atb/
└── bundle.atb
```

### 2.2 Record Structure

Each line in a bundle file is a JSON object with the following schema:

```json
{
  "event": {
    "seq": 1,
    "prev_hash": "0000000000000000000000000000000000000000000000000000000000000000",
    "type": "dev.session",
    "data": {},
    "actor_id": "paddy",
    "org_id": "pcguest",
    "workspace_id": "local"
  },
  "hash": "cdc87dac2d8d61bf8a8b8e9f2a4c5d6e..."
}
```

| Field | Type | Description |
|-------|------|-------------|
| `event.seq` | integer | 1-based sequence number within the bundle |
| `event.prev_hash` | string | Hex-encoded SHA-256 hash of the preceding event |
| `event.type` | string | Dot-namespaced event type identifier |
| `event.data` | object | Arbitrary JSON-serialisable payload |
| `event.actor_id` | string (optional) | Actor identifier for multi-tenant attribution |
| `event.org_id` | string (optional) | Organization identifier for multi-tenant attribution |
| `event.workspace_id` | string (optional) | Workspace identifier for multi-tenant attribution |
| `hash` | string | Hex-encoded SHA-256 hash of this event |

---

## 3. Hash Algorithm

### 3.1 Genesis Hash

The first event in a bundle uses a **genesis hash** as its `prev_hash`:

```
prev_hash = "0000000000000000000000000000000000000000000000000000000000000000"
```

### 3.2 Hash Computation

The hash for event `n` is computed as follows:

```
hash(n) = SHA256( UTF8(hex(hash(n-1))) || RFC8785(event(n)) )
```

Where:
- `UTF8(hex(hash(n-1)))` is the UTF-8 encoding of the previous event's hex hash string.
- `RFC8785(event(n))` is the RFC 8785 canonical JSON encoding of the event object (including `seq`, `prev_hash`, `type`, `data`, and any optional identity fields that are set).
- `||` denotes byte concatenation.

### 3.3 RFC 8785 Canonicalization

Before hashing, the event object is serialised using the [RFC 8785 JSON Canonicalization Scheme (JCS)](https://www.rfc-editor.org/rfc/rfc8785):

- Object keys are sorted by their UTF-16 code unit sequence.
- Numbers use ES6 serialisation (no trailing zeros, no unnecessary exponents).
- Strings use minimal escaping.
- No whitespace between tokens.

This ensures that the same event produces the same hash in every language and runtime.

Unset optional fields are omitted from canonicalization output. For example, if `actor_id`, `org_id`, or `workspace_id` are not set, they are excluded from the canonical JSON bytes before hashing.

---

## 4. Event Types

Event types use dot-namespaced identifiers. The following types are defined by the ATB standard:

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

---

## 5. Verification

To verify a bundle, a verifier must:

1. Read all records in order.
2. For each record at position `i`:
   a. Set `event.prev_hash` to the hash of the preceding record (or the genesis hash for `i = 0`).
   b. Set `event.seq` to `i + 1`.
   c. Compute `hash = SHA256(UTF8(prev_hash) || RFC8785(event))`.
   d. Assert that the computed hash equals the stored `hash` field.
3. If any assertion fails, the bundle has been tampered with.

---

## 6. Storage

### 6.1 Local Storage

The default storage location is `run.atb/bundle.atb` relative to the current working directory. This directory should be excluded from version control (`.gitignore`).

### 6.2 Cloud Storage (Pro)

ATB Pro supports encrypted cloud storage via Cloudflare R2. Bundles are encrypted client-side before upload using AES-256-GCM.

---

## 7. Schema Evolution (v1.0+)

ATB v1.0+ supports optional identity fields on events:

- `actor_id`
- `org_id`
- `workspace_id`

Compatibility rules:

- Old bundles (created before optional identity fields were added) verify unchanged with newer SDKs.
- New bundles that set optional identity fields produce different hashes (expected, because those fields are included in canonicalization).
- Unset optional identity fields are omitted from canonicalization (`omitempty` behavior), preserving canonical byte compatibility with legacy events that did not define these fields.

### TypeScript SDK

Optional fields (`actor_id`, `org_id`, `workspace_id`) are omitted from canonicalization when `undefined`. This ensures backward compatibility with bundles created before v1.0+.

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
