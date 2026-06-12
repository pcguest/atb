"""ATB bundle reader and verifier SDK.

The Python SDK reads ATB bundle files, verifies hash chains and signature
records, and lets callers iterate the loaded records through the public
``Bundle`` and ``Record`` models. The public surface is ``Bundle`` for bundle
I/O and verification, ``Event`` for canonical event construction,
``compute_hash``/``chain_events`` for hash parity checks, and the integration
helpers exported from this module.

Quick start::

    from atb import Bundle
    bundle = Bundle.load("run.atb/bundle.atb")
    for record in bundle.records: print(record.event["type"])
"""

from atb.bundle import Bundle
from . import event_types
from atb.action_gate import (
    ActionGate,
    ActionGateDecision,
    ActionGateDeniedError,
    ActionGateInput,
    ActionPrincipal,
)
from atb.background_job_tracker import BackgroundJobScheduleInput, BackgroundJobTracker
from atb.data_export_gate import DataExportDeniedError, DataExportGate, DataExportInput
from atb.human_override_gate import (
    HumanOverrideActionInput,
    HumanOverrideApprovalInput,
    HumanOverrideDeniedError,
    HumanOverrideGate,
)
from atb.automation_session import AutomationSession, is_capture_environment
from atb.policy_decision_recorder import PolicyDecisionActionInput, PolicyDecisionRecorder
from atb.event import Event
from atb.exceptions import ATBError, ATBVerificationError
from atb.hash import compute_hash, genesis_hash
from atb.langchain_gate import gate_langchain_tool
from atb.pageindex import ATBAppendError, ATBPageIndexRetriever, PageIndexRetrievalError

_encrypt_import_error: ModuleNotFoundError | None = None

try:
    from atb.encrypt import ATBDecryptionError, ATBEncryptionError, decrypt_bundle, encrypt_bundle
except ModuleNotFoundError as exc:
    _encrypt_import_error = exc

    class ATBEncryptionError(ATBError):
        """Raised when encryption support is unavailable.

        Args:
            *args: Positional error message arguments passed to ``Exception``.

        Returns:
            None.

        Raises:
            None.
        """

    class ATBDecryptionError(ATBError):
        """Raised when decryption support is unavailable.

        Args:
            *args: Positional error message arguments passed to ``Exception``.

        Returns:
            None.

        Raises:
            None.
        """

    def encrypt_bundle(*args, **kwargs):
        """Report that encryption helpers are unavailable.

        Args:
            *args: Ignored positional arguments.
            **kwargs: Ignored keyword arguments.

        Returns:
            Never returns.

        Raises:
            ModuleNotFoundError: Always raised when ``cryptography`` is absent.
        """
        raise ModuleNotFoundError(
            "ATB encryption helpers require the 'cryptography' package. "
            "Install sdk/python dependencies to use encrypt_bundle()."
        ) from _encrypt_import_error

    def decrypt_bundle(*args, **kwargs):
        """Report that decryption helpers are unavailable.

        Args:
            *args: Ignored positional arguments.
            **kwargs: Ignored keyword arguments.

        Returns:
            Never returns.

        Raises:
            ModuleNotFoundError: Always raised when ``cryptography`` is absent.
        """
        raise ModuleNotFoundError(
            "ATB encryption helpers require the 'cryptography' package. "
            "Install sdk/python dependencies to use decrypt_bundle()."
        ) from _encrypt_import_error

__version__ = "1.14.4"
__all__ = [
    "Bundle",
    "event_types",
    "ActionGate",
    "ActionGateDeniedError",
    "ActionGateInput",
    "ActionGateDecision",
    "ActionPrincipal",
    "BackgroundJobScheduleInput",
    "BackgroundJobTracker",
    "DataExportDeniedError",
    "DataExportGate",
    "DataExportInput",
    "HumanOverrideActionInput",
    "HumanOverrideApprovalInput",
    "HumanOverrideDeniedError",
    "HumanOverrideGate",
    "AutomationSession",
    "is_capture_environment",
    "PolicyDecisionActionInput",
    "PolicyDecisionRecorder",
    "gate_langchain_tool",
    "Event",
    "ATBError",
    "ATBVerificationError",
    "ATBPageIndexRetriever",
    "ATBAppendError",
    "PageIndexRetrievalError",
    "ATBEncryptionError",
    "ATBDecryptionError",
    "encrypt_bundle",
    "decrypt_bundle",
    "compute_hash",
    "genesis_hash",
]
