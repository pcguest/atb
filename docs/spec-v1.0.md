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
| `event.actor_id` | string (optional) | Caller-asserted actor attribution context, preserved by the hash chain after recording |
| `event.org_id` | string (optional) | Caller-asserted organisation attribution context, preserved by the hash chain after recording |
| `event.workspace_id` | string (optional) | Caller-asserted workspace attribution context, preserved by the hash chain after recording |
| `event.timestamp` | string (optional) | RFC 3339 UTC event creation time |
| `event.trace_id` | string (optional) | W3C trace context trace identifier |
| `event.span_id` | string (optional) | W3C trace context span identifier |
| `event.parent_span_id` | string (optional) | W3C trace context parent span identifier |
| `hash` | string | Hex-encoded SHA-256 hash of this event |

> **NOTE — Manifest data encoding (v1 and v2)**
>
> The `atb.bundle.manifest` record has **two supported data shapes** selected by the `version` field. The default writer remains **v1** for compatibility; **v2** is the opt-in structured manifest format.
>
> | Field           | v1 (legacy compatibility)                                                   | v2 (current structured format)                       |
> |-----------------|-----------------------------------------------------------------------------|------------------------------------------------------|
> | `event.data`    | JSON-encoded **string** containing manifest fields                           | Structured **object** containing manifest fields     |
> | `version`       | `"1"` (string)                                                               | `2` (integer)                                        |
> | `created_at`    | RFC 3339 UTC timestamp                                                       | RFC 3339 UTC timestamp                               |
> | `bundle_id`     | 32-char lowercase hex                                                        | 32-char lowercase hex                                |
> | Read pattern    | `json.Unmarshal(event.data.(string))` to recover fields                      | `event.data.(map)` directly                          |
>
> v1's double-encoding is **historical, not deliberate design** — early ATB writers stored manifest fields as a JSON string and the format is preserved verbatim so existing bundles re-verify byte-for-byte. Independent implementers MUST reproduce the v1 double-encoding exactly when writing v1 for compatibility; v2 stores the manifest as a regular structured event payload.
>
> The manifest version is independent of the bundle schema version. v1 data is a JSON-encoded string; v2 data is a structured JSON object. Both are hashed by the same RFC 8785 canonicaliser as any other event — the difference is only what `data` contains.
>
> Readers MUST handle both shapes. A reader that encounters a manifest with `version` greater than the highest version it understands (`ManifestVersionMax`, currently 2) MUST refuse to open the bundle and return an error wrapping `ErrMalformed`.

---

## 2.3 Format constraints

- Each NDJSON record must fit on a single line of at most **16,777,216 bytes** (16 MiB, `MaxLineSizeBytes`). A reader MAY reject lines exceeding this limit.
- Blank lines between records are permitted and ignored.
- The file MUST be UTF-8 encoded with LF (`\n`) line endings.

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

### 3.4 Canonical hash input

The canonical input to SHA-256 for each record is:

```text
UTF8( hex(prev_hash) + RFC8785( event ) )
```

where `event` is the JSON object containing the fields below — **and no others** — serialised via RFC 8785 canonical JSON. This table is the authoritative pinned field set for v1; it matches the JSON tags and `omitempty` annotations on the `Event` struct in `internal/event/event.go`.

