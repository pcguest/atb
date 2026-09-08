import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import Home from "./page";

describe("public homepage", () => {
  it("states the local evidence boundary and leads with the investigation product", () => {
    render(<Home />);

    expect(
      screen.getByRole("heading", { name: "Know what happened. Verify what was recorded." }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/local-first evidence system for AI agents/i),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("img", { name: /ATB View investigating the recorded incident/i }),
    ).toHaveAttribute("src", expect.stringContaining("/product/investigation.png"));
    expect(
      screen.getByRole("link", { name: "Open the full-size ATB View investigation screenshot" }),
    ).toHaveAttribute("href", "/product/investigation.png");
    expect(screen.getByText(/does not prove complete capture/i)).toBeInTheDocument();
  });
});
