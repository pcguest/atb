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
durable-but-uncommitted observations idempotently, so a **durable** observation is
never silently lost. A journal write that itself fails is different: the
observation never became durable, and the collector records a best-effort
degradation marker beside the journal so `capture status` reports `degraded` with
`known_gap = true` rather than `healthy`. The marker shares the journal's storage,
so if that storage cannot accept any write (for example total exhaustion),
offline disclosure cannot be guaranteed; the live process still degrades. A
healthy process is not evidence of complete capture; the capture state is
reported separately (see *Inspect capture health*).

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
# Terminal 1: this blocks until shutdown (Ctrl-C). First run creates
# ~/.atb/ca.crt and ~/.atb/ca.key (mode 0600).
atb intercept --bundle ./capture.atb --target openai,anthropic \
  --source-incarnation prod-acme-2026-01
```

In the workload's own environment (Terminal 2, or the workload's service
definition), route traffic through the proxy:

```bash
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
  new incarnation, use a **new bundle path**; deleting only the checkpoint
  directory would leave the journal and bundle in place and resume the same
  observation stream under a fresh incarnation.

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
- **observation_currency** — `none` when nothing was ever observed, otherwise
  `unknown`; an offline read cannot assert that observation is live.
- **capture_state** — `healthy`, `recovery_required`, `degraded`, `unknown`, or
  `not_established`. `healthy` is a static consistency verdict, not proof of
  live or complete observation.
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
atb export --format soc2 --bundle ./capture.atb --output ./capture-export.zip
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
  raw API key on argv, where it may be exposed by process listings and shell
  history depending on how the command is entered; prefer the identity
  environment/chain. `--capture-bodies` stores raw prompts,
  completions, and tool payloads; apply access controls and retention limits.

## Remove the pilot

Stop `atb intercept`, remove the trust environment variables from the workload,
delete `~/.atb/ca.crt` and `~/.atb/ca.key`, and delete the bundle plus its
`.atb/` directory.

## Operational envelope (pilot)

The supported pilot envelope is deliberately small:

- **one** local/customer-operated `atb intercept` process per bundle;
- **one** supported workload shape routed through the proxy;
- **one** journal and **one** evidence stream per bundle;
- bounded pilot duration and bounded storage (see below);
- **no** journal rotation, remote shipping, hosted control plane, multi-region,
  arbitrary connector support, or Action execution.

Bodies larger than `--max-body-bytes` (default 32 MiB, hard limit 256 MiB) are
rejected and recorded as a privacy-safe rejection event rather than buffered.
The journal grows without rotation for the life of the pilot; size the pilot
duration and disk accordingly, or restart with a new bundle path at a safe
boundary.

**Observation volume.** The commit protocol currently re-writes the whole
bundle on each observation, so commit cost grows with the bundle size (measured
on the pilot host: ~2,000 observations took ~2m20s to commit; cost per
observation rises roughly linearly with the number of committed records). The
pilot envelope is therefore **bounded to a few thousand observations per
bundle**. Beyond that, restart with a new `--bundle` path at a safe boundary
(e.g. per session). Incremental (append-only) bundle commits are the planned
remediation before higher-volume pilots.

## If the journal is lost

The journal is operational state, not evidence. If the journal (or the journal
and checkpoint together) is lost while the bundle still holds live acquisition
evidence, startup **fails closed** and `capture status` reports `degraded` with
`possible_unknown_gap = true`. ATB does not regenerate an empty journal and
declare success, and it does not invent missing-event counts. Committed evidence
already in the bundle remains verifiable with `atb verify` / `atb inspect`; what
cannot be re-established is whether durable-but-uncommitted observations were
lost.

## If credentials expire

`atb intercept` itself needs no provider credential: it observes traffic the
workload sends. If the workload's credential expires, the provider returns a 4xx
response, which is captured as an ordinary exchange; capture continuity is
unaffected. Do not pass provider credentials to ATB on the command line.

## Known limitations

- Coverage is bounded by proxy routing; ATB cannot see traffic that bypasses it.
- **Late start:** observations begin when `atb intercept` starts; activity
  before that is not observed, and ATB does not claim it covered the whole
  session. `capture status` reports `not_established` until the first
  observation and never asserts complete coverage.
- **Source time:** `acquisition.source_timestamp` is adapter-dependent. The
  intercept adapter records the observing clock; it is not a provider-original
  timestamp. Recorded order (`seq`) is append order, not causal order.
- The source incarnation is operator-supplied and is not authenticated.
- Source incarnation is recorded in operational state (checkpoint/status), not
  in the canonical evidence envelope; promoting it to portable evidence requires
  a manifest-version migration and is not done in this pilot.
- The journal is local operational state with no rotation or remote shipping.
- Commit cost grows with bundle size (the bundle is re-written per observation);
  the pilot is bounded to a few thousand observations per bundle. Restart with a
  new bundle path beyond that; incremental commits are planned.
- Event data is encoded per RFC 8785 JSON canonicalisation. Integer values above
  2^53 are not exactly representable and may be normalised; tool-argument digests
  preserve the source integer literals, but do not rely on ATB to round-trip
  arbitrary large integers in event `data`.
- ATB proves recorded integrity, not truth, completeness, or causation.
