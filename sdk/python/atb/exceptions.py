"""ATB SDK exception hierarchy."""


class ATBError(Exception):
    """Base exception for all ATB SDK errors."""


class ATBVerificationError(ATBError):
    """Raised when bundle integrity verification fails.

    Attributes:
        event_index: The 0-based index of the first tampered event.
        expected_hash: The hash stored in the bundle.
        computed_hash: The hash computed from the event data.
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
