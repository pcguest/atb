#!/usr/bin/env bash
set -euo pipefail

echo "🔍 Checking ATB registry health..."

if ! command -v jq >/dev/null 2>&1; then
  echo "❌ jq is required but not installed"
  exit 1
fi

# PyPI check
echo -n "PyPI (atb-sdk): "
pypi_response="$(curl -sS -w "\n%{http_code}" https://pypi.org/pypi/atb-sdk/json)"
pypi_body="$(echo "$pypi_response" | sed '$d')"
pypi_status="$(echo "$pypi_response" | tail -n 1)"

if [[ "$pypi_status" == "200" ]]; then
  pypi_version="$(echo "$pypi_body" | jq -r '.info.version')"
  if [[ -z "$pypi_version" || "$pypi_version" == "null" ]]; then
    echo "❌ missing version in response"
    exit 1
  fi
  echo "✅ $pypi_version"
else
  echo "❌ HTTP $pypi_status"
  exit 1
fi

# npm check
echo -n "npm (@pcguest/atb-sdk): "
npm_response="$(curl -sS -w "\n%{http_code}" https://registry.npmjs.org/@pcguest/atb-sdk)"
npm_body="$(echo "$npm_response" | sed '$d')"
npm_status="$(echo "$npm_response" | tail -n 1)"

if [[ "$npm_status" == "200" ]]; then
  npm_version="$(echo "$npm_body" | jq -r '.["dist-tags"].latest')"
  if [[ -z "$npm_version" || "$npm_version" == "null" ]]; then
    echo "❌ missing dist-tags.latest in response"
    exit 1
  fi
  echo "✅ $npm_version"
else
  echo "❌ HTTP $npm_status"
  exit 1
fi

# Version alignment check (allow patch-level drift during publish timing)
pypi_major_minor="$(echo "$pypi_version" | cut -d. -f1-2)"
npm_major_minor="$(echo "$npm_version" | cut -d. -f1-2)"

if [[ "$pypi_major_minor" == "$npm_major_minor" ]]; then
  echo "✅ Versions aligned (PyPI: $pypi_version, npm: $npm_version)"
else
  echo "⚠️ Version mismatch (PyPI: $pypi_version, npm: $npm_version)"
fi

echo "🎉 All registries healthy"
