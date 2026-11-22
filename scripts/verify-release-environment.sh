#!/usr/bin/env bash
# Fail closed unless the named GitHub environment is configured with required
# reviewers who cannot approve their own deployment.
#
# Usage: verify-release-environment.sh ENVIRONMENT_NAME
#
# The GitHub API is used in CI. Tests and local checks can inject the response
# with RELEASE_ENV_JSON so they do not need a token or network access.
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: $0 ENVIRONMENT_NAME" >&2
  exit 2
fi

ENVIRONMENT_NAME="$1"

if [ "${RELEASE_ENV_JSON+x}" = "x" ]; then
  ENVIRONMENT_RESPONSE="${RELEASE_ENV_JSON}"
else
  if [ -z "${GITHUB_REPOSITORY:-}" ]; then
    echo "GITHUB_REPOSITORY is required to read the release environment." >&2
    exit 1
  fi
  if [ -z "${GH_TOKEN:-}" ]; then
    echo "GH_TOKEN is required to read the release environment." >&2
    exit 1
  fi
  if ! ENVIRONMENT_RESPONSE="$(gh api "repos/${GITHUB_REPOSITORY}/environments/${ENVIRONMENT_NAME}")"; then
    echo "failed to read GitHub environment ${ENVIRONMENT_NAME}" >&2
    exit 1
  fi
fi

RELEASE_ENV_RESPONSE="${ENVIRONMENT_RESPONSE}" python3 - "${ENVIRONMENT_NAME}" <<'PY'
import json
import os
import sys

environment_name = sys.argv[1]
raw = os.environ.get("RELEASE_ENV_RESPONSE", "")

try:
    response = json.loads(raw)
except json.JSONDecodeError as exc:
    raise SystemExit(f"malformed environment JSON: {exc}")

if not isinstance(response, dict):
    raise SystemExit("environment response is not a JSON object")

rules = response.get("protection_rules")
if not isinstance(rules, list):
    raise SystemExit("environment response has no protection_rules list")

review_rule = next(
    (
        rule
        for rule in rules
        if isinstance(rule, dict) and rule.get("type") == "required_reviewers"
    ),
    None,
)
if review_rule is None:
    raise SystemExit("environment has no required_reviewers protection rule")

reviewers = review_rule.get("reviewers")
if not isinstance(reviewers, list) or len(reviewers) == 0:
    raise SystemExit("environment required_reviewers rule has no reviewers")

if review_rule.get("prevent_self_review") is not True:
    raise SystemExit(
        "environment required_reviewers rule does not set prevent_self_review=true"
    )

name = response.get("name")
if not isinstance(name, str) or not name:
    name = environment_name

print(f"Environment: {name}")
print(f"required_reviewers: {len(reviewers)}")
print(f"prevent_self_review: {str(review_rule['prevent_self_review']).lower()}")
print(f"Environment protection verified for {name}.")
PY
