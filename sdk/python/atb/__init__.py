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
from atb.encrypt import ATBDecryptionError, ATBEncryptionError, decrypt_bundle, encrypt_bundle
from atb.event import Event
from atb.exceptions import ATBError, ATBVerificationError
from atb.hash import compute_hash, genesis_hash

__version__ = "1.1.0"
__all__ = [
    "Bundle",
    "ActionGate",
    "ActionGateDeniedError",
    "ActionGateInput",
    "ActionGateDecision",
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
