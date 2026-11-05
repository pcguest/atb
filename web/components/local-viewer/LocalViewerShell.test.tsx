import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { LocalViewerShell } from "@/components/local-viewer/LocalViewerShell";

describe("LocalViewerShell", () => {
  beforeEach(() => {
    window.history.replaceState(null, "", "/view/?bundle_path=%2Ftmp%2Fdemo.atb&surface=evidence&seq=4&from=timeline#session=secret");
  });

  it("renders library and investigation destinations without replacing the seven surfaces", () => {
    const select = vi.fn();
    render(
      <LocalViewerShell route="/view" bundlePath="/tmp/demo.atb" surface="evidence" onSelectSurface={select}>
        <p>Content</p>
      </LocalViewerShell>,
    );

    expect(screen.getAllByRole("navigation", { name: "Library" })[0]).toBeInTheDocument();
    for (const label of ["Run", "Findings", "Timeline", "Context", "Relationships", "Evidence", "Evidence status"]) {
      expect(screen.getAllByRole("button", { name: label }).length).toBeGreaterThan(0);
    }
    expect(screen.getByRole("navigation", { name: "Breadcrumb" })).toHaveTextContent("Library/Bundles/tmp/demo.atb/Evidence");
    expect(screen.getAllByRole("link", { name: "Sessions" })[0]).toHaveAttribute(
      "href",
      "/sessions/?bundle_path=%2Ftmp%2Fdemo.atb#session=secret",
    );
    fireEvent.click(screen.getAllByRole("button", { name: "Timeline" })[0]);
    expect(select).toHaveBeenCalledWith("timeline");
  });

  it("keeps the skip control from changing the session fragment", () => {
    render(
      <LocalViewerShell route="/workspace" title="Workspace bundles">
        <p>Content</p>
      </LocalViewerShell>,
    );
    const skip = screen.getByRole("button", { name: "Skip to content" });
    fireEvent.click(skip);
    expect(window.location.hash).toBe("#session=secret");
    expect(document.getElementById("local-viewer-content")).toHaveFocus();
  });

  it("shows library breadcrumbs without investigation destinations", () => {
    render(
      <LocalViewerShell route="/workspace" title="Workspace bundles" bundlePath="/tmp/demo.atb">
        <p>Content</p>
      </LocalViewerShell>,
    );
    expect(screen.getByRole("navigation", { name: "Breadcrumb" })).toHaveTextContent("Library/Bundles");
    expect(screen.queryByRole("button", { name: "Evidence status" })).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Workspace bundles" })).toBeInTheDocument();
  });
});
