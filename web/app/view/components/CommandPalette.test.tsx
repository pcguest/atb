import React from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { CommandPalette, type PaletteAction } from "./CommandPalette";

const actions: PaletteAction[] = [
  { id: "verify", label: "Verify bundle", group: "verify", run: vi.fn() },
  { id: "findings", label: "Open findings", group: "navigate", run: vi.fn() },
  { id: "copy-digest", label: "Copy selected evidence digest", group: "copy", run: vi.fn() },
];

describe("CommandPalette", () => {
  it("opens with Ctrl+K and Cmd+K", () => {
    const { unmount } = render(<CommandPalette actions={actions} />);
    fireEvent.keyDown(window, { key: "k", ctrlKey: true });
    expect(screen.getByRole("dialog", { name: "Command palette" })).toBeInTheDocument();
    fireEvent.keyDown(window, { key: "k", ctrlKey: true });
    expect(screen.queryByRole("dialog", { name: "Command palette" })).not.toBeInTheDocument();
    unmount();

    render(<CommandPalette actions={actions} />);
    fireEvent.keyDown(window, { key: "k", metaKey: true });
    expect(screen.getByRole("dialog", { name: "Command palette" })).toBeInTheDocument();
  });

  it("uses displayed group order for keyboard selection and execution", async () => {
    const copy = vi.fn();
    const navigate = vi.fn();
    render(<CommandPalette actions={[
      { id: "copy", label: "Copy hash", group: "copy", run: copy },
      { id: "navigate", label: "Open Evidence", group: "navigate", run: navigate },
    ]} />);
    fireEvent.click(screen.getByRole("button", { name: "Open command palette" }));
    expect(screen.getByRole("combobox")).toHaveAttribute("aria-activedescendant", "palette-option-navigate");
    fireEvent.keyDown(screen.getByRole("combobox"), { key: "ArrowDown" });
    fireEvent.keyDown(screen.getByRole("combobox"), { key: "Enter" });
    await waitFor(() => expect(copy).toHaveBeenCalledOnce());
    expect(navigate).not.toHaveBeenCalled();
  });

  it("announces asynchronous failure without an unhandled rejection", async () => {
    render(<CommandPalette actions={[{ id: "fail", label: "Verify bundle", group: "verify", run: async () => { throw new Error("unavailable"); } }]} />);
    fireEvent.click(screen.getByRole("button", { name: "Open command palette" }));
    fireEvent.keyDown(screen.getByRole("combobox"), { key: "Enter" });
    expect(await screen.findByRole("alert")).toHaveTextContent("Verify bundle failed. Try again from Commands.");
  });

  it("traps focus and moves aria-activedescendant with arrow keys", () => {
    render(<CommandPalette actions={actions} />);
    fireEvent.click(screen.getByRole("button", { name: "Open command palette" }));

    const input = screen.getByRole("combobox", { name: "Filter commands" });
    expect(input).toHaveAttribute("aria-activedescendant", "palette-option-findings");
    expect(screen.getByRole("option", { name: "Open findings" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.getByText("Navigate")).toBeInTheDocument();
    expect(screen.getByText("Verify")).toBeInTheDocument();
    expect(screen.getByText("Copy")).toBeInTheDocument();
    expect(screen.queryByText("Operate")).not.toBeInTheDocument();

    fireEvent.keyDown(screen.getByRole("dialog", { name: "Command palette" }), {
      key: "ArrowDown",
    });
    expect(input).toHaveAttribute("aria-activedescendant", "palette-option-verify");

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
