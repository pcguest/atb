#!/usr/bin/env bash
# check-versions.sh — verify all five version string locations agree.
# Exits non-zero and prints FAIL lines to stderr if any disagree.
# Source of truth: the version constant in cmd/atb/main.go.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

REF_VERSION="$(grep -E '^\s+version\s+=' "$ROOT_DIR/cmd/atb/main.go" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')"

if [ -z "$REF_VERSION" ]; then
  echo "error: could not extract version from cmd/atb/main.go" >&2
  exit 1
fi

FAIL=0

check() {
  local label="$1" actual="$2"
  if [ "$actual" != "$REF_VERSION" ]; then
    printf "FAIL: %s = %s (want %s)\n" "$label" "$actual" "$REF_VERSION" >&2
    FAIL=1
  fi
}

PY_PYPROJECT="$(grep -E '^version\s*=' "$ROOT_DIR/sdk/python/pyproject.toml" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')"
PY_INIT="$(grep -E '__version__\s*=' "$ROOT_DIR/sdk/python/atb/__init__.py" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')"
TS_VERSION="$(python3 -c "import json; print(json.load(open('$ROOT_DIR/sdk/typescript/package.json'))['version'])")"
WEB_VERSION="$(python3 -c "import json; print(json.load(open('$ROOT_DIR/web/package.json'))['version'])")"

check "sdk/python/pyproject.toml"   "$PY_PYPROJECT"
check "sdk/python/atb/__init__.py"  "$PY_INIT"
check "sdk/typescript/package.json" "$TS_VERSION"
check "web/package.json"            "$WEB_VERSION"

# Skip the tag equality check when ATB_SKIP_TAG_CHECK=1 (e.g. feature branches
# that have not yet been tagged). Cross-file version agreement is always checked.
if [ "${ATB_SKIP_TAG_CHECK:-0}" != "1" ]; then
  VERSION="$REF_VERSION"
  LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "none")
  if [ "$LATEST_TAG" != "v${VERSION}" ]; then
    echo "error: latest tag $LATEST_TAG does not match version v${VERSION}"
    exit 1
  fi
fi

if [ "$FAIL" -eq 0 ]; then
  printf "ok: all version strings agree (%s)\n" "$REF_VERSION"
fi
exit "$FAIL"
