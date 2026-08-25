"""Digest-only reviewer identity evidence for oversight events."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Literal


@dataclass(frozen=True)
class IdentityEvidence:
    """Caller-provided identity context; ATB does not verify the assertion."""

    identity_provider: str
    subject: str
    assertion_type: Literal["jwt", "x509", "saml", "opaque"]
    assertion_digest: str
    auth_context: str | None = None
    raw_evidence_digest: str | None = None


_ASSERTION_TYPES = ("jwt", "x509", "saml", "opaque")


def _stripped_string(value: Any, field: str) -> str:
    if not isinstance(value, str):
        raise ValueError(f"atb: identity evidence {field} must be a string")
    return value.strip()


def identity_evidence_payload(
    evidence: IdentityEvidence | None,
) -> dict[str, Any] | None:
    if evidence is None:
        return None
    provider = _stripped_string(evidence.identity_provider, "identity_provider")
    subject = _stripped_string(evidence.subject, "subject")
    assertion_type = _stripped_string(evidence.assertion_type, "assertion_type")
    assertion_digest = _stripped_string(evidence.assertion_digest, "assertion_digest")
    if not provider or not subject or not assertion_type or not assertion_digest:
        raise ValueError(
            "atb: identity evidence requires provider, subject, assertion type, "
            "and assertion digest"
        )
    if assertion_type not in _ASSERTION_TYPES:
        raise ValueError(
            "atb: identity evidence assertion_type must be one of "
            + ", ".join(_ASSERTION_TYPES)
        )
    payload: dict[str, Any] = {
        "identity_provider": provider,
        "subject": subject,
        "assertion_type": assertion_type,
        "assertion_digest": assertion_digest,
    }
    if evidence.auth_context is not None:
        auth_context = _stripped_string(evidence.auth_context, "auth_context")
        if auth_context:
            payload["auth_context"] = auth_context
    if evidence.raw_evidence_digest is not None:
        raw_digest = _stripped_string(
            evidence.raw_evidence_digest, "raw_evidence_digest"
        )
        if raw_digest:
            payload["raw_evidence_digest"] = raw_digest
    return payload
