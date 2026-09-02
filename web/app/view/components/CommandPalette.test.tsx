import React from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { CommandPalette, type PaletteAction } from "./CommandPalette";

const actions: PaletteAction[] = [
  { id: "verify", label: "Verify bundle", group: "operate", run: vi.fn() },
  { id: "findings", label: "Open findings", group: "navigate", run: vi.fn() },
  { id: "copy-digest", label: "Copy selected evidence digest", group: "copy", run: vi.fn() },
];

describe("CommandPalette", () => {
  it("traps focus and moves aria-activedescendant with arrow keys", () => {
    render(<CommandPalette actions={actions} />);
    fireEvent.click(screen.getByRole("button", { name: "Open command palette" }));

    const input = screen.getByRole("combobox", { name: "Filter commands" });
    expect(input).toHaveAttribute("aria-activedescendant", "palette-option-verify");
    expect(screen.getByRole("option", { name: "Verify bundle" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.getByText("Operate")).toBeInTheDocument();
    expect(screen.getByText("Navigate")).toBeInTheDocument();
    expect(screen.getByText("Copy")).toBeInTheDocument();

    fireEvent.keyDown(screen.getByRole("dialog", { name: "Command palette" }), {
      key: "ArrowDown",
    });
    expect(input).toHaveAttribute("aria-activedescendant", "palette-option-findings");

    fireEvent.keyDown(screen.getByRole("dialog", { name: "Command palette" }), { key: "Tab" });
    expect(screen.getByRole("button", { name: "Close command palette" })).toHaveFocus();
    fireEvent.keyDown(screen.getByRole("dialog", { name: "Command palette" }), { key: "Tab" });
    expect(input).toHaveFocus();
  });

  it("restores focus to the Commands trigger after Escape", async () => {
    render(<CommandPalette actions={actions} />);
    const opener = screen.getByRole("button", { name: "Open command palette" });
    opener.focus();
    fireEvent.click(opener);

    expect(screen.getByRole("combobox", { name: "Filter commands" })).toHaveFocus();

    fireEvent.keyDown(window, { key: "Escape" });

    await waitFor(() => {
      expect(screen.queryByRole("dialog", { name: "Command palette" })).not.toBeInTheDocument();
      expect(opener).toHaveFocus();
    });
  });
});
