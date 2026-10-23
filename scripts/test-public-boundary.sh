#!/usr/bin/env bash
set -euo pipefail

source_root="$(git rev-parse --show-toplevel)"
fixture_root="$(mktemp -d)"
repo="$fixture_root/repo"
mkdir -p "$repo/scripts" "$repo/docs"
cp "$source_root/scripts/check-public-boundary.sh" "$repo/scripts/"
cd "$repo"
git init -q
printf '/.venv\n' > .gitignore
printf 'Portable example documentation.\n' > docs/readme.md
printf 'The Mortise integration is optional.\n' > docs/contract.md
git add .gitignore scripts/check-public-boundary.sh docs/readme.md docs/contract.md

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
if bash scripts/check-public-boundary.sh >/dev/null 2>&1; then
  echo 'superseded release working paper was not rejected' >&2
  exit 1
fi
unlink docs/releases/v1.15.3-convergence.md

mkdir -p docs/maintainers
printf 'Private visual system notes.\n' > docs/maintainers/visual-system.md
git add docs/maintainers/visual-system.md
if bash scripts/check-public-boundary.sh >/dev/null 2>&1; then
  echo 'private cross-product visual system was not rejected' >&2
  exit 1
fi
unlink docs/maintainers/visual-system.md
printf 'Example checkout: /Users/example/atb\n' > docs/unsafe.md
git add docs/unsafe.md
if bash scripts/check-public-boundary.sh >/dev/null 2>&1; then
  echo 'machine-specific documentation path was not rejected' >&2
  exit 1
fi
unlink docs/unsafe.md
ln -s missing-local-environment .venv
git add -f .venv
if bash scripts/check-public-boundary.sh >/dev/null 2>&1; then
  echo 'tracked root .venv symlink was not rejected' >&2
  exit 1
fi
printf 'public-boundary regression fixtures passed\n'
