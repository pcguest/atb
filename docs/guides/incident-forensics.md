# Incident forensics

ATB acts as a **local-first flight recorder for AI agents**: it captures what an
agent actually did into a tamper-evident bundle, so when something goes wrong you
can reconstruct it — even if the agent's own logs cannot be trusted. This guide
walks the full path: **capture → discover → review**.

## 1. Capture (intercept)

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

```bash
export PATH="$HOME/go/bin:$PATH"
mkdir -p ~/.atb/sessions
atb intercept --bundle ~/.atb/sessions/$(date +%Y%m%d-%H%M).atb
```

Session bundles can contain sensitive metadata (and raw prompts if
`--capture-bodies` was used). Keep them out of version control; verify locally,
then hand off a closed bundle to custody (for example Mortise via `--mortise`)
rather than committing it to a repository.

By default **bodies are digested, not stored** (`body_sha256` + `body_bytes`);
credential headers are stripped. See [`docs/public-surface.md`](../public-surface.md#atb-intercept).

> **No live traffic?** Generate a fixture bundle:
>
> ```bash
> go run ./examples/bundles/incident-capture/
> ```

> **Need the full Tenon pilot path?** Generate the approved + anomalous
> synthetic session:
>
> ```bash
> go run ./examples/bundles/tenon-pilot/
> ```
>
> Then follow [Tenon pilot walkthrough](./tenon-pilot.md) for verify, incident
> report, viewer, and optional Mortise handoff commands.

> **No provider credentials?** Run the complete SDK oversight demo:
>
> ```bash
> make build
> PYTHONPATH=sdk/python ATB_BIN=./atb python examples/python/agent_incident_demo.py
> ```

## 2. Seal it (optional)

```bash
atb sign --bundle examples/bundles/incident-capture/incident-capture.atb --key atb-key.pem
atb anchor --bundle …   # RFC 3161; network required
```

## 3. Discover sessions

```bash
atb incident list --bundle examples/bundles/incident-capture/incident-capture.atb
```

The `tool_without_approval` anomaly flags a privileged tool with no recorded
human approval earlier in the session.

## 4. Review one session

The example below matches the **unsigned** shipped fixture. After §2, signature
metadata appears; findings are unchanged.

```bash
atb incident report \
  --bundle examples/bundles/incident-capture/incident-capture.atb \
  --session sess-incident-7731
```

Add `--format json` or `--format ndjson` for machine-readable output.

Open the same evidence in the local viewer:

```bash
atb view \
  --bundle examples/bundles/incident-capture/incident-capture.atb \
  --profile atb.profile.privileged_tool_action
```

The sessions page exposes the session list, actor grouping, anomaly badges,
capture/action event families, and schema status.

## 5. SDK append path (without intercept)

When events are appended directly instead of captured via the proxy:

```bash
atb bundle new
atb append ai.request.received --data='{"request_id":"req-1042",…}'
atb append ai.policy.decision --data='{"decision":"deny",…}'
atb snapshot incident_review_failed
atb verify --profile atb.profile.policy_decision --format json
atb compliance pack \
  --bundle run.atb/bundle.atb \
  --profile atb.profile.policy_decision \
  --regime eu-ai-act \
  --out incident-review-evidence.zip
```

See [Quickstart](../quickstart.md) for a full policy-decision example.
For live SDK calls, use `wrap_openai`/`wrap_anthropic` in Python or
`wrapOpenAI`/`wrapAnthropic` in TypeScript. Each wrapper emits an
`atb.capture.scope` event that states its capture boundary.

## What the report proves (and what it doesn't)

- **Integrity PASS** — the recorded sequence was not altered after capture.
- **Signature valid** — attributes the bundle to a signing key (when signed).
- **`tool_without_approval`** — a control was *missing* in the record, not proof of tampering.
- **Reviewer identity evidence present** — an IdP assertion digest was recorded
  without alteration; ATB did not validate the assertion.

A session report scopes review; the **full signed bundle** remains the authoritative
evidence object. Every event row's record hash is checkable against that bundle.

## Honest limits

- The proxy sees provider API traffic routed through it; direct bypass is not captured.
- `ai.action.error` from intercept recognises Anthropic `tool_result` failures;
  OpenAI tool messages carry no standard error flag.
