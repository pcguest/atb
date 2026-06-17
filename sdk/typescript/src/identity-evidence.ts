/** Digest-only reviewer or operator identity evidence supplied by the caller. */
export interface IdentityEvidence {
  identityProvider: string;
  subject: string;
  authContext?: string;
  assertionType: "jwt" | "x509" | "saml" | "opaque";
  assertionDigest: string;
  rawEvidenceDigest?: string;
}

/** Convert public camelCase options to the canonical event payload shape. */
export function identityEvidencePayload(
  evidence: IdentityEvidence | undefined
): Record<string, unknown> | undefined {
  if (!evidence) {
    return undefined;
  }
  const required = {
    identity_provider: evidence.identityProvider?.trim(),
    subject: evidence.subject?.trim(),
    assertion_type: evidence.assertionType,
    assertion_digest: evidence.assertionDigest?.trim(),
  };
  if (
    !required.identity_provider ||
    !required.subject ||
    !required.assertion_type ||
    !required.assertion_digest
  ) {
    throw new Error("atb: identity evidence requires provider, subject, assertion type, and assertion digest");
  }
  const payload: Record<string, unknown> = required;
  if (evidence.authContext?.trim()) {
    payload.auth_context = evidence.authContext.trim();
  }
  if (evidence.rawEvidenceDigest?.trim()) {
    payload.raw_evidence_digest = evidence.rawEvidenceDigest.trim();
  }
  return payload;
}
