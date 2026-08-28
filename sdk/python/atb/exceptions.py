"""ATB SDK exception hierarchy."""

from __future__ import annotations


class ATBError(Exception):
    """Base exception for all ATB SDK errors.

    Args:
        *args: Positional error message arguments passed to ``Exception``.

    Returns:
        None.

    Raises:
        None.
    """


class ATBVerificationError(ATBError):
    """Raised when bundle integrity verification fails.

    Attributes:
        event_index: The 0-based index of the first tampered event.
        expected_hash: The hash stored in the bundle.
        computed_hash: The hash computed from the event data.

    Args:
        message: Human-readable verification failure message.
        event_index: Optional zero-based record index.
        expected_hash: Optional stored hash value.
        computed_hash: Optional recomputed hash value.

    Returns:
        None.

    Raises:
        None.
    """

    def __init__(
        self,
        message: str,
        event_index: int | None = None,
        expected_hash: str | None = None,
        computed_hash: str | None = None,
    ) -> None:
        super().__init__(message)
        self.event_index = event_index
        self.expected_hash = expected_hash
        self.computed_hash = computed_hash
