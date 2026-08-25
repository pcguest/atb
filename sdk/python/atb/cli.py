"""Console-script stub for the ATB Python SDK."""

from __future__ import annotations

import sys

DEPRECATION_MESSAGE = """ATB CLI is not included in the Python SDK.
Install the standalone Go CLI instead:

  go install github.com/pcguest/atb/cmd/atb@latest

The SDK package remains available for Python integrations only. The `atb`
console-script stub will be removed in a future major release.
"""


def main() -> int:
    """Print the SDK CLI deprecation message.

    Args:
        None.

    Returns:
        Process exit code ``1``.

    Raises:
        None.
    """
    print(DEPRECATION_MESSAGE, file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
