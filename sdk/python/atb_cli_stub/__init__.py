"""Compatibility stub for the deprecated Python CLI wrapper."""

from __future__ import annotations

import sys


def main() -> int:
    print(
        "ATB CLI is not included in the Python SDK.\n"
        "Install the standalone Go CLI instead:\n\n"
        "  go install github.com/pcguest/atb/cmd/atb@latest\n\n"
        "The SDK package remains available for Python integrations only. "
        "This compatibility stub will be removed in a future major release.",
        file=sys.stderr,
    )
    return 1
