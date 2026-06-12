# Agent incident forensics

ATB acts as a **local-first flight recorder for AI agents**: it captures what an
agent actually did into a tamper-evident bundle, so when something goes wrong you
can reconstruct it — even if the agent's own logs cannot be trusted. This guide
walks the full path: **capture → discover → review**.

## 1. Capture (the recorder)

Run the local capture proxy and route your agent's provider traffic through it
as an HTTPS forward proxy:

```bash
atb intercept --bundle run.atb/bundle.atb --target anthropic,openai
# then, in the agent's environment:
export HTTPS_PROXY=http://127.0.0.1:8080
# trust the proxy's local CA (path printed on first run), e.g.:
export SSL_CERT_FILE=~/.atb/ca.crt        # Python (httpx/requests honour this)
export NODE_EXTRA_CA_CERTS=~/.atb/ca.crt  # Node.js
```

Only hosts named in `--target` are intercepted; CONNECT requests to any other
host are refused, so unrelated traffic does not silently flow through the
recorder.

### Daily capture ritual (macOS / Linux)

For routine local use, keep one dated session bundle per work block:

```bash
export PATH="$HOME/go/bin:$PATH"   # atb from `go install`
mkdir -p ~/.atb/sessions
atb intercept --bundle ~/.atb/sessions/$(date +%Y%m%d-%H%M).atb
```

Session bundles can contain sensitive operational metadata (and raw prompts if
`--capture-bodies` was used). Keep them out of version control; verify and
review them locally with `atb verify` and `atb incident list`, and hand off a
closed bundle to custody (for example a Custos ingest endpoint via `--custos`)
rather than committing it to a repository.

Every request, response, tool call, and failed tool result is recorded into the
bundle as it happens. By default **bodies are digested, not stored** (only
`body_sha256` + `body_bytes`); credential headers are stripped. Pass
`--capture-bodies` only where storing raw prompts/completions is acceptable. See
[`docs/public-surface.md`](../public-surface.md#atb-intercept).

> **Trying it without live traffic?** Generate a representative incident bundle:
>
> ```bash
> go run ./examples/bundles/incident-capture/   # writes incident-capture.atb
> ```
>
> It models an agent that invoked a privileged tool (`delete_user_records`) with
> no human approval, and the tool failed. The rest of this guide uses it.

## 2. Seal it (attribution, optional but recommended)

Sign the bundle so the evidence is attributable to a key:

```bash
atb sign --bundle examples/bundles/incident-capture/incident-capture.atb \
  --key atb-key.pem
```

For long-term legal weight you can also `atb anchor` to an RFC 3161 timestamp
authority (network required).

## 3. Discover the sessions

```bash
atb incident list --bundle examples/bundles/incident-capture/incident-capture.atb
```

```
# Sessions in `examples/bundles/incident-capture/incident-capture.atb`

| Session | Actor | Exchanges | Profile | CAS | Anomalies |
| --- | --- | --- | --- | --- | --- |
| `sess-incident-7731` | agent-support-bot | 1 | atb.profile.privileged_tool_action | Insufficient | tool_without_approval |
```

The `tool_without_approval` anomaly is the lead: an agent ran a privileged tool
with no recorded human approval.

## 4. Review one session

The example below matches the **unsigned** shipped fixture (`incident-capture.atb`).
If you completed §2, the report also shows signature metadata; findings are
unchanged.

```bash
atb incident report \
  --bundle examples/bundles/incident-capture/incident-capture.atb \
  --session sess-incident-7731
```

```
# Incident report — session `sess-incident-7731`

- Integrity (hash chain): **PASS**
- Signature: none
- Actor: agent-support-bot
- Anomalies: **tool_without_approval, action_failed**

## Findings

Each anomaly flag, what it means, and the event(s) that triggered it.

| Severity | Finding | Triggered at | Detail |
| --- | --- | --- | --- |
| HIGH   | Tool call with no preceding approval | seq 3 | A privileged tool was invoked without a human approval recorded earlier in the session. … |
| MEDIUM | Action error recorded                | seq 5 | An ai.action.error was recorded: an attempted action did not complete successfully. … |

## Events

| Seq | Type | Time | Summary | Record hash |
| --- | --- | --- | --- | --- |
| 1 | `atb.llm.request`  | … | POST /v1/messages              | `601a0b59…` |
| 2 | `atb.llm.response` | … | POST /v1/messages → 200        | `3e401192…` |
| 3 | `atb.tool.call`    | … | tool=delete_user_records       | `470c0514…` |
| 5 | `ai.action.error`  | … | action=toolu_delete_1 error_class=failed | `a7b71a78…` |
| 7 | `atb.session.close`| … |                                | `72ab81f8…` |
```

The **Findings** section does the correlation for you: rather than handing you a
bare flag, it names the severity, explains the flag in plain English, and points
at the exact event sequence that triggered it (seq 3 ran the tool with no
approval; seq 5 recorded the failure).

Add `--format json` for a machine-readable report, or `--format ndjson` for one
JSON object per event — each line tagged with the `triggered_flags` it set off,
ready for SIEM ingestion and per-record alerting.

## What the report proves (and what it doesn't)

- **Integrity PASS** proves the recorded sequence was not altered after the fact.
- **Signature valid** attributes the bundle to a signing key (chain of custody).
- **`tool_without_approval`** proves a control was *missing*, not that records
  were tampered with. Integrity and oversight sufficiency are independent.

A bundle's integrity is verified across the **whole** hash chain, so a single
session cannot be carved into an independently verifiable sub-bundle. The full
signed bundle therefore remains the authoritative evidence; the report scopes one
session for review, and **every event row's record hash is checkable against that
bundle**. Hashes shown above are per-bundle and will differ on regeneration.

## Honest limits

- The proxy sees provider API traffic; an agent calling a provider directly,
  bypassing the proxy, is not captured. Completeness is bounded by what flows
  through the recorder.
- `ai.action.error` from capture currently recognises Anthropic `tool_result`
  failures (`is_error: true`); OpenAI tool messages carry no standard error flag.
