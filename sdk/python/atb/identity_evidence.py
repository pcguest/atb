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


def identity_evidence_payload(
    evidence: IdentityEvidence | None,
) -> dict[str, Any] | None:
    if evidence is None:
        return None
    provider = evidence.identity_provider.strip()
    subject = evidence.subject.strip()
    assertion_digest = evidence.assertion_digest.strip()
    if (
        not provider
        or not subject
        or not evidence.assertion_type
        or not assertion_digest
    ):
        raise ValueError(
            "atb: identity evidence requires provider, subject, assertion type, "
            "and assertion digest"
        )
    payload: dict[str, Any] = {
        "identity_provider": provider,
        "subject": subject,
        "assertion_type": evidence.assertion_type,
        "assertion_digest": assertion_digest,
    }
    if evidence.auth_context and evidence.auth_context.strip():
        payload["auth_context"] = evidence.auth_context.strip()
    if evidence.raw_evidence_digest and evidence.raw_evidence_digest.strip():
        payload["raw_evidence_digest"] = evidence.raw_evidence_digest.strip()
    return payload
