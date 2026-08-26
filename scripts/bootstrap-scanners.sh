#!/usr/bin/env bash
# SPDX-License-Identifier: MIT
set -euo pipefail

readonly TRIVY_VERSION="0.73.0"
readonly GOSEC_VERSION="2.27.1"
readonly DEST_DIR="${1:-.tmp/bin}"

case "$(uname -s)" in
  Darwin) os=darwin; trivy_os=macOS ;;
  Linux) os=linux; trivy_os=Linux ;;
  *) echo "unsupported scanner platform: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch=amd64; trivy_arch=64bit ;;
  arm64|aarch64) arch=arm64; trivy_arch=ARM64 ;;
  *) echo "unsupported scanner architecture: $(uname -m)" >&2; exit 1 ;;
esac

case "${os}/${arch}" in
  darwin/amd64)
    trivy_sha=d39d1374dd3e35d48621b82df9b6625fe69f9920cc67d2739ed81bb679f16f51
    gosec_sha=117cf8dfe02b8746dad579f6ad01019e7c548bb36451e400993d662714dddcd9
    ;;
  darwin/arm64)
    trivy_sha=80cc25faaf6378e37701202d0b4f9f43d9e413d198d594ba60fdf559fe44a683
    gosec_sha=e2d31bb4572471f47489dd6d2f3c98e9261dc65b1889c2a01c48d73d4e40038b
    ;;
  linux/amd64)
    trivy_sha=2edd39da482bb4e9831962487b68f68e3928ec3137794757f54d00383d79547b
    gosec_sha=a1cc5fba45fb51131ba05dee4029b364f62f4b6739b8f24236f93de82f40da40
    ;;
  linux/arm64)
    trivy_sha=13833d97e8a1a5367471c372a173180157f593bece570e20d5d925fef552f5dd
    gosec_sha=33582a6ed6878e4a0456585a8c3b043eef74d989d606bef85afc1a0f9b12f475
    ;;
esac

checksum() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

download_and_verify() {
  local url=$1 archive=$2 expected=$3 actual
  curl --proto '=https' --tlsv1.2 --fail --location --silent --show-error "$url" --output "$archive"
  actual=$(checksum "$archive")
  if [[ "$actual" != "$expected" ]]; then
    echo "checksum mismatch for $(basename "$archive"): expected $expected, got $actual" >&2
    exit 1
  fi
}

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/atb-scanners.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT
mkdir -p "$DEST_DIR"

trivy_archive="trivy_${TRIVY_VERSION}_${trivy_os}-${trivy_arch}.tar.gz"
download_and_verify \
  "https://github.com/aquasecurity/trivy/releases/download/v${TRIVY_VERSION}/${trivy_archive}" \
  "$tmp_dir/$trivy_archive" "$trivy_sha"
tar -xzf "$tmp_dir/$trivy_archive" -C "$tmp_dir" trivy
install -m 0755 "$tmp_dir/trivy" "$DEST_DIR/trivy.new"
mv "$DEST_DIR/trivy.new" "$DEST_DIR/trivy"

gosec_archive="gosec_${GOSEC_VERSION}_${os}_${arch}.tar.gz"
download_and_verify \
  "https://github.com/securego/gosec/releases/download/v${GOSEC_VERSION}/${gosec_archive}" \
  "$tmp_dir/$gosec_archive" "$gosec_sha"
tar -xzf "$tmp_dir/$gosec_archive" -C "$tmp_dir" gosec
install -m 0755 "$tmp_dir/gosec" "$DEST_DIR/gosec.new"
mv "$DEST_DIR/gosec.new" "$DEST_DIR/gosec"

trivy_found=$("$DEST_DIR/trivy" --version | awk 'NR == 1 {print $2}')
gosec_found=$(go version -m "$DEST_DIR/gosec" | awk '$1 == "mod" && $2 == "github.com/securego/gosec/v2" {print $3}')
[[ "$trivy_found" == "$TRIVY_VERSION" ]] || { echo "installed Trivy version mismatch: $trivy_found" >&2; exit 1; }
[[ "$gosec_found" == "v$GOSEC_VERSION" ]] || { echo "installed gosec version mismatch: $gosec_found" >&2; exit 1; }

printf 'Installed verified scanners in %s:\n  Trivy %s\n  gosec v%s\n' "$DEST_DIR" "$trivy_found" "$GOSEC_VERSION"
