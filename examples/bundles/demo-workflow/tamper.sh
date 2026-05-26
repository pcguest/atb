#!/bin/bash
# Tamper a middle event payload while leaving the stored hash unchanged (chain failure demo).
set -euo pipefail
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC="${1:-$DIR/demo-workflow.atb}"
OUT="${2:-$DIR/demo-workflow-tampered.atb}"
cp "$SRC" "$OUT"
python3 - "$OUT" <<'PY'
import json, sys
path = sys.argv[1]
with open(path, "r", encoding="utf-8") as f:
    lines = f.read().splitlines()
mid = max(2, len(lines) // 2)
while mid < len(lines) and not lines[mid].strip():
    mid += 1
obj = json.loads(lines[mid])
data = obj.get("event", {}).get("data", {})
if isinstance(data, dict) and data:
    key = next(iter(data))
    val = data[key]
    if isinstance(val, str) and val:
        data[key] = val[:-1] + ("Z" if val[-1] != "Z" else "Y")
    elif isinstance(val, list) and val and isinstance(val[0], str):
        val[0] = val[0][:-1] + ("Z" if val[0][-1] != "Z" else "Y")
    else:
        data[key] = "tampered"
else:
    raise SystemExit("no mutable data field in middle event")
lines[mid] = json.dumps(obj, separators=(",", ":"), ensure_ascii=False)
with open(path, "w", encoding="utf-8") as f:
    f.write("\n".join(lines) + "\n")
print(f"Tampered event line {mid} ({obj.get('event', {}).get('type', '?')}) in {path}")
PY
