"""
ATB (Agent Trace Bundle) Python SDK.

Provides a Pythonic interface for creating, appending to, and verifying
tamper-evident ATB bundles that record AI agent workflow events.

Quick start::

    from atb import Bundle
    from atb.event_types import (
        AI_MODEL_INVOKED_EVENT_TYPE,
        AI_MODEL_OUTPUT_EVENT_TYPE,
        AI_REQUEST_RECEIVED_EVENT_TYPE,
    )

    bundle = Bundle()
    bundle.append(
        AI_REQUEST_RECEIVED_EVENT_TYPE,
        {
            "request_id": "req-001",
            "actor_id_hash": "sha256-actor-abc",
            "purpose_tag": "rag_answer",
        },
    )
    bundle.append(
        AI_MODEL_INVOKED_EVENT_TYPE,
        {
            "model_provider": "openai",
            "model_id": "gpt-4o",
            "model_parameters_digest": "sha256-params-def",
            "prompt_digest": "sha256-prompt-ghi",
        },
    )
    bundle.append(
        AI_MODEL_OUTPUT_EVENT_TYPE,
        {
            "output_digest": "sha256-output-jkl",
            "output_format": "text/plain",
        },
    )
    bundle.save("run.atb/bundle.atb")

    # Later — verify integrity
    b = Bundle.load("run.atb/bundle.atb")
    b.verify()  # raises ATBVerificationError on tampering
"""

from atb.bundle import Bundle
from . import event_types
from atb.action_gate import ActionGate, ActionGateDecision, ActionGateDeniedError, ActionGateInput
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

__version__ = "1.8.0"
__all__ = [
    "Bundle",
    "event_types",
    "ActionGate",
    "ActionGateDeniedError",
    "ActionGateInput",
    "ActionGateDecision",
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
