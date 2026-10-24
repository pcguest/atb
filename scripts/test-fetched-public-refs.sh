#!/usr/bin/env bash
set -euo pipefail

source_root="$(git rev-parse --show-toplevel)"
fixture_root="$(mktemp -d)"
trap 'rm -rf -- "$fixture_root"' EXIT
repo="$fixture_root/repo"
snapshot="$fixture_root/advertised-refs.txt"
mkdir -p "$repo"
git -C "$repo" init -q
git -C "$repo" fetch -q "$source_root" HEAD:refs/remotes/origin/main
git -C "$repo" remote add origin https://example.invalid/atb.git
sha="$(git -C "$repo" rev-parse refs/remotes/origin/main)"
git -C "$repo" update-ref refs/remotes/origin/pr/9 "$sha"
git -C "$repo" -c tag.gpgSign=false tag v1.0.0 "$sha"

cat > "$snapshot" <<EOF
$sha refs/heads/main
$sha refs/tags/v1.0.0
$sha refs/pull/9/head
$sha refs/pull/10/merge
EOF

output="$(cd "$repo" && bash "$source_root/scripts/check-fetched-public-refs.sh" origin "$snapshot")"
[[ "$output" == *"advertised_refs_verified=3 branches=1 tags=1 pull_request_heads=1"* ]] || {
  echo "known advertised refs were counted incorrectly: $output" >&2
  exit 1
}

printf '%s refs/notes/unexpected\n' "$sha" >> "$snapshot"
if output="$(cd "$repo" && bash "$source_root/scripts/check-fetched-public-refs.sh" origin "$snapshot" 2>&1)"; then
  echo "unknown advertised ref namespace was silently accepted" >&2
  exit 1
else
  status=$?
fi
if [[ "$status" -ne 2 || "$output" != *"unrecognised advertised ref namespace"* ]]; then
  echo "unknown ref failed for an unrelated reason (exit=$status): $output" >&2
  exit 1
fi

echo "advertised-ref regression fixtures passed"
