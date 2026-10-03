# ATB end-user validation harness

A reproducible, bounded test of the ATB product experience — not the unit suite.
It acts as a technically competent user who did not build ATB: import a saved
chatlog, verify integrity, inspect provenance, derive and resolve a semantic
evidence locator, and probe bounded failure states.

## Run

```bash
# Needs python3 and an `atb` binary (ATB_BIN, PATH, or ../../atb).
harnesses/end-user/run.sh --atb /path/to/atb --out harnesses/end-user/atb-enduser.report.json
```

Exit code 0 means every required step passed. The JSON report conforms to
`../report-schema.json`.

## What it proves

| Step | Public surface |
| --- | --- |
| A1 discover | `atb --help`, README, chatlog-import docs |
| A2 capture | `atb import chatlog --from generic-jsonl` |
| A3 verify | `atb verify --format json` (`chain_valid`) |
| A4 inspect | `atb inspect --json` (timeline + acquisition provenance) |
| A5 locator | viewer API `GET /api/v1/bundle/locate` (identity, not integrity) |
| A6 failures | malformed locator, unknown head, changed source representation |

## Source material

`fixtures/development-session.jsonl` is a **REPRESENTATIVE_FIXTURE** synthesised
from a small, non-sensitive excerpt of the ATB/Mortise development workflow. It
contains no secrets, credentials, or private chat content. The harness never
commits raw source material and writes only to a scratch directory.

## Known friction (recorded, not hidden)

- ATB has no CLI or UI surface that *emits* a semantic locator. The user must
  assemble it from `atb inspect --json` plus the published grammar. The viewer
  resolves locators but does not offer a copy control. (Product hardening.)
- The viewer API requires the local session token (`--session-token`); a
  headless user must know to pass it as `X-ATB-Session-Token`.