| Go field       | JSON key          | Type          | Always emitted? | Omitted when                            |
|----------------|-------------------|---------------|-----------------|-----------------------------------------|
| `Sequence`     | `seq`             | integer       | yes             | never                                   |
| `PrevHash`     | `prev_hash`       | string (hex)  | yes             | never                                   |
| `Type`         | `type`            | string        | yes             | never                                   |
| `HashAlgo`     | `hash_algo`       | string        | conditional     | empty string (runtime currently always sets `"sha256"`) |
| `Data`         | `data`            | any JSON      | yes             | never (may be `null`)                   |
| `ActorID`      | `actor_id`        | string        | conditional     | nil pointer (field absent)              |
| `OrgID`        | `org_id`          | string        | conditional     | nil pointer (field absent)              |
| `WorkspaceID`  | `workspace_id`    | string        | conditional     | nil pointer (field absent)              |
| `Timestamp`    | `timestamp`       | string (RFC 3339) | conditional | empty string                            |
| `TraceID`      | `trace_id`        | string (32 hex) | conditional   | empty string                            |
| `SpanID`       | `span_id`         | string (16 hex) | conditional   | empty string                            |
| `ParentSpanID` | `parent_span_id`  | string (16 hex) | conditional   | empty string                            |

**Stability note:** If you add a new field to the `Event` struct, it MUST appear in this table before the struct change lands, and the manifest version MUST be bumped (see §9 *Schema versioning*). Failure to do this silently breaks all existing bundle verification. The cross-language canonical-hash golden corpus at `internal/hash/testdata/golden.json` exists to detect such drift; any change that causes `TestCanonicalHashGolden` to fail is a breaking schema change.

### 3.5 Canonical hash golden tests

The file `internal/hash/testdata/golden.json` pins the exact byte output of ATB's RFC 8785 + SHA-256 canonicalisation pipeline across a corpus of representative events (minimal, all-optional-fields-set, structured/null/float/unicode payloads, genesis sentinel, and a chained pair). Any change that causes

```text
go test ./internal/hash/... -run TestCanonicalHashGolden
```

to fail is a **breaking schema change** and requires a manifest version bump (§9), a `CHANGELOG.md` entry, and regeneration of the golden file with the new expected values. The same corpus must be mirrored byte-for-byte in the Python and TypeScript SDKs. Do not regenerate the Go golden file without updating those.

To regenerate after a vetted format change:

```text
go test ./internal/hash/... -run TestCanonicalHashGolden -update
```

---

## 4. Event Types

Event types use dot-namespaced identifiers. The following types are defined by the ATB standard:

### Reserved system event types

| Type | Description |
|------|-------------|
| `atb.bundle.manifest` | Bundle manifest record. First record in a new bundle (`seq = 0`). |
| `atb.bundle.anchor` | RFC 3161 TSA anchor record appended after anchoring. |
| `atb.bundle.signature` | Ed25519 bundle signature record. |
| `atb.snapshot` | Named bundle checkpoint appended by `atb snapshot`; `bundle_hash` commits to the serialised bundle prefix that existed immediately before the snapshot record was appended. See §4.1 for the `data` payload schema. |

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

### 4.1 `atb.snapshot` data payload

The `data` field of an `atb.snapshot` record is a JSON object with the following fields. All fields are required.

| Go field     | JSON key       | Type    | Required | Description                                                              |
|--------------|----------------|---------|----------|--------------------------------------------------------------------------|
| `Name`       | `name`         | string  | yes      | Human-readable snapshot label provided by the operator                   |
| `BundleHash` | `bundle_hash`  | string  | yes      | Hex SHA-256 of the bundle NDJSON prefix that existed before this record  |
| `RecordCount`| `record_count` | integer | yes      | Number of records in that prefix (i.e., before this snapshot record)     |
| `SnapshotAt` | `snapshot_at`  | string  | yes      | RFC 3339 UTC timestamp of when the snapshot was taken                    |

The Go source of truth is `snapshotEventData` in `cmd/atb/snapshot.go`.

