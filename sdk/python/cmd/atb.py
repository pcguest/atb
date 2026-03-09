"""Python console-script shim for invoking the ATB CLI binary."""

from __future__ import annotations

import os
import subprocess
import sys


def main() -> int:
    binary = os.getenv("ATB_BIN", "atb")
    proc = subprocess.run([binary, *sys.argv[1:]], check=False)
    if proc.returncode is None:
        return 1
    return proc.returncode


if __name__ == "__main__":
    raise SystemExit(main())
