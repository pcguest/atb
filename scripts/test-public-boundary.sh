#!/usr/bin/env bash
set -euo pipefail

source_root="$(git rev-parse --show-toplevel)"
fixture_root="$(mktemp -d)"
trap 'rm -rf -- "$fixture_root"' EXIT
repo="$fixture_root/repo"
mkdir -p "$repo/scripts" "$repo/docs"
cp "$source_root/scripts/check-public-boundary.sh" "$repo/scripts/"
cd "$repo"
git init -q
printf '/.venv\n' > .gitignore
printf 'Portable example documentation.\n' > docs/readme.md
printf 'The Mortise integration is optional.\n' > docs/contract.md
git add .gitignore scripts/check-public-boundary.sh docs/readme.md docs/contract.md

expect_boundary_rejection() {
  local description="$1"
  local output status
  if output="$(bash scripts/check-public-boundary.sh 2>&1)"; then
    echo "$description was not rejected" >&2
    exit 1
  else
    status=$?
  fi
  if [[ "$status" -ne 1 || "$output" != *"public-boundary check failed:"* ]]; then
    echo "$description failed for an unrelated reason (exit=$status): $output" >&2
    exit 1
  fi
}

# The exact root ignore must match a file, symlink, or directory.
touch .venv
git check-ignore --quiet --no-index -- .venv || { echo 'root .venv file is not ignored' >&2; exit 1; }
unlink .venv
ln -s missing-local-environment .venv
git check-ignore --quiet --no-index -- .venv || { echo 'root .venv symlink is not ignored' >&2; exit 1; }
unlink .venv
mkdir .venv
git check-ignore --quiet --no-index -- .venv || { echo 'root .venv directory is not ignored' >&2; exit 1; }
rmdir .venv

bash scripts/check-public-boundary.sh
mkdir -p docs/releases
printf 'Private release working notes.\n' > docs/releases/v1.15.3-convergence.md
git add docs/releases/v1.15.3-convergence.md
expect_boundary_rejection 'superseded release working paper'
unlink docs/releases/v1.15.3-convergence.md

mkdir -p docs/maintainers
printf 'Private visual system notes.\n' > docs/maintainers/visual-system.md
git add docs/maintainers/visual-system.md
expect_boundary_rejection 'private cross-product visual system'
unlink docs/maintainers/visual-system.md
printf 'Example checkout: "/Users/example/src/atb"\n' > docs/unsafe.md
git add docs/unsafe.md
expect_boundary_rejection 'quoted nested machine-specific documentation path'
unlink docs/unsafe.md
ln -s missing-local-environment .venv
git add -f .venv
expect_boundary_rejection 'tracked root .venv symlink'
printf 'public-boundary regression fixtures passed\n'
