import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { CopyAction, EmptyState, ErrorState, LoadingState, StatusBadge } from "./investigation";

describe("investigation state primitives", () => {
  beforeEach(() => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
    });
  });

  it.each([
    ["verified", "Verified", "text-verified"],
    ["warning", "Inconclusive", "text-warning"],
    ["danger", "Failed", "text-danger"],
    ["unknown", "Untrusted", "text-muted-foreground"],
  ] as const)("renders the %s semantic state", (tone, label, className) => {
    render(<StatusBadge tone={tone}>{label}</StatusBadge>);
    expect(screen.getByText(label)).toHaveClass(className);
  });

  it("gives loading, empty, and error states distinct semantics", () => {
    render(
      <>
        <LoadingState label="Loading findings…" />
        <EmptyState title="No findings">The recorded evidence yielded none.</EmptyState>
        <ErrorState>Reload the local viewer.</ErrorState>
      </>,
    );
    expect(screen.getByRole("status")).toHaveTextContent("Loading findings");
    expect(screen.getByText("No findings")).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("Reload the local viewer");
  });

  it("disables empty copy actions and confirms successful copies", async () => {
    const { rerender } = render(<CopyAction value="" label="Copy digest" />);
    expect(screen.getByRole("button", { name: "Copy digest" })).toBeDisabled();
    rerender(<CopyAction value={"f".repeat(64)} label="Copy digest" />);
    fireEvent.click(screen.getByRole("button", { name: "Copy digest" }));
    await waitFor(() => expect(screen.getByText("Copied")).toBeInTheDocument());
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith("f".repeat(64));
  });
});
