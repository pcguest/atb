import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import WorkspacePage from "@/app/workspace/page";

vi.mock("@/lib/agent-mode", () => ({
  detectAgentMode: vi.fn(),
}));

vi.mock("@/lib/api-client", () => ({
  useWorkspaceBundlesQuery: vi.fn(),
}));

import { detectAgentMode } from "@/lib/agent-mode";
import { useWorkspaceBundlesQuery } from "@/lib/api-client";

describe("WorkspacePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.history.replaceState(null, "", "/workspace#session=secret");
    vi.mocked(detectAgentMode).mockResolvedValue(true);
    vi.mocked(useWorkspaceBundlesQuery).mockReturnValue({
      isLoading: false,
      isError: false,
      isSuccess: true,
      data: { bundles: [] },
    } as never);
  });

  it("uses the local-viewer shell and library navigation", async () => {
    render(<WorkspacePage />);
    expect(screen.getByRole("heading", { name: "Workspace bundles" })).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "Breadcrumb" })).toHaveTextContent("Library/Bundles");
    expect(screen.getAllByRole("link", { name: "Sessions" })[0]).toHaveAttribute(
      "href",
      "/sessions/#session=secret",
    );
    await waitFor(() => expect(detectAgentMode).toHaveBeenCalled());
  });
});
