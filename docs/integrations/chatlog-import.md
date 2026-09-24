# Chatlog import

Capture v1 adds a narrow, local import path for saved AI chatlogs.

`atb import chatlog` reads a user-supplied file on the local machine and writes canonical ATB events into a local `.atb` bundle. It does not scrape provider accounts, proxy provider APIs, or claim complete capture of a workflow.

## Scope and limits

- Local-only: the importer reads a local file and writes a local bundle.
- User-controlled input: you choose which chatlog file to import.
- No provider scraping: Capture v1 does not log into a provider account or fetch remote history for you.
- No completeness guarantee: imported evidence is limited to what the saved chatlog contains.
- No compliance verdict: profile checks and CAS still describe recorded evidence quality within the declared boundary, not legal compliance or certification.

## Generic JSONL schema

Capture v1 fully implements `--from generic-jsonl`.

Each line in the file must be one JSON object.

Required fields:

- `role`: one of `user`, `assistant`, `system`, `tool`
- `content`: string content for that record
- `timestamp`: RFC 3339 timestamp

Core optional fields:

- `model`: model identifier such as `gpt-4o` or `claude-3-opus`
- `tool_name`: required for `role: "tool"`
- `tool_args`: JSON value describing tool input
- `conversation_id`
- `session_id`

ATB-ready optional fields:

- `request_id`
- `actor_id_hash`
- `purpose_tag`
- `model_provider`
- `model_parameters_digest`
- `prompt_digest`
- `output_digest`
- `output_format`

When these ATB-ready fields are absent, the importer fills bounded defaults so the imported event shape remains canonical:

- `request_id`: deterministic per imported user turn
- `actor_id_hash`: deterministic local surrogate derived from request context
- `purpose_tag`: `chatlog_import`
- `model_provider`: guessed from the model name when possible, otherwise `unknown`
- `model_parameters_digest`: digest of an empty JSON object when no parameter record is present
- `prompt_digest` and `output_digest`: derived from imported content
- `output_format`: `text/plain`

## Example chatlog

```jsonl
{"role":"system","content":"Use the HR handbook and answer plainly.","timestamp":"2026-04-24T09:00:00Z","session_id":"sess-hr-001"}
{"role":"user","content":"Can I carry annual leave into next year?","timestamp":"2026-04-24T09:00:10Z","session_id":"sess-hr-001","request_id":"req-hr-001","actor_id_hash":"sha256:user-hr-001","purpose_tag":"rag_answer"}
{"role":"assistant","content":"I will check the handbook.","timestamp":"2026-04-24T09:00:11Z","session_id":"sess-hr-001","model":"gpt-4o-mini"}
{"role":"tool","content":"{\"policy\":\"Up to five days may be carried over with manager approval.\"}","timestamp":"2026-04-24T09:00:12Z","session_id":"sess-hr-001","tool_name":"hr.policy.lookup","tool_args":{"query":"annual leave carry over"}}
{"role":"assistant","content":"Yes. Up to five days may be carried over with manager approval.","timestamp":"2026-04-24T09:00:13Z","session_id":"sess-hr-001","model":"gpt-4o-mini"}
```

The repository includes this example at [`testdata/chatlog.jsonl`](../../testdata/chatlog.jsonl).

## Event mapping

For `generic-jsonl`, Capture v1 maps chatlog records into canonical ATB events as follows:

- first and subsequent imported user turns -> `ai.request.received`
- assistant turns with a `model` field -> `ai.model.invoked`, `ai.model.output`, `ai.response.sent`
- assistant turns without a `model` field -> `ai.response.sent`
- tool records -> `ai.tool.exec`
- system records -> used to build prompt context digests, but not emitted as standalone canonical events in Capture v1

Illustrative `atb append` equivalents for the example above:

```bash
atb append ai.request.received --data='{"request_id":"req-hr-001","actor_id_hash":"sha256:user-hr-001","purpose_tag":"rag_answer","input_digest":"sha256:...","input_format":"text/plain","session_id":"sess-hr-001"}'
atb append ai.model.invoked --data='{"request_id":"req-hr-001","model_provider":"openai","model_id":"gpt-4o-mini","model_parameters_digest":"sha256:...","prompt_digest":"sha256:...","session_id":"sess-hr-001"}'
atb append ai.model.output --data='{"request_id":"req-hr-001","output_digest":"sha256:...","output_format":"text/plain","session_id":"sess-hr-001"}'
atb append ai.tool.exec --data='{"request_id":"req-hr-001","tool_name":"hr.policy.lookup","tool_args":{"query":"annual leave carry over"},"tool_args_digest":"sha256:...","tool_output_digest":"sha256:...","session_id":"sess-hr-001"}'
atb append ai.response.sent --data='{"request_id":"req-hr-001","output_digest":"sha256:...","output_format":"text/plain","session_id":"sess-hr-001"}'
```

## Usage

Create or reuse a local bundle and import the chatlog:

```bash
atb import chatlog --from generic-jsonl --input testdata/chatlog.jsonl
```

Imports are capped at 256 MiB by default. Files larger than this limit are
rejected before any records are written. Use `--max-input-size <bytes>` to
override the cap, or split the input file first.

Import and label the end state with a snapshot:

```bash
atb import chatlog \
  --from generic-jsonl \
  --input testdata/chatlog.jsonl \
  --snapshot imported_chatlog
```

Use `--bundle <path>` when you want a bundle path other than `run.atb/bundle.atb`.

