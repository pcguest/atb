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

## Related paths

- Use SDKs when you can instrument a workflow in-process.
- Use the MCP bridge when a local MCP-compatible toolchain is the right fit.
- Use `atb capture run` when you want a lightweight wrapper that prepares bundle paths and capture environment variables for a child process.
