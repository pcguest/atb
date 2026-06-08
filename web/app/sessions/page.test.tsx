import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";

import SessionsPage from "./page";

// The page's value is that it mounts the previously-orphaned session and schema
// views. Stub them so this test asserts the wiring (they are rendered) rather
// than their internals, which are covered by their own component tests.
vi.mock("@/components/SessionList", () => ({
  default: () => <div data-testid="session-list" />,
}));
vi.mock("@/components/SchemaStatus", () => ({
  default: () => <div data-testid="schema-status" />,
}));
vi.mock("@/components/ActorSessions", () => ({
  default: () => <div data-testid="actor-sessions" />,
}));

describe("SessionsPage", () => {
  it("mounts the session, schema-status, and actor-grouped views", () => {
    render(<SessionsPage />);
    expect(screen.getByTestId("session-list")).toBeInTheDocument();
    expect(screen.getByTestId("schema-status")).toBeInTheDocument();
    expect(screen.getByTestId("actor-sessions")).toBeInTheDocument();
  });

  it("documents how the workspace index is populated", () => {
    render(<SessionsPage />);
    expect(screen.getByText(/atb view --sessions/)).toBeInTheDocument();
  });
});
