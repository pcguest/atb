#!/usr/bin/env bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

scanner="${1:-gitleaks}"
if ! command -v "$scanner" >/dev/null 2>&1 && [[ ! -x "$scanner" ]]; then
  echo "gitleaks executable not found: $scanner" >&2
  exit 2
fi

if [[ "$(git rev-parse --is-shallow-repository)" == "true" ]]; then
  echo "refusing full-history audit from a shallow repository" >&2
  exit 2
fi

refs=()
while IFS= read -r ref; do
  refs+=("$ref")
done < <(git for-each-ref --format='%(refname)')

commit_count="$(git rev-list --all --full-history --count)"
ref_count="${#refs[@]}"
if [[ "$commit_count" -eq 0 || "$ref_count" -eq 0 ]]; then
  echo "refusing empty history audit (commits=$commit_count refs=$ref_count)" >&2
  exit 2
fi

for ref in "${refs[@]}"; do
  if ! git rev-parse --verify --quiet "$ref^{commit}" >/dev/null; then
    echo "ref does not resolve to a commit: $ref" >&2
    exit 2
  fi
done

echo "gitleaks_version=$("$scanner" version)"
echo "history_scope=all locally fetched refs with full history"
echo "history_refs=$ref_count reachable_commits=$commit_count"

tmp_root="$(mktemp -d)"
trap 'rm -rf -- "$tmp_root"' EXIT
history_log="$tmp_root/history.log"
if "$scanner" git --redact=100 --no-banner --log-opts="--all --full-history" . 2>&1 | tee "$history_log"; then
  :
else
  status=$?
  echo "full-history secret scan failed (exit=$status)" >&2
  exit "$status"
fi
if ! grep -Eq '[1-9][0-9]* commits scanned[.]' "$history_log" ||
   ! grep -Eq 'scanned ~[1-9][0-9]* bytes' "$history_log"; then
  echo "refusing zero-work full-history scan" >&2
  exit 2
fi
rm -f -- "$history_log"

scan_root="$tmp_root/current-tree"
symlink_root="$tmp_root/symlink-targets"
mkdir "$scan_root" "$symlink_root"
regular_files=0
symlinks=0
while IFS= read -r -d '' path; do
  if [[ -L "$path" ]]; then
    symlinks=$((symlinks + 1))
    readlink "./$path" > "$symlink_root/$symlinks"
    continue
  fi
  [[ -f "$path" ]] || continue
  parent="${path%/*}"
  [[ "$parent" == "$path" ]] && parent="."
  mkdir -p "$scan_root/$parent"
  cp "./$path" "$scan_root/$path"
  regular_files=$((regular_files + 1))
done < <(git ls-files -z --cached --others --exclude-standard)

if [[ "$regular_files" -eq 0 && "$symlinks" -eq 0 ]]; then
  echo "refusing empty current-tree scan" >&2
  exit 2
fi

echo "current_tree_scope=tracked and non-ignored untracked regular files and symlink targets"
echo "current_tree_files=$regular_files current_tree_symlinks=$symlinks"
tree_log="$tmp_root/current-tree.log"
if [[ "$regular_files" -gt 0 ]]; then
  if "$scanner" dir --redact=100 --no-banner "$scan_root" 2>&1 | tee "$tree_log"; then
    :
  else
    status=$?
    echo "current-tree secret scan failed (exit=$status)" >&2
    exit "$status"
  fi
  if ! grep -Eq 'scanned ~[1-9][0-9]* bytes' "$tree_log"; then
    echo "refusing zero-work current-tree scan" >&2
    exit 2
  fi
fi
if [[ "$symlinks" -gt 0 ]]; then
  symlink_log="$tmp_root/symlink-targets.log"
  if "$scanner" dir --redact=100 --no-banner "$symlink_root" 2>&1 | tee "$symlink_log"; then
    :
  else
    status=$?
    echo "symlink-target secret scan failed (exit=$status)" >&2
    exit "$status"
  fi
  if ! grep -Eq 'scanned ~[1-9][0-9]* bytes' "$symlink_log"; then
    echo "refusing zero-work symlink-target scan" >&2
    exit 2
  fi
fi
