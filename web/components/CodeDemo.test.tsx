import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import CodeDemo from "./CodeDemo";

describe("public integration examples", () => {
  it("keeps selection, focus and the named panel synchronized for keyboard review", async () => {
    const user = userEvent.setup();
    render(<CodeDemo />);
    const cli = screen.getByRole("tab", { name: "CLI (Go)" });
    cli.focus();
    await user.keyboard("{ArrowRight}");
    const python = screen.getByRole("tab", { name: "Python SDK" });
    expect(python).toHaveFocus();
    expect(python).toHaveAttribute("aria-selected", "true");
    expect(cli).toHaveAttribute("tabindex", "-1");
    expect(screen.getByRole("tabpanel", { name: "Python SDK" })).toHaveTextContent(
      "from atb import Bundle",
    );
    await user.keyboard("{End}");
    expect(screen.getByRole("tab", { name: "TypeScript SDK" })).toHaveFocus();
    await user.keyboard("{ArrowRight}");
    expect(cli).toHaveFocus();
    expect(screen.getByRole("tabpanel", { name: "CLI (Go)" })).toBeInTheDocument();
  });
});
