import React from "react";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { BundleMetaPanel } from "@/app/view/components/bundle-meta/BundleMetaPanel";

describe("BundleMetaPanel", () => {
  it("renders screen-reader hash labels for returned hash fields", () => {
    const headHash = "b".repeat(64);
    const genesisHash = "a".repeat(64);

    const { getAllByRole, getByLabelText, getByText } = render(
      <BundleMetaPanel
        loading={false}
        verification={{
          status: "valid",
          message: "ok",
          bundle_path: "/tmp/test.atb",
          chain_length: 8,
          head_hash: headHash,
        }}
        meta={{
          bundle_path: "/tmp/test.atb",
          event_count: 8,
          type_counts: { "ai.llm.request": 4 },
          genesis_hash: genesisHash,
          verified_at: "2026-03-12T00:00:00.000Z",
          verified: true,
          verification_message: "ok",
        }}
        chainLength={8}
      />,
    );

    expect(getByLabelText(`SHA-256 hash: ${headHash}`)).toBeInTheDocument();
    expect(getByLabelText(`SHA-256 hash: ${genesisHash}`)).toBeInTheDocument();
    expect(getByText("2026-03-12T00:00:00.000Z")).toBeInTheDocument();
    expect(getAllByRole("button", { name: "Copy SHA-256 hash to clipboard" })).toHaveLength(2);
  });
});
