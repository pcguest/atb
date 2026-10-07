# Continuous-capture pilot kit (`atb intercept`)

This is the packaging and onboarding guide for an **external technical pilot** of
the `atb intercept` continuous-capture path. It is not a production product. The
authoritative description of what capture does and does not prove is
[Continuous capture with `atb intercept`](../capture/continuous-capture.md); this document
says what a design partner receives and how to use it.

## What you receive

A kit produced from a tagged ATB release (or, failing that, a checksummed build
from a pinned commit). It contains:

| Item | Purpose |
| --- | --- |
| `atb` (or `atb.exe`) for your OS/arch | The CLI, including `atb intercept`, `capture status`, `capture handoff`, `verify`, `inspect`, `export` |
| `checksums.txt` | SHA-256 of every file in the kit |
| `PILOT-COMMIT` | The exact commit the kit was built from |
| `continuous-capture.md` | The authoritative pilot doc (what is captured, limits) |
| `PILOT-README.md` | This onboarding guide |

Verify the kit before use:

```bash
# Linux
sha256sum -c checksums.txt
# macOS
shasum -a 256 -c checksums.txt
```

## Install

`atb` is a single static binary; there is nothing else to install.

```bash
install -m 0755 atb /usr/local/bin/atb   # or place it on PATH
atb version
```

`atb intercept` needs no bundled assets. On first run it creates a local capture
CA at `~/.atb/ca.crt` and `~/.atb/ca.key` (key mode `0600`).

## Start capture

```bash
# Blocks until Ctrl-C. First run creates the local capture CA.
atb intercept --bundle ./capture.atb --target openai,anthropic \
  --source-incarnation <your-deployment-token>
```

Route the workload's provider traffic through the proxy (in the workload's own
environment):

```bash
export HTTPS_PROXY=http://127.0.0.1:8080
export SSL_CERT_FILE=$HOME/.atb/ca.crt
export NODE_EXTRA_CA_CERTS=$HOME/.atb/ca.crt
export CURL_CA_BUNDLE=$HOME/.atb/ca.crt
```

Only traffic that honours `HTTPS_PROXY` and trusts the CA is captured. Clients
that pin certificates or bypass the proxy are not seen; use an SDK wrapper for
those calls.

## Check capture health

```bash
atb capture status --bundle ./capture.atb
atb capture status --bundle ./capture.atb --format json
```

Read `capture_state` (`healthy`, `recovery_required`, `degraded`, `unknown`,
`not_established`), `observation_currency`, and the gap flags. `healthy` is a
static consistency verdict, not proof that observation is live or complete.

## Investigate and verify offline

Capture, Mortise, and any paid tooling are optional for verification. With the
capture process stopped and no network:

```bash
atb verify --bundle ./capture.atb
atb inspect --bundle ./capture.atb --json
atb capture status --bundle ./capture.atb
atb capture handoff --bundle ./capture.atb --seq <n>
atb export --format soc2 --bundle ./capture.atb --output ./capture-export.zip
```

`atb capture handoff` emits a portable evidence reference
(`atb://evidence/1/<head>?seq=<n>&record=<hash>`) plus the capture state, known
gaps, and limitations, so the exact record can move into a ticket, GRC, or
governance workflow without losing identity or overclaiming completeness.

## Restart and recovery

Restart `atb intercept` with the same `--bundle` (and the same incarnation, by
omission or the same token). Startup validates the checkpoint source,
incarnation, adapter version, and bundle binding, then replays durable but
uncommitted observations. If continuity cannot be established, startup fails
closed and `capture status` shows `degraded`/`recovery_required`.

## Remove the pilot

Stop `atb intercept`, remove the trust environment variables from the workload,
delete `~/.atb/ca.crt` and `~/.atb/ca.key`, and delete the bundle plus its
`.atb/` directory.

## Bounded pilot envelope

- **One** local/customer-operated `atb intercept` per bundle; **one** supported
  workload shape; **one** journal and evidence stream per bundle.
- Bodies bounded by `--max-body-bytes` (default 32 MiB, hard limit 256 MiB).
- **A few thousand observations per bundle.** The commit protocol re-writes the
  bundle per observation, so commit cost grows with bundle size (measured:
  ~2,000 observations ≈ 2m20s). Restart with a new `--bundle` path at a safe
  boundary (e.g. per session) for higher volume.
- No journal rotation, remote shipping, hosted control plane, multi-region,
  arbitrary connector support, or Action execution.

## What ATB proves — and does not

ATB proves the **integrity of what was recorded**, not truth, completeness,
causation, intent, safety, or policy compliance. Absence of a record does not
prove the event did not occur. A digest identifies a captured representation,
not the provider-original object. A source incarnation is an operational token,
not an authenticated origin. Recorded order is append order, not causal order.
See [Continuous capture](../capture/continuous-capture.md) for the full limitations.
