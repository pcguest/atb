import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { StatsOverview } from "@/components/dashboard/StatsOverview";

describe("StatsOverview", () => {
  it("shows verification status and grouped event family counts", () => {
    render(
      <StatsOverview
        verification={{
          status: "valid",
          message: "ok",
          bundle_path: "/tmp/bundle.atb",
          chain_length: 7,
          head_hash: "a".repeat(64),
        }}
        meta={{
          bundle_path: "/tmp/bundle.atb",
          event_count: 7,
          type_counts: {
            "ai.llm.call": 3,
            "ai.tool.exec": 2,
            "ai.chain.run": 1,
            "dev.session": 1,
          },
          verified: true,
          verification_message: "ok",
        }}
        viewerHealth={{
          total: 95,
          continuity: 40,
          bundle_integrity: 25,
          event_chain: 20,
          freshness: 10,
        }}
        role="engineer"
      />,
    );

    expect(screen.getByTestId("event-count-value")).toHaveTextContent("7");
    expect(screen.getByTestId("verification-status-value")).toHaveTextContent("valid");
    expect(screen.getByTestId("event-family-counts-value")).toHaveTextContent(
      "llm 3 · tool 2 · chain 1 · other 1",
    );
  });
});
