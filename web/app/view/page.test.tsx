import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useUIStore } from "@/lib/state/ui-store";

vi.mock("@/app/view/components/ui/skeleton", () => ({
  Skeleton: () => <div data-testid="skeleton" />,
}));
vi.mock("@/components/dashboard/EventInspector", () => ({
  EventInspector: () => <div data-testid="event-inspector" />,
}));
vi.mock("@/components/dashboard/ProfileCAS", () => ({
  ProfileCAS: () => <div data-testid="profile-cas" />,
}));
vi.mock("@/components/dashboard/TraceGraph", () => ({
  TraceGraph: () => <div data-testid="trace-graph" />,
}));

const overview = {
  bundle_path: "demo.atb",
  event_count: 3,
  integrity_valid: true,
  integrity_status: "VERIFIED",
  profile: null,
  finding_count: 1,
  critical_findings: 0,
  custody_state: "Local only",
  summary: "One bounded finding requires review.",
};
const finding = {
  flag: "tool_without_approval",
  severity: "high",
  title: "No matching approval",
  detail: "No matching earlier approval exists in the captured evidence.",
  basis: "captured session evidence",
  boundedness: "captured_evidence_only",
  what_atb_can_conclude: "No captured match exists.",
  what_atb_cannot_conclude: "ATB cannot prove universal absence.",
  event_seqs: [2],
};

vi.mock("@/lib/api-client", () => ({
  useInvestigationOverviewQuery: () => ({ data: overview, isLoading: false, isError: false }),
  useInvestigationTrustQuery: () => ({
    data: {
      proof_statement: "ATB proves the integrity and order of the records presented in a bundle.",
      integrity_valid: true,
      canonicalisation: "rfc8785",
      signature_status: "absent",
      anchor_status: "absent",
      profile_pass: false,
      coverage_score: 0,
      coverage_grade: "",
      assessment_coverage: 0,
      assurance_valid: false,
      external_corroboration: false,
      custody_state: "Local only",
      limitations: ["ATB does not prove complete capture."],
    },
  }),
  useInvestigationFindingsQuery: () => ({ data: { findings: [finding] } }),
  useInvestigationTimelineQuery: () => ({ data: { events: [] } }),
  useInvestigationContextQuery: () => ({
    data: { lineage: { units: [], operations: [], warnings: [] }, capabilities: [] },
  }),
  useInvestigationRelationshipsQuery: () => ({ data: { relationships: [] } }),
  useBundleEventsQuery: () => ({ data: { pages: [] }, hasNextPage: false, fetchNextPage: vi.fn() }),
  useBundleGraphQuery: () => ({ data: null, isFetching: false }),
  useRunBundleVerifyMutation: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useRevealFieldMutation: () => ({ mutateAsync: vi.fn(), isPending: false }),
  flattenEventPages: () => [],
}));

import ViewPage from "./page";

beforeEach(() => useUIStore.setState({ role: "engineer" }));
afterEach(cleanup);

describe("ATB View investigation model", () => {
  it("opens on the Incident summary rather than the graph", () => {
    render(<ViewPage />);
    expect(screen.getByText("What happened?")).toBeInTheDocument();
    expect(screen.getByText(overview.summary)).toBeInTheDocument();
    expect(screen.queryByTestId("trace-graph")).not.toBeInTheDocument();
  });

  it("provides every primary investigation surface", () => {
    render(<ViewPage />);
    for (const label of [
      "Incident",
      "Findings",
      "Timeline",
      "Context",
      "Relationships",
      "Evidence",
      "Trust",
    ]) {
      expect(screen.getAllByRole("button", { name: label }).length).toBeGreaterThan(0);
    }
  });

  it("keeps evidence visible in Auditor presentation mode", () => {
    useUIStore.setState({ role: "auditor" });
    render(<ViewPage />);
    fireEvent.click(screen.getAllByRole("button", { name: "Evidence" })[0]);
    expect(screen.getByText("Exact records")).toBeInTheDocument();
    expect(screen.queryByText(/available in the Engineer role/i)).not.toBeInTheDocument();
  });

  it("opens the same Trust action from navigation", () => {
    render(<ViewPage />);
    fireEvent.click(screen.getAllByRole("button", { name: "Trust" })[0]);
    expect(screen.getByText(/ATB proves the integrity and order/)).toBeInTheDocument();
    expect(screen.getByText("What ATB does not prove")).toBeInTheDocument();
  });
});
