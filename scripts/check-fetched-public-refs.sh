#!/usr/bin/env bash
set -euo pipefail

remote="${1:-origin}"
if ! git remote get-url "$remote" >/dev/null 2>&1; then
  echo "Git remote not configured: $remote" >&2
  exit 2
fi

if ! remote_refs="$(git ls-remote --refs "$remote")"; then
  echo "unable to enumerate advertised refs from $remote" >&2
  exit 2
fi

branches=0
tags=0
pull_heads=0
total=0
while read -r sha remote_ref; do
  [[ -n "${remote_ref:-}" ]] || continue
  local_ref=""
  case "$remote_ref" in
    refs/heads/*)
      local_ref="refs/remotes/$remote/${remote_ref#refs/heads/}"
      branches=$((branches + 1))
      ;;
    refs/tags/*)
      local_ref="$remote_ref"
      tags=$((tags + 1))
      ;;
    refs/pull/*/head)
      number="${remote_ref#refs/pull/}"
      number="${number%/head}"
      local_ref="refs/remotes/$remote/pr/$number"
      pull_heads=$((pull_heads + 1))
      ;;
    *)
      continue
      ;;
  esac

  actual="$(git rev-parse --verify --quiet "$local_ref" 2>/dev/null || true)"
  if [[ "$actual" != "$sha" ]]; then
    echo "advertised ref missing or stale after fetch: $remote_ref" >&2
    exit 2
  fi
  total=$((total + 1))
done <<< "$remote_refs"

if [[ "$total" -eq 0 || "$branches" -eq 0 ]]; then
  echo "refusing incomplete advertised-ref inventory (refs=$total branches=$branches)" >&2
  exit 2
fi

echo "advertised_refs_verified=$total branches=$branches tags=$tags pull_request_heads=$pull_heads remote=$remote"
