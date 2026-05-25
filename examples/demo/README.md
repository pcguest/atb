# Profile workflow SDK demos

Five-minute scripts demonstrating the v1.12 profile workflow helpers against real bundles, with CLI verification.

## Python

```bash
pip install -e sdk/python
python examples/demo/profile_workflows_demo.py
```

## TypeScript

```bash
cd sdk/typescript && npm install
npx tsx ../../examples/demo/profile_workflows_demo.ts
```

Set `ATB_BIN=/path/to/atb` when the CLI is not on `PATH`.

## Timing (cold cache, script runtime)

Measured on macOS with repo-root `./atb` and dependencies pre-installed:

| Script | Wall time |
|--------|-----------|
| Python (`profile_workflows_demo.py`) | ~1 s |
| TypeScript (`profile_workflows_demo.ts`) | ~5 s (includes cold `npx tsx` on first run) |

First-time setup (venv + `pip install -e sdk/python`, or `npm install` in `sdk/typescript`) adds ~10–15 s. Both scripts finish well under 30 seconds once dependencies are installed.

## What runs

1. **BackgroundJobTracker** — schedules and runs a nightly analytics export job.
2. **DataExportGate** — records the export profile event sequence and verifies `atb.profile.data_export` on a checkpoint bundle.
3. **PolicyDecisionRecorder** — records a retention exception decision on the continuing bundle.
4. **HumanOverrideGate** — records a supervisor-approved legal-hold export path.

The script verifies `data_export` before appending policy/override events so unrelated `ai.policy.decision` records do not violate export profile relations.

## SDK surface note (v1.13.0)

Python `PolicyDecisionRecorder` defaults omit `policyId` / `policyVersion` in the policy payload helper, while TypeScript defaults include them. Both emit valid events; align defaults in a future minor release.
