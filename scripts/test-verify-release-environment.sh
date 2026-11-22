#!/usr/bin/env bash
# Local regression fixtures for verify-release-environment.sh. The verifier is
# driven through RELEASE_ENV_JSON so these tests never call GitHub.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
verifier="${script_dir}/verify-release-environment.sh"

run_case() {
  local description="$1"
  local expected_status="$2"
  local json="$3"
  local output status

  set +e
  output="$(RELEASE_ENV_JSON="${json}" bash "${verifier}" docker-release 2>&1)"
  status=$?
  set -e

  if [ "${expected_status}" = "success" ]; then
    if [ "${status}" -ne 0 ]; then
      echo "FAIL: ${description} exited ${status}, expected success" >&2
      printf '%s\n' "${output}" >&2
      exit 1
    fi
  elif [ "${status}" -ne 1 ]; then
    echo "FAIL: ${description} exited ${status}, expected fail-closed exit 1" >&2
    printf '%s\n' "${output}" >&2
    exit 1
  fi

  echo "PASS: ${description}"
}

valid='{
  "name": "docker-release",
  "protection_rules": [
    {
      "type": "required_reviewers",
      "prevent_self_review": true,
      "reviewers": [
        {"type": "User", "reviewer": {"login": "maintainer"}}
      ]
    }
  ]
}'

zero_reviewers='{
  "name": "docker-release",
  "protection_rules": [
    {
      "type": "required_reviewers",
      "prevent_self_review": true,
      "reviewers": []
    }
  ]
}'

prevent_false='{
  "name": "docker-release",
  "protection_rules": [
    {
      "type": "required_reviewers",
      "prevent_self_review": false,
      "reviewers": [
        {"type": "User", "reviewer": {"login": "maintainer"}}
      ]
    }
  ]
}'

prevent_missing='{
  "name": "docker-release",
  "protection_rules": [
    {
      "type": "required_reviewers",
      "reviewers": [
        {"type": "User", "reviewer": {"login": "maintainer"}}
      ]
    }
  ]
}'

missing_rule='{
  "name": "docker-release",
  "protection_rules": []
}'

missing_environment='{
  "message": "Not Found",
  "documentation_url": "https://docs.github.com/rest/deployments/environments"
}'

malformed='{"name": "docker-release", "protection_rules": ['

run_case "valid reviewer with prevent_self_review=true" success "${valid}"
run_case "zero reviewers" failure "${zero_reviewers}"
run_case "prevent_self_review=false" failure "${prevent_false}"
run_case "missing prevent_self_review" failure "${prevent_missing}"
run_case "empty environment object" failure '{}'
run_case "missing environment response" failure "${missing_environment}"
run_case "missing required_reviewers rule" failure "${missing_rule}"
run_case "malformed JSON" failure "${malformed}"

echo "release-environment protection fixtures passed"
