#!/usr/bin/env bash
# Build a self-contained external-pilot kit for the atb intercept
# continuous-capture path. See docs/pilot/continuous-capture-pilot-kit.md.
#
# Usage: scripts/build-pilot-kit.sh [output-dir] [GOOS] [GOARCH]
#   output-dir  default: dist/pilot
#   GOOS/GOARCH default: the host platform
set -euo pipefail

out="${1:-dist/pilot}"
goos="${2:-$(go env GOOS)}"
goarch="${3:-$(go env GOARCH)}"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

# The kit records the commit it was built from, so the tree must be clean
# unless the caller explicitly accepts an untraceable local build.
if [ -n "$(git status --porcelain)" ] && [ "${ATB_PILOT_ALLOW_DIRTY:-0}" != "1" ]; then
	echo "error: working tree is dirty; commit or stash so PILOT-COMMIT matches the artefacts" >&2
	echo "       (set ATB_PILOT_ALLOW_DIRTY=1 to build a local, untraceable kit)" >&2
	exit 1
fi

mkdir -p "$out"
out="$(cd "$out" && pwd)"

bin_name="atb"
if [ "$goos" = "windows" ]; then
	bin_name="atb.exe"
fi

echo "building $bin_name for ${goos}/${goarch} from $(git rev-parse --short HEAD)"
CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
	go build -trimpath -o "$out/$bin_name" ./cmd/atb

cp docs/capture/continuous-capture.md "$out/continuous-capture.md"
cp docs/pilot/continuous-capture-pilot-kit.md "$out/PILOT-README.md"
git rev-parse HEAD > "$out/PILOT-COMMIT"

# Portable SHA-256: prefer sha256sum, fall back to shasum.
if command -v sha256sum >/dev/null 2>&1; then
	( cd "$out" && sha256sum "$bin_name" continuous-capture.md PILOT-README.md PILOT-COMMIT > checksums.txt )
else
	( cd "$out" && shasum -a 256 "$bin_name" continuous-capture.md PILOT-README.md PILOT-COMMIT > checksums.txt )
fi

echo "pilot kit written to $out"
ls -1 "$out"
