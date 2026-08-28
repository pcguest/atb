import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach, type Mock } from "vitest";
import { useRouter } from "next/navigation";
import SessionList from "./SessionList";
import { useSessionsQuery } from "@/lib/api-client";

// Mock the api-client hook so the component is tested through the authenticated
// data path it actually uses (the hook attaches the session token).
vi.mock("@/lib/api-client", () => ({
  useSessionsQuery: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useRouter: vi.fn(),
}));

const mockSessions = [
  {
    session_id: "s1",
    actor: { display_name: "User One", email: "user1@example.com" },
    started_at: "2026-05-28T10:00:00Z",
    closed_at: "2026-05-28T10:05:00Z",
    exchange_count: 5,
    inferred_profile: "atb.profile.privileged_tool_action",
    cas_grade: "High",
    anomaly_flags: [],
    bundle_path: "/path/to/bundle1.atb",
  },
  {
    session_id: "s2",
    actor: { display_name: "api-key:abcd", email: "" },
    started_at: "2026-05-28T11:00:00Z",
    closed_at: null,
    exchange_count: 2,
    inferred_profile: "atb.profile.data_export",
    cas_grade: "Low",
    anomaly_flags: ["unresolved_identity", "session_not_closed"],
    bundle_path: "/path/to/bundle2.atb",
  },
];

describe("SessionList", () => {
  const mockPush = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    (useRouter as Mock).mockReturnValue({
      push: mockPush,
    });
    // Mock window.location.hash for navigation tests
    Object.defineProperty(window, "location", {
      value: {
        hash: "#session=test-token",
      },
      writable: true,
    });
  });

  it("renders loading state", () => {
    (useSessionsQuery as Mock).mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    });
    render(<SessionList />);
    expect(screen.getByText("Loading sessions…")).toBeInTheDocument();
  });

  it("renders error state", () => {
    (useSessionsQuery as Mock).mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error("Failed to fetch"),
    });
    render(<SessionList />);
    expect(screen.getByText("Error: Failed to fetch")).toBeInTheDocument();
  });

  it("renders no sessions found state", () => {
    (useSessionsQuery as Mock).mockReturnValue({
      data: [],
      isLoading: false,
      error: null,
    });
    render(<SessionList />);
    expect(screen.getByText("No sessions found.")).toBeInTheDocument();
  });

  it("renders sessions correctly", async () => {
    (useSessionsQuery as Mock).mockReturnValue({
      data: mockSessions,
      isLoading: false,
      error: null,
    });
    render(<SessionList />);

    await waitFor(() => {
      expect(screen.getByText("User One")).toBeInTheDocument();
      expect(screen.getByText("api-key:abcd")).toBeInTheDocument();
      expect(screen.getByText("privileged_tool_action")).toBeInTheDocument();
      expect(screen.getByText("data_export")).toBeInTheDocument();
      expect(screen.getByText("High")).toBeInTheDocument();
      expect(screen.getByText("Low")).toBeInTheDocument();
      expect(
        screen.getByTitle("Actor identity is unresolved (e.g., API key)."),
      ).toBeInTheDocument();
      expect(screen.getByTitle("Session has not been explicitly closed.")).toBeInTheDocument();
    });

    expect(screen.getByRole("table", { name: "Sessions" })).toHaveAttribute("aria-rowcount", "3");
    expect(screen.getAllByRole("columnheader")).toHaveLength(6);
    expect(screen.getAllByRole("row")).toHaveLength(3);
    expect(
      screen.getByRole("button", {
        name: /Open session s1 for User One, started/,
      }),
    ).toBeInTheDocument();
  });

  it("navigates to bundle viewer on row click, preserving the session token", async () => {
    (useSessionsQuery as Mock).mockReturnValue({
      data: mockSessions,
      isLoading: false,
      error: null,
    });
    render(<SessionList />);

    await waitFor(() => {
      fireEvent.click(screen.getByText("User One"));
    });

    expect(mockPush).toHaveBeenCalledWith(
      "/view?bundlePath=%2Fpath%2Fto%2Fbundle1.atb#session=test-token",
    );
  });

  it.each(["Enter", " "])(
    "navigates from a focused session control with the %j key",
    async (key) => {
      const user = userEvent.setup();
      (useSessionsQuery as Mock).mockReturnValue({
        data: mockSessions,
        isLoading: false,
        error: null,
      });
      render(<SessionList />);

      const sessionControl = await screen.findByRole("button", {
        name: /Open session s1 for User One, started/,
      });
      sessionControl.focus();
      await user.keyboard(key === " " ? "[Space]" : "[Enter]");

      expect(mockPush).toHaveBeenCalledWith(
        "/view?bundlePath=%2Fpath%2Fto%2Fbundle1.atb#session=test-token",
      );
    },
  );
});
