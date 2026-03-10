"""Console-script entrypoint for the ATB Python package."""

from __future__ import annotations

import os
import shutil
import subprocess
import sys
from importlib import metadata


def _package_version() -> str:
    try:
        return metadata.version("atb-sdk")
    except metadata.PackageNotFoundError:
        return "0.0.0"


def _is_version_request(args: list[str]) -> bool:
    return len(args) == 1 and args[0] in {"--version", "-V", "version"}


def _resolve_binary() -> str | None:
    configured = os.getenv("ATB_BIN")
    if configured:
        return configured

    # Avoid recursively invoking this same console-script wrapper.
    current = os.path.realpath(sys.argv[0])
    for candidate in ("atb-go", "atb"):
        path = shutil.which(candidate)
        if path and os.path.realpath(path) != current:
            return path
    return None


def main() -> int:
    args = sys.argv[1:]
    if _is_version_request(args):
        print(_package_version())
        return 0

    binary = _resolve_binary()
    if not binary:
        print(
            "ATB CLI binary not found. Set ATB_BIN to the Go CLI binary path.",
            file=sys.stderr,
        )
        return 1

    proc = subprocess.run([binary, *args], check=False)
    if proc.returncode is None:
        return 1
    return proc.returncode


if __name__ == "__main__":
    raise SystemExit(main())
