#!/usr/bin/env bash
# check-versions.sh — verify all version string locations agree.
# Exits non-zero and prints FAIL lines to stderr if any disagree.
# Source of truth: the latest git tag (leading "v" stripped).
# Set ATB_SKIP_TAG_CHECK=1 on untagged branches to compare against
# cmd/atb/main.go instead of a git tag.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

FAIL=0

MAIN_GO_VERSION="$(grep -E '^\s+version\s+=' "$ROOT_DIR/cmd/atb/main.go" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')"
PY_PYPROJECT="$(grep -E '^version\s*=' "$ROOT_DIR/sdk/python/pyproject.toml" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')"
PY_INIT="$(grep -E '__version__\s*=' "$ROOT_DIR/sdk/python/atb/__init__.py" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')"
TS_VERSION="$(python3 -c "import json; print(json.load(open('$ROOT_DIR/sdk/typescript/package.json'))['version'])")"
TS_LOCK_ROOT="$(python3 -c "import json; print(json.load(open('$ROOT_DIR/sdk/typescript/package-lock.json'))['version'])")"
TS_LOCK_PKG="$(python3 -c "import json; d=json.load(open('$ROOT_DIR/sdk/typescript/package-lock.json')); print(d['packages']['']['version'])")"
WEB_VERSION="$(python3 -c "import json; print(json.load(open('$ROOT_DIR/web/package.json'))['version'])")"
WEB_LOCK_ROOT="$(python3 -c "import json; print(json.load(open('$ROOT_DIR/web/package-lock.json'))['version'])")"
WEB_LOCK_PKG="$(python3 -c "import json; d=json.load(open('$ROOT_DIR/web/package-lock.json')); print(d['packages']['']['version'])")"

if [ "${ATB_SKIP_TAG_CHECK:-0}" = "1" ]; then
  REF_VERSION="$MAIN_GO_VERSION"
else
  LATEST_TAG="$(git -C "$ROOT_DIR" describe --tags --abbrev=0 2>/dev/null || echo "none")"

  if [ "$LATEST_TAG" = "none" ]; then
    echo "error: no git tag found and ATB_SKIP_TAG_CHECK is not set; cannot determine expected version" >&2
    exit 1
  fi

  REF_VERSION="${LATEST_TAG#v}"
fi

check() {
  local label="$1" actual="$2"
  if [ "$actual" != "$REF_VERSION" ]; then
    printf "FAIL: %s = %s (want %s)\n" "$label" "$actual" "$REF_VERSION" >&2
    FAIL=1
  fi
}

check "cmd/atb/main.go"                                    "$MAIN_GO_VERSION"
check "sdk/python/pyproject.toml"                          "$PY_PYPROJECT"
check "sdk/python/atb/__init__.py"                         "$PY_INIT"
check "sdk/typescript/package.json"                        "$TS_VERSION"
check "sdk/typescript/package-lock.json (root)"            "$TS_LOCK_ROOT"
check "sdk/typescript/package-lock.json (packages[\"\"])" "$TS_LOCK_PKG"
check "web/package.json"                                   "$WEB_VERSION"
check "web/package-lock.json (root)"                       "$WEB_LOCK_ROOT"
check "web/package-lock.json (packages[\"\"])"             "$WEB_LOCK_PKG"

if [ "$FAIL" -eq 0 ]; then
  printf "ok: all version strings agree (%s)\n" "$REF_VERSION"
fi
exit "$FAIL"
