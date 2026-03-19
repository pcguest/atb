"""Compatibility stub for the deprecated Python console-script wrapper."""

from __future__ import annotations

import sys


def main() -> int:
    print(
        "ATB CLI is not included in the Python SDK.\n"
        "Install the standalone Go CLI with:\n\n"
        "  go install github.com/pcguest/atb/cmd/atb@latest\n",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
