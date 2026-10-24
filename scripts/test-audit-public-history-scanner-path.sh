#!/usr/bin/env bash
set -euo pipefail

source_root="$(git rev-parse --show-toplevel)"
fixture_root="$source_root/.tmp/audit-scanner-path-$$"
trap 'rm -rf -- "$fixture_root"' EXIT
mkdir -p "$fixture_root/bin"

cat > "$fixture_root/bin/fake-gitleaks" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  version) echo "fake-gitleaks 1.0" ;;
  git) printf '1 commits scanned.\nscanned ~1 bytes\n' ;;
  dir) printf 'scanned ~1 bytes\n' ;;
  *) echo "unexpected fake-gitleaks command: ${1:-}" >&2; exit 2 ;;
esac
EOF
chmod +x "$fixture_root/bin/fake-gitleaks"

output="$(cd "$source_root/scripts" && bash "$source_root/scripts/audit-public-history.sh" "../.tmp/audit-scanner-path-$$/bin/fake-gitleaks")"
[[ "$output" == *"gitleaks_version=fake-gitleaks 1.0"* ]] || {
  echo "relative scanner path was not preserved across the script's root change: $output" >&2
  exit 1
}

echo "history-audit relative scanner regression fixture passed"
