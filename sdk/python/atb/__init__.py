"""
ATB (Agent Trace Bundle) Python SDK.

Provides a Pythonic interface for creating, appending to, and verifying
tamper-evident ATB bundles that record AI agent workflow events.

Quick start::

    from atb import Bundle

    bundle = Bundle()
    bundle.append("dev.session", {"date": "2025-01-15", "features": ["init"]})
    bundle.append("decision", {"choice": "Go over Rust", "reason": "velocity"})
    bundle.save("run.atb/bundle.atb")

    # Later — verify integrity
    b = Bundle.load("run.atb/bundle.atb")
    b.verify()  # raises ATBVerificationError on tampering
"""

from atb.bundle import Bundle
from atb.action_gate import ActionGate, ActionGateDecision, ActionGateDeniedError, ActionGateInput
from atb.event import Event
from atb.exceptions import ATBError, ATBVerificationError
from atb.hash import compute_hash, genesis_hash
from atb.langchain_gate import gate_langchain_tool

_encrypt_import_error: ModuleNotFoundError | None = None

try:
    from atb.encrypt import ATBDecryptionError, ATBEncryptionError, decrypt_bundle, encrypt_bundle
except ModuleNotFoundError as exc:
    _encrypt_import_error = exc

    class ATBEncryptionError(ATBError):
        """Raised when encryption support is unavailable."""

    class ATBDecryptionError(ATBError):
        """Raised when decryption support is unavailable."""

    def encrypt_bundle(*args, **kwargs):
        raise ModuleNotFoundError(
            "ATB encryption helpers require the 'cryptography' package. "
            "Install sdk/python dependencies to use encrypt_bundle()."
        ) from _encrypt_import_error

    def decrypt_bundle(*args, **kwargs):
        raise ModuleNotFoundError(
            "ATB encryption helpers require the 'cryptography' package. "
            "Install sdk/python dependencies to use decrypt_bundle()."
        ) from _encrypt_import_error

__version__ = "0.9.0b1"
__all__ = [
    "Bundle",
    "ActionGate",
    "ActionGateDeniedError",
    "ActionGateInput",
    "ActionGateDecision",
    "gate_langchain_tool",
    "Event",
    "ATBError",
    "ATBVerificationError",
    "ATBEncryptionError",
    "ATBDecryptionError",
    "encrypt_bundle",
    "decrypt_bundle",
    "compute_hash",
    "genesis_hash",
]
