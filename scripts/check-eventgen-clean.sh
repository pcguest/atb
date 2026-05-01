#!/usr/bin/env bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

generated_files=(
  "internal/event/types_generated.go"
  "sdk/python/atb/event_types_generated.py"
  "sdk/typescript/src/eventTypes_generated.ts"
)

go generate ./internal/event/...

missing=()
for file in "${generated_files[@]}"; do
  if ! git ls-files --error-unmatch "$file" >/dev/null 2>&1; then
    missing+=("$file")
  fi
done

if ((${#missing[@]} > 0)); then
  echo "eventgen: generated files are not tracked:" >&2
  printf '  %s\n' "${missing[@]}" >&2
  exit 1
fi

if ! git diff --exit-code -- "${generated_files[@]}"; then
  echo "eventgen: generated files are stale; run go generate ./internal/event/..." >&2
  exit 1
fi
