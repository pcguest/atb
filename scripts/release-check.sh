#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

SOURCE_VERSION="$(grep -E '^\s+version\s+=' cmd/atb/main.go | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')"
EXPECT="${EXPECT:-$SOURCE_VERSION}"
if [[ ! "$EXPECT" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "EXPECT must be stable SemVer without a leading v; got $EXPECT" >&2
  exit 1
fi
if [[ "$EXPECT" != "$SOURCE_VERSION" ]]; then
  echo "EXPECT $EXPECT does not match cmd/atb/main.go $SOURCE_VERSION" >&2
  exit 1
fi
ATB_SKIP_TAG_CHECK=1 bash scripts/check-versions.sh

PYTHON_BIN="${ATB_RELEASE_PYTHON:-python3.11}"
if ! "$PYTHON_BIN" -c 'import sys; raise SystemExit(sys.version_info < (3, 11))'; then
  echo "ATB release tooling requires Python 3.11+; got $PYTHON_BIN" >&2
  exit 1
fi

# The Go suite contains cross-language parity tests that import the Python SDK.
# Build one exact, disposable environment before running any tests and point
# both parity-test conventions at it. This keeps the release preflight valid on
# a clean machine instead of depending on packages installed in global Python.
VENV_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$VENV_DIR"
}
trap cleanup EXIT
"$PYTHON_BIN" -m venv "$VENV_DIR"
ATB_PYTHON_BIN="$VENV_DIR/bin/python"
export ATB_PYTHON_BIN
ATB_PYTHON="$ATB_PYTHON_BIN"
export ATB_PYTHON
"$ATB_PYTHON_BIN" -m pip install --upgrade pip
"$ATB_PYTHON_BIN" -m pip install -r sdk/python/requirements-dev.txt
"$ATB_PYTHON_BIN" -m pip install -r sdk/python/requirements-release.txt

echo "[1/7] Go tests"
go_packages=()
while IFS= read -r package; do
  go_packages+=("$package")
done < <(go list ./... | grep -v '/node_modules/')
go test -skip TestInstalledBinarySmokeFlow "${go_packages[@]}"

echo "[2/7] TypeScript lockfile + build"
(
  cd sdk/typescript
  npm ci --dry-run
  npm ci
  npm run typecheck
  npm run test
  npm run build
)

echo "[3/7] Python tests + package build"
(
  cd sdk/python
  "$ATB_PYTHON_BIN" -m pip install -e .[dev] --no-deps
  "$ATB_PYTHON_BIN" -m pytest -q
  "$ATB_PYTHON_BIN" -m build --no-isolation
  "$ATB_PYTHON_BIN" -m twine check dist/*
)

echo "[4/7] Web dashboard build"
(
  cd web
  npm ci
  npm run build
)

echo "[5/7] Installed binary smoke gate"
go test -v -run TestInstalledBinarySmokeFlow ./cmd/atb

echo "[6/7] Docker smoke build"
if [[ "${SKIP_DOCKER:-}" == "1" ]]; then
  echo "Skipping Docker smoke build (SKIP_DOCKER=1). The CI workflow docker-publish.yml will build and publish the image on tag push."
elif command -v docker >/dev/null 2>&1; then
  docker build --build-arg "ATB_VERSION=v${EXPECT}" --platform linux/amd64 -t atb:release-smoke .
  label_version="$(docker image inspect --format '{{ index .Config.Labels "org.opencontainers.image.version" }}' atb:release-smoke)"
  binary_version="$(docker run --rm atb:release-smoke version | awk '{print $NF}')"
  if [[ "$label_version" != "v${EXPECT}" || "$binary_version" != "$EXPECT" ]]; then
    echo "Docker version mismatch: label=$label_version binary=$binary_version expected=v${EXPECT}/$EXPECT" >&2
    exit 1
  fi
  echo "Docker label and binary versions agree at v${EXPECT}"
else
  echo "docker not found; skipping docker smoke build"
fi

echo "[7/7] Version metadata"
"$ATB_PYTHON_BIN" - <<'PY'
import pathlib
import json

try:
    import tomllib
except ModuleNotFoundError:  # Python 3.9 and 3.10 support floor.
    import tomli as tomllib

pyproject = tomllib.loads(pathlib.Path("sdk/python/pyproject.toml").read_text())
package_json = json.loads(pathlib.Path("sdk/typescript/package.json").read_text())
print("python version:", pyproject["project"]["version"])
print("typescript version:", package_json["version"])
PY

echo "release preflight completed"
