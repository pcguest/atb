#!/usr/bin/env bash
set -euo pipefail

# ci-atb-capture.sh — wraps a CI step with atb capture run.
# Usage: ci-atb-capture.sh <snapshot-name> -- <command> [args...]
# Requires: ATB_CI_BUNDLE_PATH env var set to the bundle path for this run.
# Exits with the wrapped command's exit code.
SNAPSHOT_NAME="$1"
shift 2

exec atb capture run \
	--bundle "${ATB_CI_BUNDLE_PATH}" \
	--snapshot "${SNAPSHOT_NAME}" \
	-- "$@"
