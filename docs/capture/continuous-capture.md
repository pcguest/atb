# Continuous capture with `atb intercept` (external pilot)

This is an **external pilot**, not a production-ready product. It describes the
single continuous-capture path ATB supports today and exactly what it does and
does not prove.

## What it is

`atb intercept` is a local HTTPS forward proxy. Traffic that a workload routes
through it (via `HTTPS_PROXY` and the local capture CA) is observed as it
happens and recorded into a manifest-v3 ATB bundle. Each observation is:

1. durably journalled (append-only NDJSON, fsync per entry),
2. materialised as live ATB evidence (`acquisition.mode = "live"`),
3. committed to the bundle (durable save), then
4. acknowledged by advancing the acquisition checkpoint bound to the committed
   bundle head.

If the collector stops between steps 1 and 3, startup recovery replays the
durable-but-uncommitted observations idempotently, so an observation is never
silently lost. A healthy process is not evidence of complete capture; the
capture state is reported separately (see *Inspect capture health*).

## Supported workload shape

- Any workload that routes provider API traffic through an HTTP(S) forward proxy
  and trusts the local capture CA (`SSL_CERT_FILE`, `CURL_CA_BUNDLE`,
  `NODE_EXTRA_CA_CERTS`).
- Provider hosts in `--target` (default `openai`, `anthropic`). Clients that
  ignore `HTTPS_PROXY` or pin certificates bypass capture; use an SDK wrapper for
  those calls.
- Request/response bodies are **digested by default**. Raw bodies are recorded
  only with `--capture-bodies` (see *What is captured*).

## Start

```bash
# First run creates ~/.atb/ca.crt and ~/.atb/ca.key (mode 0600).
atb intercept --bundle ./capture.atb --target openai,anthropic \
  --source-incarnation prod-acme-2026-01

# Route the workload through the proxy in its own environment:
export HTTPS_PROXY=http://127.0.0.1:8080
export SSL_CERT_FILE=$HOME/.atb/ca.crt
export NODE_EXTRA_CA_CERTS=$HOME/.atb/ca.crt
```

### Source incarnation

`--source-incarnation <token>` is an opaque, non-secret label for the specific
deployment/instance of the proxied source. It does not authenticate the source.
- Omit it to reuse the checkpoint's incarnation (a clean restart resumes), or to
  generate one on first run.
- Supplying a token that differs from the checkpoint's incarnation **fails
  startup** rather than silently resuming the wrong source. To start a genuinely
  new incarnation against an existing bundle, use a new bundle path (or remove
  the checkpoint directory).

## Where state lives

- ATB evidence: the bundle at `--bundle`.
- Observation journal: `<bundle-dir>/.atb/capture/<bundle-name>.journal.ndjson`.
- Acquisition checkpoint: `<bundle-dir>/.atb/checkpoints/…checkpoint.json`.

The journal and checkpoint are **operational state**, not evidence. The bundle
is the evidence.

## Inspect capture health

```bash
atb capture status --bundle ./capture.atb
atb capture status --bundle ./capture.atb --format json
```

This reads state offline and separates:

- **process_health** — always `unknown` offline; a running process is never
  reported as complete capture.
- **capture_state** — `healthy`, `recovery_required`, `degraded`, `unknown`, or
  `not_established`.
- **journal backlog** — durable observations not yet committed to evidence.
- **known_gap / possible_unknown_gap** — bounded gap disclosure. ATB does not
  invent missing-event counts it cannot prove.

## Restart and recovery

Restart `atb intercept` with the same `--bundle` (and the same incarnation, by
omission or by passing the same token). Startup validates the checkpoint source,
incarnation, adapter version, and bundle binding, then replays any durable but
uncommitted observations. If continuity cannot be established, startup fails
closed with a specific error; the capture state will show `degraded` or
`recovery_required`.

## Investigate an incident

```bash
atb verify --bundle ./capture.atb
atb inspect --bundle ./capture.atb --json      # full records incl. acquisition provenance
atb capture status --bundle ./capture.atb
atb capture handoff --bundle ./capture.atb --seq <n>   # portable handoff (no Mortise)
atb export --bundle ./capture.atb              # offline export
```

## Verify offline

Capture, Mortise, and any paid tooling are optional for verification. With the
capture process stopped and no network, `atb verify`, `atb inspect`, and
`atb export` operate on the bundle alone.

## What is captured / not captured

Captured (per observed exchange, subject to `--target` and proxy routing):
request/response metadata, provider/model, usage, tool-call names and digested
tool arguments, capture-scope attestation, capture rejections, and session
close. Request/response **bodies are digested by default** (`body_sha256`,
`body_bytes`); raw bodies are stored only with `--capture-bodies`.

Not captured: provider traffic that bypasses the proxy; in-process SDK calls;
raw credential headers (allowlisted headers only); query strings; content the
provider never sent through the proxy.

## Secrets

- Credential headers (`Authorization`, `X-Api-Key`, `Cookie`, …) are excluded by
  an allowlist and never recorded.
- Unresolved actors are labelled with the constant sentinel `unresolved`; no
  credential material (not even a suffix) enters evidence, the journal, logs, or
  the checkpoint.
- Do **not** pass secrets on the command line. `--identity-map key=name` puts a
  raw API key on argv and persists it in `~/.atb/identity-map.yaml`; prefer the
  identity environment/chain. `--capture-bodies` stores raw prompts,
  completions, and tool payloads; apply access controls and retention limits.

## Remove the pilot

Stop `atb intercept`, remove the trust environment variables from the workload,
delete `~/.atb/ca.crt` and `~/.atb/ca.key`, and delete the bundle plus its
`.atb/` directory.

## Known limitations

- Coverage is bounded by proxy routing; ATB cannot see traffic that bypasses it.
- The source incarnation is operator-supplied and is not authenticated.
- Source incarnation is recorded in operational state (checkpoint/status), not
  in the canonical evidence envelope; promoting it to portable evidence requires
  a manifest-version migration and is not done in this pilot.
- The journal is local operational state with no rotation or remote shipping.
- ATB proves recorded integrity, not truth, completeness, or causation.
