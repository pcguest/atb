#!/usr/bin/env bash
set -euo pipefail

scanner="${1:-gitleaks}"
fixture_root="$(mktemp -d)"
repo_root="$(git rev-parse --show-toplevel)"
case "$fixture_root" in
  "$repo_root"/*) echo "fixture directory is inside the repository" >&2; exit 2 ;;
esac

positive="$fixture_root/positive"
negative="$fixture_root/negative"
mkdir -p "$positive" "$negative"
key_prefix="api"
key_suffix="key"
key="${key_prefix}_${key_suffix}"
payload="abCdef0123456789XYZ"
printf '%s=%s\n' "$key" "$payload" > "$positive/fixture.txt"
printf 'ordinary example text\n' > "$negative/fixture.txt"

if "$scanner" dir --redact=100 --no-banner --exit-code=2 "$positive" >/dev/null 2>&1; then
  echo "positive synthetic secret fixture was not detected" >&2
  exit 1
else
  status=$?
  if [[ "$status" -ne 2 ]]; then
    echo "positive fixture scan failed unexpectedly (exit=$status)" >&2
    exit "$status"
  fi
fi

"$scanner" dir --redact=100 --no-banner "$negative" >/dev/null
printf '%s\n' "$positive/fixture.txt" "$negative/fixture.txt" | while IFS= read -r path; do unlink "$path"; done
rmdir "$positive" "$negative" "$fixture_root"
echo "synthetic secret scanner fixtures passed"