Snapshot names are validated before any bundle I/O. A valid name is non-empty
after trimming whitespace, is at most 128 runes, and contains no ASCII control
characters, `/`, `\`, or NUL.

### 4.2 `atb.bundle.signature` data payload

Required fields:

| JSON key      | Type           | Description                                                              |
|---------------|----------------|--------------------------------------------------------------------------|
| `bundle_hash` | string (hex)   | SHA-256 of the pre-signature bundle NDJSON bytes                         |
| `signature`   | string (base64)| Ed25519 signature over the raw 32-byte `bundle_hash`                     |
| `pubkey`      | string (base64)| Raw 32-byte Ed25519 public key                                            |

A back-compatibility alias `public_key` is accepted by the verifier for bundles signed before the CLI standardised on `pubkey`. New writers MUST emit `pubkey`.

#### Pre-image (KMS signing contract)

The signed pre-image is **`SHA-256(pre-signature bundle NDJSON bytes)`** — i.e. the 32-byte digest of every NDJSON byte written to disk *before* the signature record itself is appended. This is the byte sequence passed to every signer (local Ed25519, AWS KMS, GCP KMS, Vault Transit) as the message to sign. Backends that hash internally (KMS) MUST be configured to receive the digest, not the bundle bytes; backends that sign the message verbatim (local Ed25519) sign the 32-byte digest directly. The verifier MUST recompute the same digest from the bundle prefix and feed it to the algorithm's `Verify` primitive.

Optional fields (current, additive):

The following optional fields may appear in the `atb.bundle.signature` data payload. Implementations MUST ignore unknown fields and MUST NOT reject signatures that omit them.

| JSON key       | Type    | Description                                                                                     |
|----------------|---------|-------------------------------------------------------------------------------------------------|
| `key_id`       | string  | Opaque key identifier scoped to the backend (empty/absent for `local`)                          |
| `backend`      | string  | Signing backend: `local`, `https-http`, `aws-kms`, `gcp-kms`, `vault`, or `local:fallback:<backend>` |
| `algorithm`    | string  | Signing algorithm: `ed25519` or `ecdsa-p256`. **Absent or empty MUST be treated as `ed25519`** for backward compatibility with bundles written before this field existed. |
| `signed_at`    | string  | RFC 3339 timestamp of the signing operation (writers SHOULD use RFC 3339 nano-precision)        |

Newly-signed bundles emitted by the local signer carry `algorithm="ed25519"`, `backend="local"`, and a `signed_at` timestamp explicitly. Bundles signed before these fields were emitted continue to verify: the verifier defaults `algorithm` to `ed25519` and treats absent `backend`/`signed_at` as legacy/implicit local.

#### Reserved additive extensions (non-breaking)

The following optional fields are tracked for a future minor release:

| JSON key       | Type    | Description                                                                                     |
|----------------|---------|-------------------------------------------------------------------------------------------------|
| `signed_over`  | string  | Hex SHA-256 of the pre-signature bundle bytes (already the implicit pre-image; made explicit)   |

The signing policy for the bundle (which backends are permitted, minimum signatures required) will be declared in a `signing_policy` field added to the manifest in schema version 2.

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
- `ai.human.approval`
- `ai.job.scheduled`, `ai.job.started`, `ai.job.step`, `ai.job.completed`

The built-in profile templates currently evaluate the following required event sets:

- `atb.profile.privileged_tool_action` requires `atb.bundle.manifest`, `ai.request.received`, `ai.action.precommit`, `ai.policy.decision`, `ai.action.executed`, and `ai.action.committed`. `ai.human.approval` is warning-level evidence and is evaluated when actions execute.
- `atb.profile.rag_answer` requires `atb.bundle.manifest`, `ai.request.received`, `ai.model.invoked`, and `ai.model.output`. `ai.retrieval.executed`, `ai.policy.decision`, and `ai.response.sent` are warning-level evidence only.
- `atb.profile.data_export` requires `atb.bundle.manifest`, `ai.request.received`, `ai.policy.decision`, `ai.action.precommit`, `ai.action.executed`, and `ai.action.committed`. `ai.human.approval` is warning-level evidence and is evaluated when exports execute. `data.export.*` event types remain the taxonomy for export-specific lifecycle records, and the current built-in template evaluates the `ai.action.*` control-plane flow.
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

### 5.1 Reader API contract

The implementation exposes two read contracts:

- `Load` parses bundle NDJSON without validating that the file is an ATB
  bundle or that the hash chain is intact. It is for inspection and
  compatibility paths that need to look at bytes that may be malformed.
- `LoadVerified` is the integrity-sensitive gate. It requires a manifest
  record, rejects non-bundle NDJSON, and verifies the hash chain before
  returning bundle data.

The bundle package exposes typed error sentinels for callers that need stable
classification:

| Error | Meaning |
|-------|---------|
| `ErrMalformed` | The bundle structure cannot be parsed or contains an unsupported manifest shape. |
| `ErrNoManifest` | A validating operation required a manifest record but found no records. |
| `ErrTamper` | The hash chain, sequence numbering, or previous-hash linkage failed verification. |
| `ErrNotABundle` | The file parses as NDJSON but record 0 is not an ATB manifest. |
| `ErrBundleLocked` | Another process currently holds the advisory writer lock for the bundle. |

---

## 6. Storage

### 6.1 Local Storage

The default storage location is `run.atb/bundle.atb` relative to the current working directory. This directory should be excluded from version control (`.gitignore`).

`Save` and `SignTo` write through the same durability pattern: acquire the
bundle advisory lock, serialise the complete result to a temporary file, fsync
that file, atomically rename it into place, and fsync the parent directory.
This gives crash-safe completed writes and prevents concurrent writers from
interleaving records in the same bundle.

### 6.2 Optional Encrypted Payloads

ATB supports optional client-side bundle encryption via `atb encrypt` / `atb decrypt`.

- Cipher: AES-256-GCM
- Wire format: `ATBE` magic + version + salt + nonce + auth tag + ciphertext
- Key derivation: PBKDF2-SHA256 with versioned parameters
  - `0x01`: PBKDF2-SHA256 (`100000` iterations). Retained for decrypt compatibility with existing encrypted bundles.
  - `0x02`: PBKDF2-SHA256 (`600000` iterations). Default for newly encrypted bundles.

`atb decrypt` accepts both wire-format versions. `atb encrypt` writes version `0x02`.

Push transport behaviour is documented separately in `docs/spec/bundle-push.md`. See [the push spec](./spec/bundle-push.md). That document is outside the frozen v1.0 local storage contract.

### 6.3 CLI exit codes

| Constant | Code | Meaning |
|----------|------|---------|
| `exitSuccess` | `0` | Success. |
| `exitUserError` | `1` | User/input error, including bad flags, missing local files, or invalid operator input. |
| `exitIntegrityFailure` | `2` | Bundle integrity verification failure. |
| `exitVerifyFailure` | `3` | Profile verification failure. |
| `exitSystemError` | `3` | System/runtime failure. This intentionally shares code `3` with `exitVerifyFailure` for compatibility. |
| `exitLockContention` | `9` | Bundle lock contention; downstream automation should retry after a short delay. |

Contention-sensitive commands that write a bundle (`atb sign`, `atb snapshot`, `atb capture run`, and `atb append`) accept `--lock-wait <duration>`. The default is `0`, which preserves the fail-fast behaviour: a held bundle lock exits with code `9` immediately. When the duration is greater than zero, the command retries advisory lock acquisition until the duration elapses.

Snapshot-related commands classify `appendSnapshot` failures through
`snapshotExitCode`. `atb snapshot`, `atb capture run`, and `atb import chatlog`
therefore map snapshot validation, bundle load, integrity, system, and lock
contention errors through the same exit-code table when snapshotting is
enabled.

The local `snapshot`, `capture run`, `import chatlog`, and `verify` paths wrap
bundle file operations with a default five-minute context timeout. Cancellation
or timeout is reported through the same exit-code classification as the
underlying operation.

The same setting may be supplied through `ATB_LOCK_WAIT`, for example:

```text
ATB_LOCK_WAIT=10s atb snapshot ci_checkpoint
```

If a process crashes while holding the lock, the `.lock` sidecar file may remain. Manual removal of `<bundle>.lock` clears that stale sidecar file; remove it only after confirming no ATB writer is still running.

### 6.4 Capture run environment contract

`atb capture run` wraps a child process and exposes the capture context to it through environment variables. Three variables are exported under the default `ATB` prefix:

- `ATB_BUNDLE_PATH` — absolute path to the bundle the child should append to.
- `ATB_CAPTURE_RUN_ID` — opaque, per-invocation identifier; a fresh value is generated for every `capture run`.
- `ATB_CAPTURE_MODE` — current capture mode marker (typically the string `run`).

When `--env-prefix <NAME>` is supplied, the same three variables are *additionally* exported under the supplied prefix (for example, `MYAPP_BUNDLE_PATH`); the default `ATB_*` exports are always present alongside.

The wrapper exits with the child's exit status. If the child terminates due to a signal, `atb capture run` exits `128 + signal`, the standard POSIX convention, without remapping. A non-zero child exit suppresses any `--snapshot` event that would otherwise have been appended on success.

---

## 7. KMS and remote signing

The default signing backend is local Ed25519. `https-http` delegates signing to
a remote HTTP signer that returns the signature and public verification key.

Tagged CLI builds can include concrete KMS clients:

- `aws-kms`, built with `-tags awskms`
- `gcp-kms`, built with `-tags gcpkms`
- `vault`, built with `-tags vault`

The Vault backend reads the Transit key type from `/v1/transit/keys/<name>` and selects `algorithm="ed25519"` or `algorithm="ecdsa-p256"` accordingly; other key types are rejected. If the Vault response omits the `type` field (older API versions), the signer falls back to `ecdsa-p256` and emits a stderr warning so signing still succeeds with documented provenance.

These backends sign the 32-byte SHA-256 pre-signature bundle digest and embed
the returned public verification key in the bundle signature record, so
verification does not require a live KMS call. The Go verifier supports
`algorithm="ecdsa-p256"` as well as Ed25519, including the DER and uncompressed
P-256 public-key encodings emitted by the supported backends. AWS KMS and GCP
KMS use ECDSA P-256 because native Ed25519 is not uniformly available across
the target services. Signing remains operationally dependent on a build that
includes the relevant backend tag, provider credentials and network access,
and a compatible asymmetric signing key; those dependencies do not apply when
verifying a bundle whose signature record embeds its public key.

---

## 8. Schema Evolution (v1.0+)

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

## 9. Schema versioning

The manifest `version` field governs bundle compatibility. The default writer
format remains **1**. Version **1** stores manifest `data` as a JSON-encoded
string and is retained for compatibility with existing bundles. Version **2**
is the opt-in structured-object manifest format. The product
CLI/SDK SemVer policy is separate and lives
in `VERSIONING.md`; this section governs the on-disk bundle format only.

**Breaking change (requires manifest version bump):**

- Adding, removing, or renaming a field on `Event` that is included in the canonical hash input (see §3.4)
- Changing the hash algorithm or canonicalisation rule
- Changing the pre-image for bundle signatures
- Changing how the manifest `data` field is encoded

**Non-breaking (no manifest version bump required):**

- Adding optional fields to any event's `data` payload (readers MUST ignore unknown fields)
- Adding new event types to the registry
- Adding new obligation profiles

**Float serialisation note:** A canonicalisation change was made in v1.1.2 that altered the serialisation of floating-point values where `|f| >= 1e21` or `0 < |f| < 1e-6` (these now use exponential form). Bundles written before v1.1.2 that contain such values will not re-verify under the current implementation. These are considered distinct format revisions; the manifest version was not bumped at the time. A future v2 manifest will formally separate these.

This document describes ATB specification version **1.0**. Future versions will be backwards-compatible unless a major version increment is made.
