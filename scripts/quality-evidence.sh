#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

GOTOOLCHAIN="${GOTOOLCHAIN:-go1.26.4}"
GOVERSION="$(GOTOOLCHAIN="$GOTOOLCHAIN" go env GOVERSION 2>/dev/null | tr ' ' '_' || true)"
GOCACHE="${GOCACHE:-$ROOT/.gocache/${GOVERSION:-default}}"
GOENV=(env "GOCACHE=$GOCACHE" "GOTOOLCHAIN=$GOTOOLCHAIN")
PYTHON_BIN="${PYTHON:-}"
if [[ -z "$PYTHON_BIN" ]]; then
	if command -v python3 >/dev/null 2>&1; then
		PYTHON_BIN="python3"
	elif command -v python >/dev/null 2>&1; then
		PYTHON_BIN="python"
	else
		echo "python3 or python is required for Python SDK evidence" >&2
		exit 127
	fi
fi

echo "== Go coverage =="
GO_COVER_PACKAGES="$("${GOENV[@]}" go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./... | grep -v '^$' | grep -v '/web/node_modules/')"
"${GOENV[@]}" go test $GO_COVER_PACKAGES -coverprofile=coverage.out
"${GOENV[@]}" go tool cover -func=coverage.out | tail -n 1
"${GOENV[@]}" go test ./pkg/api/v1 -cover

echo
echo "== Go benchmarks =="
"${GOENV[@]}" go test ./test/performance ./cmd/atb -run='^$' -bench=. -benchmem

echo
echo "== Python SDK tests =="
(
	cd sdk/python
	"$PYTHON_BIN" -m pytest -q
)

echo
echo "== TypeScript SDK tests =="
(
	cd sdk/typescript
	npm test
)

echo
echo "== Web coverage =="
(
	cd web
	npm run test:coverage
)
