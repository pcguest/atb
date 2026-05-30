import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ProfileCAS } from "@/components/dashboard/ProfileCAS";

const apiClient = vi.hoisted(() => ({
  useBundleProfileQuery: vi.fn(),
  useRunBundleVerifyMutation: vi.fn(),
}));

vi.mock("@/lib/api-client", () => apiClient);

describe("ProfileCAS", () => {
  it("shows chain and anchor status in the report body", () => {
    apiClient.useBundleProfileQuery.mockReturnValue({
      data: {
        profile_id: "atb.profile.rag_answer",
        pass: true,
        chain_valid: true,
        anchor_status: "verified",
        cas_score: 0.92,
        cas_grade: "High",
        sub_scores: { xc: 1 },
        critical_failures: [],
        warnings: [],
        exclusions: [],
        provability_gaps: [],
      },
      isError: false,
      isLoading: false,
      error: null,
    });
    apiClient.useRunBundleVerifyMutation.mockReturnValue({
      isPending: false,
      isError: false,
      error: null,
      mutate: vi.fn(),
    });

    render(<ProfileCAS />);
    fireEvent.click(screen.getByRole("button", { name: /Profile & CAS/i }));

    expect(screen.getByText(/chain valid · anchor verified/i)).toBeInTheDocument();
  });
});
