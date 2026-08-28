#!/usr/bin/env bash
# Smoke-test atb view API endpoints against a running viewer session.
# Usage:
#   atb view --bundle examples/quickstart/run.atb/bundle.atb \
#     --profile atb.profile.policy_decision --no-open --port 18890 &
#   bash scripts/smoke-view.sh http://127.0.0.1:18890 '<session-token-from-stderr>'
set -euo pipefail

BASE="${1:-http://127.0.0.1:18890}"
TOKEN="${2:-}"

if [[ -z "$TOKEN" ]]; then
  echo "usage: smoke-view.sh <base-url> <session-token>" >&2
  exit 1
fi

HDR=(-H "X-ATB-Session-Token: $TOKEN")

assert_json() {
  local path="$1"
  local body
  body=$(curl -sf "${HDR[@]}" "$BASE$path")
  python3 -c "import json,sys; json.load(sys.stdin)" <<<"$body" >/dev/null
  echo "$body"
}

echo "→ GET /api/v1/verification"
VERIFY=$(assert_json "/api/v1/verification")
python3 - <<'PY' "$VERIFY"
import json, sys
v = json.loads(sys.argv[1])
assert v.get("status") == "valid", v
assert v.get("chain_length", 0) >= 10, v
print(f"  status=valid chain_length={v['chain_length']}")
PY

echo "→ GET /api/v1/bundle/meta"
META=$(assert_json "/api/v1/bundle/meta")
python3 - <<'PY' "$META"
import json, sys
m = json.loads(sys.argv[1])
assert m.get("event_count", 0) >= 10, m
assert m.get("verified") is True, m
print(f"  event_count={m['event_count']} verified={m['verified']}")
PY

echo "→ GET /api/v1/bundle/events?limit=5"
EVENTS=$(assert_json "/api/v1/bundle/events?limit=5")
python3 - <<'PY' "$EVENTS"
import json, sys
e = json.loads(sys.argv[1])
assert e.get("total", 0) >= 10, e
assert len(e.get("events", [])) >= 1, e
print(f"  total={e['total']} first_type={e['events'][0]['type']}")
PY

echo "→ GET /api/v1/bundle/profile"
PROFILE=$(assert_json "/api/v1/bundle/profile")
python3 - <<'PY' "$PROFILE"
import json, sys
p = json.loads(sys.argv[1])
assert p.get("pass") is True, p
assert p.get("profile_id"), p
assert p.get("cas_grade"), p
print(f"  profile={p['profile_id']} pass={p['pass']} grade={p['cas_grade']}")
PY

echo "→ GET /api/v1/bundle/verify/report"
REPORT=$(curl -sf "${HDR[@]}" "$BASE/api/v1/bundle/verify/report")
python3 - <<'PY' "$REPORT"
import json, sys
r = json.loads(sys.argv[1])
assert r.get("report_version") == "verify.report.v1", r
assert r.get("pass") is True, r
assert r.get("sub_scores"), r
print(f"  report_version={r['report_version']} pass={r['pass']} xc={r['sub_scores'].get('XC')}")
PY

echo "✓ smoke-view passed"
