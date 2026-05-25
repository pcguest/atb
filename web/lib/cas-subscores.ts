/** CAS sub-score metadata — phrasing from docs/cas-guide.md */

export type CASSubScoreMeta = {
  code: string;
  name: string;
  definition: string;
};

export const CAS_SUB_SCORES: Record<string, CASSubScoreMeta> = {
  EC: {
    code: "EC",
    name: "Event Coverage",
    definition: "Required event types present",
  },
  FC: {
    code: "FC",
    name: "Field Completeness",
    definition: "Required fields populated on present events",
  },
  RC: {
    code: "RC",
    name: "Relation Consistency",
    definition: "Cross-event ID binding",
  },
  TC: {
    code: "TC",
    name: "Temporal Consistency",
    definition: "Causal event ordering",
  },
  SC: {
    code: "SC",
    name: "Source Commitment",
    definition: "Governance binding of recorded events",
  },
  XC: {
    code: "XC",
    name: "External Corroboration",
    definition: "Evidence beyond the bundle",
  },
  AC: {
    code: "AC",
    name: "Anchor Coverage",
    definition: "RFC 3161 TSA verification",
  },
  GC: {
    code: "GC",
    name: "Gating Completeness",
    definition: "Control-plane path from intent to effect",
  },
};

export function casSubScoreMeta(code: string): CASSubScoreMeta | undefined {
  return CAS_SUB_SCORES[code.toUpperCase()];
}

export function casSubScoreInlineLabel(code: string): string {
  const meta = casSubScoreMeta(code);
  if (!meta) {
    return code.toUpperCase();
  }
  const shortName = meta.name.split(" ")[0] ?? meta.name;
  return `${meta.code} · ${shortName}`;
}
