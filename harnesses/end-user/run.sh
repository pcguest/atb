#!/usr/bin/env bash
# ATB end-user validation harness — thin wrapper around run.py.
#
# Usage:
#   ./run.sh [--atb /path/to/atb] [--out atb-enduser.report.json]
#
# Requires python3 and an `atb` binary (ATB_BIN, PATH, or ../../atb).
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec python3 "$HERE/run.py" "$@"
