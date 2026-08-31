#!/usr/bin/env bash
# Verify that image metadata and the embedded ATB ELF executable both match
# the requested platform. Runtime success alone is insufficient when binfmt
# or other transparent emulation can execute a foreign-architecture binary.
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "usage: $0 IMAGE linux/amd64|linux/arm64" >&2
  exit 2
fi

IMAGE="$1"
PLATFORM="$2"
case "$PLATFORM" in
  linux/amd64) EXPECTED_MACHINE=62; EXPECTED_ARCH=amd64 ;;
  linux/arm64) EXPECTED_MACHINE=183; EXPECTED_ARCH=arm64 ;;
  *)
    echo "unsupported target platform: $PLATFORM" >&2
    exit 2
    ;;
esac

IMAGE_PLATFORM="$(docker image inspect --format '{{.Os}}/{{.Architecture}}' "$IMAGE")"
if [ "$IMAGE_PLATFORM" != "$PLATFORM" ]; then
  echo "image platform $IMAGE_PLATFORM does not match requested $PLATFORM" >&2
  exit 1
fi

CHECK_DIR="$(mktemp -d)"
CONTAINER_ID=""
cleanup() {
  if [ -n "$CONTAINER_ID" ]; then
    docker rm -f "$CONTAINER_ID" >/dev/null 2>&1 || true
  fi
  rm -rf "$CHECK_DIR"
}
trap cleanup EXIT

CONTAINER_ID="$(docker create --platform "$PLATFORM" "$IMAGE")"
docker cp "$CONTAINER_ID:/app/atb" "$CHECK_DIR/atb"

python3 - "$CHECK_DIR/atb" "$EXPECTED_MACHINE" "$EXPECTED_ARCH" <<'PY'
import pathlib
import sys

binary = pathlib.Path(sys.argv[1])
expected_machine = int(sys.argv[2])
expected_arch = sys.argv[3]
header = binary.read_bytes()[:20]

if len(header) < 20 or header[:4] != b"\x7fELF":
    raise SystemExit(f"{binary}: not an ELF executable")
if header[4] != 2:
    raise SystemExit(f"{binary}: expected ELF64, class byte is {header[4]}")
if header[5] != 1:
    raise SystemExit(f"{binary}: expected little-endian ELF, data byte is {header[5]}")

machine = int.from_bytes(header[18:20], byteorder="little")
names = {62: "amd64/x86-64", 183: "arm64/AArch64"}
actual_arch = names.get(machine, f"ELF e_machine={machine}")
if machine != expected_machine:
    raise SystemExit(
        f"embedded /app/atb architecture {actual_arch} does not match expected {expected_arch}"
    )
print(f"embedded /app/atb architecture: {actual_arch}")
PY

echo "requested platform: $PLATFORM"
echo "image platform: $IMAGE_PLATFORM"
