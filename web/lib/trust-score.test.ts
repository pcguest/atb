import { describe, expect, it } from "vitest";

import { calculateTrustScore } from "@/lib/trust-score";

describe("calculateTrustScore", () => {
  it("returns full score when verification is valid and fresh", () => {
    const result = calculateTrustScore({
      verification: {
        status: "valid",
        message: "ok",
        bundle_path: "/tmp/run.atb",
        chain_length: 10,
        head_hash: "a".repeat(64),
      },
      lastTimestamp: "2026-03-12T01:00:00.000Z",
      now: new Date("2026-03-12T08:00:00.000Z"),
    });

    expect(result.total).toBe(100);
    expect(result.continuity).toBe(40);
    expect(result.encryption).toBe(30);
    expect(result.timestamp).toBe(20);
    expect(result.freshness).toBe(10);
  });

  it("downgrades score when verification is invalid and stale", () => {
    const result = calculateTrustScore({
      verification: {
        status: "invalid",
        message: "tampered",
        bundle_path: "/tmp/run.atb",
        chain_length: 0,
      },
      lastTimestamp: "2026-03-01T01:00:00.000Z",
      now: new Date("2026-03-12T08:00:00.000Z"),
    });

    expect(result.total).toBe(0);
    expect(result.continuity).toBe(0);
    expect(result.encryption).toBe(0);
    expect(result.timestamp).toBe(0);
    expect(result.freshness).toBe(0);
  });
});
