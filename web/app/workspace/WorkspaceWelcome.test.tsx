import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { WorkspaceWelcome } from "@/app/workspace/WorkspaceWelcome";

describe("WorkspaceWelcome", () => {
  it("shows first-run capture instructions", () => {
    render(<WorkspaceWelcome />);

    expect(screen.getByTestId("workspace-welcome")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "No bundles yet" })).toBeInTheDocument();
    expect(screen.getByText(/atb agent run/i)).toBeInTheDocument();
    expect(screen.getByText(/AutomationSession/i)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Agent guide/i })).toHaveAttribute(
      "href",
      expect.stringContaining("docs/guides/agent.md"),
    );
  });
});
