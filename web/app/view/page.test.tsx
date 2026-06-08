import { render, screen, cleanup } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useUIStore } from "@/lib/state/ui-store";

// The dashboard children are covered by their own tests; stub them so this test
// asserts the page wiring — that RoleSelector is mounted and that the role it
// controls gates the auditor/executive panels — not their internals. RoleSelector,
// the UI store, and the roles helpers are kept REAL: they are what is under test.
vi.mock("@/app/view/components/AuditorCompliancePanel", () => ({
  AuditorCompliancePanel: () => <div data-testid="auditor-panel" />,
}));
vi.mock("@/app/view/components/ExecutiveSummaryPanel", () => ({
  ExecutiveSummaryPanel: () => <div data-testid="executive-panel" />,
}));
vi.mock("@/app/view/components/ui/skeleton", () => ({
  Skeleton: () => <div data-testid="skeleton" />,
}));
vi.mock("@/components/dashboard/EventInspector", () => ({
  EventInspector: () => <div data-testid="event-inspector" />,
}));
vi.mock("@/components/dashboard/ProfileCAS", () => ({
  ProfileCAS: () => <div data-testid="profile-cas" />,
}));
vi.mock("@/components/dashboard/SessionAnomalies", () => ({
  SessionAnomalies: () => <div data-testid="session-anomalies" />,
}));
vi.mock("@/components/dashboard/StatsOverview", () => ({
  StatsOverview: () => <div data-testid="stats-overview" />,
}));
vi.mock("@/components/dashboard/TraceGraph", () => ({
  TraceGraph: () => <div data-testid="trace-graph" />,
}));
vi.mock("@/components/dashboard/TraceTimeline", () => ({
  TraceTimeline: () => <div data-testid="trace-timeline" />,
}));

vi.mock("@/lib/api-client", () => ({
  useVerificationQuery: () => ({ data: { status: "valid" }, isError: false, error: null }),
  useBundleMetaQuery: () => ({
    data: { bundle_path: "demo.atb", last_timestamp: null },
    isError: false,
  }),
  useBundleEventsQuery: () => ({
    data: { pages: [] },
    isError: false,
    isLoading: false,
    isFetching: false,
    hasNextPage: false,
    isFetchingNextPage: false,
    fetchNextPage: vi.fn(),
  }),
  useBundleGraphQuery: () => ({ data: null, isError: false, isLoading: false, isFetching: false }),
  useRevealFieldMutation: () => ({ mutateAsync: vi.fn(), isPending: false }),
  flattenEventPages: () => [],
}));

import ViewPage from "./page";

beforeEach(() => {
  // Reset to the default role between cases; the store persists to localStorage.
  useUIStore.setState({ role: "engineer" });
});

afterEach(() => {
  cleanup();
});

describe("ViewPage role selector", () => {
  it("mounts the role selector in the dashboard header", () => {
    render(<ViewPage />);
    expect(screen.getByTestId("role-selector")).toBeInTheDocument();
  });

  it("shows the executive summary only for the executive role", () => {
    useUIStore.setState({ role: "executive" });
    render(<ViewPage />);
    expect(screen.getByTestId("executive-panel")).toBeInTheDocument();
    expect(screen.queryByTestId("auditor-panel")).not.toBeInTheDocument();
  });

  it("shows the auditor compliance panel only for the auditor role", () => {
    useUIStore.setState({ role: "auditor" });
    render(<ViewPage />);
    expect(screen.getByTestId("auditor-panel")).toBeInTheDocument();
    expect(screen.queryByTestId("executive-panel")).not.toBeInTheDocument();
  });

  it("hides both role panels for the engineer role", () => {
    useUIStore.setState({ role: "engineer" });
    render(<ViewPage />);
    expect(screen.queryByTestId("auditor-panel")).not.toBeInTheDocument();
    expect(screen.queryByTestId("executive-panel")).not.toBeInTheDocument();
  });
});
