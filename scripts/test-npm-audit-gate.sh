#!/usr/bin/env bash
# Deterministic regression proof for scripts/npm-audit-gate.mjs.
#
# Verifies the two-signal contract without requiring a vulnerable lockfile:
#   - runtime HIGH/CRITICAL blocks (exit 1);
#   - full-tree visibility reports HIGH/CRITICAL without blocking (exit 0);
#   - an audit execution failure fails closed (exit 2) in both signals.
set -u

here="$(cd "$(dirname "$0")" && pwd)"
gate="$here/npm-audit-gate.mjs"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

fail=0
check() {
  if [ "$2" != "$3" ]; then
    echo "FAIL: $1 (expected exit $2, got $3)"
    fail=1
  else
    echo "ok: $1 -> exit $3"
  fi
}

cat > "$tmp/high.json" <<'JSON'
{
  "auditReportVersion": 2,
  "vulnerabilities": {
    "example-pkg": {
      "name": "example-pkg",
      "severity": "high",
      "via": [{ "title": "Synthetic high advisory" }],
      "fixAvailable": false
    }
  },
  "metadata": { "vulnerabilities": { "info": 0, "low": 0, "moderate": 0, "high": 1, "critical": 0, "total": 1 } }
}
JSON

cat > "$tmp/clean.json" <<'JSON'
{
  "auditReportVersion": 2,
  "vulnerabilities": {},
  "metadata": { "vulnerabilities": { "info": 0, "low": 0, "moderate": 0, "high": 0, "critical": 0, "total": 0 } }
}
JSON

cat > "$tmp/error.json" <<'JSON'
{ "error": { "code": "ENOLOCK", "summary": "This command requires an existing lockfile." } }
JSON

printf 'not json' > "$tmp/bad.json"

node "$gate" runtime --report "$tmp/high.json" >/dev/null 2>&1
check "runtime HIGH blocks" 1 $?

node "$gate" visibility --report "$tmp/high.json" >/dev/null 2>&1
check "visibility reports HIGH without blocking" 0 $?

node "$gate" runtime --report "$tmp/clean.json" >/dev/null 2>&1
check "runtime clean passes" 0 $?

node "$gate" visibility --report "$tmp/clean.json" >/dev/null 2>&1
check "visibility clean passes" 0 $?

node "$gate" runtime --report "$tmp/error.json" >/dev/null 2>&1
check "runtime audit error fails closed" 2 $?

node "$gate" visibility --report "$tmp/error.json" >/dev/null 2>&1
check "visibility audit error fails closed" 2 $?

node "$gate" runtime --report "$tmp/bad.json" >/dev/null 2>&1
check "unparseable report fails closed" 2 $?

if [ "$fail" -ne 0 ]; then
  echo "npm-audit-gate self-test: FAIL"
  exit 1
fi
echo "npm-audit-gate self-test: PASS"
