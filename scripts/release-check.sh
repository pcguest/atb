#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PYTHON_BIN="python3"
if command -v python >/dev/null 2>&1; then
  PYTHON_BIN="python"
fi

echo "[1/7] Go tests"
go test -skip TestInstalledBinarySmokeFlow ./...

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
  VENV_DIR="$(mktemp -d)"
  "$PYTHON_BIN" -m venv "$VENV_DIR"
  # shellcheck disable=SC1090
  source "$VENV_DIR/bin/activate"
  python -m pip install --upgrade pip
  python -m pip install -e .[dev]
  pytest -q
  python -m pip install build twine
  python -m build
  twine check dist/*
  deactivate
  rm -rf "$VENV_DIR"
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
if command -v docker >/dev/null 2>&1; then
  docker build --platform linux/amd64 -t atb:release-smoke .
  docker run --rm atb:release-smoke version
else
  echo "docker not found; skipping docker smoke build"
fi

echo "[7/7] Version metadata"
"$PYTHON_BIN" - <<'PY'
import pathlib
import tomllib
import json

pyproject = tomllib.loads(pathlib.Path("sdk/python/pyproject.toml").read_text())
package_json = json.loads(pathlib.Path("sdk/typescript/package.json").read_text())
print("python version:", pyproject["project"]["version"])
print("typescript version:", package_json["version"])
PY

echo "release preflight completed"
