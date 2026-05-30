import type { VerificationResponse } from "@/lib/types";

export type ViewerHealthBreakdown = {
  continuity: number;
  bundle_integrity: number;
  event_chain: number;
  freshness: number;
  total: number;
};

export type ViewerHealthInput = {
  verification: VerificationResponse | null;
  lastTimestamp?: string;
  now?: Date;
};

const maxWeights = {
  continuity: 40,
  bundle_integrity: 30,
  event_chain: 20,
  freshness: 10,
} as const;

function calculateFreshnessScore(lastTimestamp: string | undefined, now: Date): number {
  if (!lastTimestamp) {
    return 0;
  }

  const parsed = new Date(lastTimestamp);
  if (Number.isNaN(parsed.getTime())) {
    return 0;
  }

  const ageHours = Math.max(0, (now.getTime() - parsed.getTime()) / (1000 * 60 * 60));
  if (ageHours <= 24) {
    return maxWeights.freshness;
  }
  if (ageHours <= 72) {
    return Math.round(maxWeights.freshness * 0.5);
  }
  return 0;
}

/**
 * Calculate Viewer Health Score for the dashboard.
 *
 * Formula:
 * - continuity (40): `verification.status === "valid"`
 * - bundle_integrity (30): `verification.head_hash` is present
 * - event_chain (20): `verification.chain_length > 0`
 * - freshness (10): based on `lastTimestamp` age (<=24h full, <=72h half, older zero)
 *
 * Returns an integer from 0 to 100.
 */
export function calculateViewerHealthScore(input: ViewerHealthInput): ViewerHealthBreakdown {
  const now = input.now ?? new Date();
  const verification = input.verification;

  const continuity = verification?.status === "valid" ? maxWeights.continuity : 0;
  const bundle_integrity = verification?.head_hash ? maxWeights.bundle_integrity : 0;
  const event_chain = verification && verification.chain_length > 0 ? maxWeights.event_chain : 0;
  const freshness = calculateFreshnessScore(input.lastTimestamp, now);
  const total = continuity + bundle_integrity + event_chain + freshness;

  return {
    continuity,
    bundle_integrity,
    event_chain,
    freshness,
    total,
  };
}
