#!/usr/bin/env bash
# Export a public-safe demo tree from the private ATB repository.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${ATB_PUBLIC_EXPORT_DIR:-$ROOT/../atb-demo-export}"
DRY_RUN=0
APPROVED_BY="${APPROVED_BY:-}"
MANIFEST="${ATB_PUBLIC_EXPORT_MANIFEST:-$ROOT/scripts/public-export.manifest.yaml}"

usage() {
  cat <<'EOF'
Usage: scripts/export-public-demo.sh [--dry-run] [--output DIR]

Environment:
  APPROVED_BY                  Required (non dry-run) reviewer identifier before writing export tree.
  ATB_PUBLIC_EXPORT_MANIFEST   Override manifest path (default: scripts/public-export.manifest.yaml)

Writes EXPORT_REVIEW.md beside the export output summarising included paths.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=1; shift ;;
    --output) OUT="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage; exit 1 ;;
  esac
done

if [[ "$DRY_RUN" -eq 0 && -z "$APPROVED_BY" ]]; then
  echo "export-public-demo: set APPROVED_BY=<reviewer> or pass --dry-run" >&2
  exit 1
fi

if [[ ! -f "$MANIFEST" ]]; then
  echo "export-public-demo: manifest not found: $MANIFEST" >&2
  exit 1
fi

manifest_list() {
  local section="$1"
  awk -v section="$section" '
    $0 ~ ("^" section ":[[:space:]]*$") { in_section=1; next }
    in_section && /^[a-zA-Z_]+:/ { exit }
    in_section && /^  - / {
      line=$0
      sub(/^  - /, "", line)
      sub(/\/$/, "", line)
      print line
    }
  ' "$MANIFEST"
}

REVIEW="$OUT/EXPORT_REVIEW.md"
mkdir -p "$OUT"

include_items=()
while IFS= read -r line; do
  [[ -n "$line" ]] && include_items+=("$line")
done < <(manifest_list include)

exclude_paths=()
while IFS= read -r line; do
  [[ -n "$line" ]] && exclude_paths+=("$line")
done < <(manifest_list exclude_paths)

secret_patterns=()
while IFS= read -r line; do
  [[ -n "$line" ]] && secret_patterns+=("$line")
done < <(manifest_list secret_patterns)

if [[ ${#include_items[@]} -eq 0 ]]; then
  echo "export-public-demo: manifest include list is empty" >&2
  exit 1
fi

echo "# ATB public export review" >"$REVIEW"
echo "" >>"$REVIEW"
echo "- Source: $ROOT" >>"$REVIEW"
echo "- Manifest: $MANIFEST" >>"$REVIEW"
echo "- Output: $OUT" >>"$REVIEW"
echo "- Dry run: $DRY_RUN" >>"$REVIEW"
echo "- Approved by: ${APPROVED_BY:-<none>}" >>"$REVIEW"
echo "" >>"$REVIEW"
echo "## Included paths" >>"$REVIEW"

if [[ "$DRY_RUN" -eq 0 ]]; then
  rm -rf "$OUT"
  mkdir -p "$OUT"
fi

rsync_excludes=()
for path in "${exclude_paths[@]}"; do
  [[ -n "$path" ]] || continue
  rsync_excludes+=(--exclude "$path")
done
for pattern in node_modules dist .venv; do
  rsync_excludes+=(--exclude "$pattern")
done

copy_item() {
  local item="$1"
  local src="$ROOT/$item"
  local dest="$OUT/$item"
  if [[ ! -e "$src" ]]; then
    echo "missing include path: $item" >&2
    return 1
  fi
  echo "- $item" >>"$REVIEW"
  if [[ "$DRY_RUN" -eq 0 ]]; then
    mkdir -p "$(dirname "$dest")"
    if [[ -d "$src" ]]; then
      rsync -a "${rsync_excludes[@]}" "$src/" "$dest/"
    else
      cp "$src" "$dest"
    fi
  fi
}

for item in "${include_items[@]}"; do
  copy_item "$item"
done

if [[ "$DRY_RUN" -eq 0 ]]; then
  while IFS= read -r from to; do
    [[ -n "$from" && -n "$to" ]] || continue
    while IFS= read -r -d '' file; do
      if grep -q "$from" "$file" 2>/dev/null; then
        sed -i '' "s/$from/$to/g" "$file" 2>/dev/null || sed -i "s/$from/$to/g" "$file"
      fi
    done < <(find "$OUT/web" -type f \( -name '*.tsx' -o -name '*.ts' -o -name '*.md' \) -print0 2>/dev/null || true)
  done < <(awk '
    /^text_rewrites:/ { in_section=1; next }
    in_section && /^[a-zA-Z_]+:/ { exit }
    in_section && /^  - from:/ { from=$3; next }
    in_section && /^    to:/ { print from, $2 }
  ' "$MANIFEST")

  if [[ -x "$ROOT/examples/bundles/generate.sh" ]] && command -v atb >/dev/null 2>&1; then
    ATB_BIN=atb bash "$ROOT/examples/bundles/generate.sh" || true
  fi
  if [[ -d "$ROOT/examples/bundles" ]]; then
    mkdir -p "$OUT/examples/bundles"
    cp "$ROOT/examples/bundles/"*.atb "$OUT/examples/bundles/" 2>/dev/null || true
  fi
fi

echo "" >>"$REVIEW"
echo "## Secret scan" >>"$REVIEW"
fail=0
if [[ "$DRY_RUN" -eq 0 ]]; then
  secret_hits="/tmp/atb-export-secrets.txt"
  : >"$secret_hits"
  for pattern in "${secret_patterns[@]}"; do
    [[ -n "$pattern" ]] || continue
    if rg -n "$pattern" "$OUT" >>"$secret_hits" 2>/dev/null; then
      :
    fi
  done
  if [[ -s "$secret_hits" ]]; then
    echo "FAILED: potential secrets found" >>"$REVIEW"
    cat "$secret_hits" >>"$REVIEW"
    fail=1
  else
    echo "PASSED: no secret patterns matched" >>"$REVIEW"
  fi
  if find "$OUT" -path '*/internal/*' -print -quit | grep -q .; then
    echo "FAILED: internal/ path leaked into export" >>"$REVIEW"
    fail=1
  fi
else
  echo "SKIPPED (dry-run)" >>"$REVIEW"
fi

if [[ "$fail" -ne 0 ]]; then
  echo "export-public-demo: secret scan or path guard failed; see $REVIEW" >&2
  exit 1
fi

echo "export-public-demo: wrote $REVIEW"
if [[ "$DRY_RUN" -eq 0 ]]; then
  echo "export-public-demo: export tree at $OUT"
fi
