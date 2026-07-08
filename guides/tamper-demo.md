# Tamper demonstration

This guide shows valid bundle verification, deliberate tampering, and how the
CLI and local viewer respond. It takes about five minutes and requires no network
access.

## 1. Create and verify a bundle

```bash
go build -o ./atb ./cmd/atb   # or use a release binary
rm -rf run.atb
./atb bundle new
./atb append ai.request.received \
  --data '{"request_id":"req-demo","actor_id_hash":"sha256-user-demo","purpose_tag":"tamper-demo"}'
./atb append ai.policy.decision \
  --data '{"policy_id":"pol-demo","policy_version":"1","decision":"allow","decision_reason_codes":["demo"],"subject_id_hash":"sha256-user-demo","action_id":"act-demo"}'
./atb snapshot tamper_demo
./atb verify --profile atb.profile.policy_decision --format json
```

Expected: `"pass": true` and `"integrity"` chain valid when present in full JSON output.

## 2. Tamper with the bundle file

Edit one byte inside `run.atb/bundle.atb` (any event JSON field). For example,
change a character in a `request_id` value after the bundle was written.

```bash
# Example: corrupt the bundle on disk (macOS/Linux)
python3 - <<'PY'
from pathlib import Path
p = Path("run.atb/bundle.atb")
data = bytearray(p.read_bytes())
for i, b in enumerate(data):
    if b == ord("d") and i > 100:
        data[i] = ord("X")
        break
p.write_bytes(data)
PY
```

## 3. Verify again

```bash
./atb verify --profile atb.profile.policy_decision --format json
```

Expected: `"pass": false`, integrity failure, and non-empty `critical_failures` or
integrity error fields depending on tamper location.

Exit code is typically `2` (`exitIntegrityFailure`) for chain breakage.

## 4. Open the local viewer

Rebuild with embedded UI if needed (`cd web && npm ci && npm run build && cd .. && go build -o ./atb ./cmd/atb`).

```bash
./atb view --bundle run.atb/bundle.atb --no-open --port 18080
```

On a **valid** bundle: green verification banner, timeline and inspector load.

On a **tampered** bundle: red `TAMPER DETECTED` banner; event APIs return `403`;
no event payloads are served.

## 5. Restore or regenerate

Regenerate a clean bundle by repeating step 1, or restore from a signed export /
WORM copy if you are testing handoff workflows.

## What this demonstrates

- **L1 integrity**: hash chain detects post-capture edits.
- **Viewer gate**: UI withholds payloads when verification fails.
- **Limitation**: replacing the entire file before load is a host-trust problem;
  pair ATB with WORM export for adversarial environments. See
  [provability-ladder.md](../provability-ladder.md).
