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

## What runs

1. **BackgroundJobTracker** — schedules and runs a nightly analytics export job.
2. **DataExportGate** — records the export profile event sequence and verifies `atb.profile.data_export` on a checkpoint bundle.
3. **PolicyDecisionRecorder** — records a retention exception decision on the continuing bundle.
4. **HumanOverrideGate** — records a supervisor-approved legal-hold export path.

The script verifies `data_export` before appending policy/override events so unrelated `ai.policy.decision` records do not violate export profile relations.

## SDK surface note (v1.13.0)

Python `PolicyDecisionRecorder` defaults omit `policyId` / `policyVersion` in the policy payload helper, while TypeScript defaults include them. Both emit valid events; align defaults in a future minor release.