## Verify the imported bundle

For the example above, the imported evidence is suitable for the built-in RAG profile:

```bash
atb verify --bundle run.atb/bundle.atb --profile atb.profile.rag_answer --format json
```

`pass: true` means the imported chain is intact and the selected profile found its required recorded evidence. It does not mean the source provider recorded every relevant step or that the workflow was captured completely.

## Acquisition continuity, source identity, and the source digest

Capture v1 records *where* each imported event came from so a later re-import can be related to the earlier one. This section defines that contract precisely enough to reproduce the digest.

### Source record identity

For `generic-jsonl`, a **source record is one exchange**: a `user` turn together with the `assistant` and `tool` turns that follow it, up to the next `user` turn or the end of the file. `system` turns are not part of any exchange.

`source_record_id` is the exchange identifier:

- the `request_id` field of the opening `user` turn, when present; otherwise
- a deterministic surrogate `req-<8hex>-<NNN>`, where `<8hex>` is the first 4 bytes (8 hex characters) of `SHA-256(namespace)` and `<NNN>` is the 1-based index of the `user` turn in the file.

`namespace` is the first non-empty `conversation_id` or `session_id` in the file, otherwise the literal `chatlog`. A surrogate id is positional: inserting an earlier `user` turn shifts the `<NNN>` of every later exchange. Supply `request_id` in the source when stable identity across edits matters.

### Source digest

`source_digest` is `sha256:` + lowercase hex `SHA-256` over the **exact raw text** of every line in the exchange, in file order, joined by a single line feed (`"\n"`, no trailing newline).

| Aspect | Definition |
|--------|------------|
| Input | The raw JSON text of each `user`/`assistant`/`tool` record in the exchange. |
| Per-line bytes | The physical input line with leading/trailing whitespace trimmed. For a line that is a JSON array (`[{...},{...}]`), each element is a record and its raw element text is used (the surrounding `[` `]` and inter-element commas are not digested). |
| Join | `"\n"` between records, in file order; no trailing separator. |
| Encoding | UTF-8 bytes of the raw text. |
| Normalisation | None. Field order, interior whitespace, escaping, and numeric formatting are all significant. Only the outer whitespace of each physical line is trimmed. The per-field `strings.TrimSpace` applied when building mapped events does **not** apply to the digest. |
| Algorithm | SHA-256, lowercase hex, prefixed `sha256:`. |
| Excluded | Blank lines; `system` turns; the JSON array framing for array-form input. |

Reproduction: extract the exchange's records as raw text, join with `"\n"`, SHA-256 the UTF-8 bytes, prefix `sha256:`.

### What digest equality and inequality mean

- **Equal** `source_digest` for the same `source_record_id` ⇒ the recorded raw text of the exchange is byte-identical between the two observations. The re-import is treated as `UNCHANGED` and appends no duplicate evidence.
- **Unequal** `source_digest` for the same `source_record_id` ⇒ the observed raw text differs (`CHANGED`). Exactly one bounded `atb.acquisition.finding` (`source_record_changed`) is recorded, carrying the previous and current digests.

A changed digest establishes only that **the representation ATB observed differs**. It does **not** establish that the content is false, that any change was malicious or unauthorised, or which version is correct. Reformatting, re-wrapping, or a corrected re-export is sufficient to change the digest. Neither result establishes that unobserved records are unchanged or that the source was captured completely.

### Source digest vs. bundle integrity

The `source_digest` fingerprints an **external source file**; it is not the bundle's hash chain. It is stored inside an event's acquisition metadata, which is part of the record's hashed content, so tampering with a stored `source_digest` breaks the hash chain like any other record edit. `atb verify` proves the bundle is intact and ordered; the source digest lets a later import recognise that the *external* source changed since it was last observed. Neither proves the source was complete, true, or unmodified before it was read.

### Flags

```bash
# Re-import to detect source changes; unchanged records are skipped.
atb import chatlog --from generic-jsonl --input edited.jsonl --reconcile

# Continue an interrupted acquisition, validated against the saved checkpoint.
atb import chatlog --from generic-jsonl --input edited.jsonl --continue

# Override the checkpoint location (default: <bundle dir>/.atb/checkpoints/).
atb import chatlog --from generic-jsonl --input edited.jsonl --checkpoint ./cp.json
```

Checkpoints are written atomically and validated on `--continue`/`--reconcile`: a checkpoint whose source system, stream, or adapter does not match the current import is rejected rather than silently resumed. An explicit `--checkpoint` path is an intentional local operator-selected path; checkpoint files are operational state, not evidence and are not exposed through a remote import API.

Reconciliation builds on the acquisition provenance already recorded in the bundle, so ATB verifies the existing bundle's hash chain before appending. If the bundle does not verify, `--reconcile`/`--continue` fails explicitly rather than reconciling onto an unverified chain:

```bash
atb verify --bundle run.atb/bundle.atb   # integrity is always the authority
atb import chatlog --from generic-jsonl --input edited.jsonl --reconcile
```

A plain import (without `--reconcile`/`--continue`) still appends without requiring the prior bundle to verify, for tooling that inspects or extends partial bundles. ATB never repairs a broken chain; a bundle that failed verification before the import still fails verification after it.

## Related paths

- Use SDKs when you can instrument a workflow in-process.
- Use the MCP bridge when a local MCP-compatible toolchain is the right fit.
- Use `atb capture run` when you want a lightweight wrapper that prepares bundle paths and capture environment variables for a child process.
