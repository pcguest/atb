import type { VerificationResponse } from "@/lib/types";

export type TrustScoreBreakdown = {
  continuity: number;
  encryption: number;
  timestamp: number;
  freshness: number;
  total: number;
};

export type TrustScoreInput = {
  verification: VerificationResponse | null;
  lastTimestamp?: string;
  now?: Date;
};

const maxWeights = {
  continuity: 40,
  encryption: 30,
  timestamp: 20,
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
 * Calculate Trust Score for the dashboard.
 *
 * Formula:
 * - continuity (40): `verification.status === "valid"`
 * - encryption (30): `verification.head_hash` is present
 * - timestamp (20): `verification.chain_length > 0`
 * - freshness (10): based on `lastTimestamp` age (<=24h full, <=72h half, older zero)
 *
 * Returns an integer from 0 to 100.
 */
export function calculateTrustScore(input: TrustScoreInput): TrustScoreBreakdown {
  const now = input.now ?? new Date();
  const verification = input.verification;

  const continuity = verification?.status === "valid" ? maxWeights.continuity : 0;
  const encryption = verification?.head_hash ? maxWeights.encryption : 0;
  const timestamp = verification && verification.chain_length > 0 ? maxWeights.timestamp : 0;
  const freshness = calculateFreshnessScore(input.lastTimestamp, now);
  const total = continuity + encryption + timestamp + freshness;

  return {
    continuity,
    encryption,
    timestamp,
    freshness,
    total,
  };
}
