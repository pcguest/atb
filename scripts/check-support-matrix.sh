#!/usr/bin/env bash
# check-support-matrix.sh — verify declared tool support does not drift.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FAIL=0

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  FAIL=1
}

expect() {
  local label="$1" actual="$2" expected="$3"
  if [ "$actual" != "$expected" ]; then
    fail "$label = $actual (want $expected)"
  fi
}

require_file_contains() {
  local path="$1" pattern="$2" label="$3"
  if ! grep -Eq "$pattern" "$ROOT_DIR/$path"; then
    fail "$label missing from $path"
  fi
}

GO_MOD_VERSION="$(awk '/^go / { print $2; exit }' "$ROOT_DIR/go.mod")"
MAKEFILE_TOOLCHAIN="$(awk '/^GOTOOLCHAIN[[:space:]]*\?=/ { print $3; exit }' "$ROOT_DIR/Makefile")"
expect "go.mod Go version" "$GO_MOD_VERSION" "1.26.7"
expect "Makefile GOTOOLCHAIN" "$MAKEFILE_TOOLCHAIN" "go1.26.7"

while IFS= read -r wf; do
  if grep -q "setup-go@" "$wf"; then
    if grep -q 'go-version:' "$wf" && ! grep -q 'go-version: "1.26.7"' "$wf"; then
      fail "${wf#"$ROOT_DIR/"} has non-matrix go-version"
    fi
  fi
  if grep -q "setup-python@" "$wf"; then
    if grep -q 'python-version:' "$wf" && ! grep -q 'python-version: "3.11"' "$wf"; then
      fail "${wf#"$ROOT_DIR/"} has non-matrix python-version"
    fi
  fi
  if grep -q "setup-node@" "$wf"; then
    if grep -q 'node-version:' "$wf" && ! grep -q 'node-version: "22"' "$wf"; then
      fail "${wf#"$ROOT_DIR/"} has non-matrix node-version"
    fi
  fi
done < <(find "$ROOT_DIR/.github/workflows" -name '*.yml' -print)

PY_REQUIRES="$(awk -F '"' '/^requires-python/ { print $2; exit }' "$ROOT_DIR/sdk/python/pyproject.toml")"
expect "sdk/python requires-python" "$PY_REQUIRES" ">=3.9"
for version in 3.9 3.10 3.11 3.12; do
  require_file_contains "sdk/python/pyproject.toml" "Programming Language :: Python :: $version" "Python $version classifier"
done

TS_ENGINE="$(python3 -c "import json; print(json.load(open('$ROOT_DIR/sdk/typescript/package.json'))['engines']['node'])")"
expect "sdk/typescript engines.node" "$TS_ENGINE" ">=18"

require_file_contains "docs/support-matrix.md" "Go 1\\.26\\.7" "Go support matrix entry"
require_file_contains "docs/support-matrix.md" "Python 3\\.9-3\\.12" "Python SDK support matrix entry"
require_file_contains "docs/support-matrix.md" "Python 3\\.11" "Python CI support matrix entry"
require_file_contains "docs/support-matrix.md" "Node\\.js >=18" "Node SDK support matrix entry"
require_file_contains "docs/support-matrix.md" "Node\\.js 22" "Node CI support matrix entry"
require_file_contains "docs/support-matrix.md" "npm ci" "npm package-manager support matrix entry"

if [ "$FAIL" -eq 0 ]; then
  echo "ok: support matrix declarations agree"
fi
exit "$FAIL"
