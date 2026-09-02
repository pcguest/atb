import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useUIStore } from "@/lib/state/ui-store";

vi.mock("@/app/view/components/ui/skeleton", () => ({
  Skeleton: () => <div data-testid="skeleton" />,
}));
vi.mock("@/components/dashboard/EventInspector", () => ({
  EventInspector: () => <div data-testid="event-inspector" />,
}));
vi.mock("@/components/dashboard/TraceGraph", () => ({
  TraceGraph: () => <div data-testid="trace-graph" />,
}));

const overview: {
  bundle_path: string;
  event_count: number;
  integrity_valid: boolean;
  integrity_status: string;
  profile: {
    profile_id: string;
    pass: boolean;
    coverage_score?: number;
    coverage_grade?: string;
    critical_failures: unknown[];
    warnings: string[];
  } | null;
  finding_count: number;
  critical_findings: number;
  custody_state: string;
  summary: string;
} = {
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
const trust = {
  proof_statement: "ATB proves the integrity and order of the records presented in a bundle.",
  integrity_valid: true,
  canonicalisation: "rfc8785",
  signature_status: "absent",
  anchor_status: "absent",
  profile_pass: false,
  coverage_score: 0 as number | undefined,
  coverage_grade: "",
  assessment_coverage: 0,
  assurance_valid: false,
  external_corroboration: false,
  custody_state: "Local only",
  limitations: ["ATB does not prove complete capture."],
};

vi.mock("@/lib/api-client", () => ({
  useInvestigationOverviewQuery: () => ({ data: overview, isLoading: false, isError: false }),
  useInvestigationTrustQuery: () => ({
    data: trust,
  }),
  useInvestigationFindingsQuery: () => ({ data: { findings: [finding] } }),
  useInvestigationTimelineQuery: () => ({
    data: {
      events: [
        {
          seq: 2,
          type: "atb.tool.call",
          label: "Captured tool call",
          timestamp: "2026-09-01T00:00:00Z",
          hash: "sha256:tool",
          family: "tool",
          causal_edge: false,
        },
      ],
    },
  }),
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
afterEach(() => {
  cleanup();
  overview.integrity_valid = true;
  overview.profile = null;
  trust.coverage_score = 0;
  trust.coverage_grade = "";
});

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
    expect(screen.getByText("Integrity")).toBeInTheDocument();
    expect(screen.getByText("Coverage")).toBeInTheDocument();
    expect(screen.getByText("Corroboration")).toBeInTheDocument();
    expect(screen.getByText("Hash chain verified")).toBeInTheDocument();
  });

  it("does not render omitted coverage as 0%", () => {
    overview.profile = {
      profile_id: "atb.profile.privileged_tool_action",
      pass: true,
      coverage_score: 0,
      coverage_grade: "",
      critical_failures: [],
      warnings: [],
    };
    render(<ViewPage />);
    expect(screen.getByText("Evidence coverage")).toBeInTheDocument();
    expect(screen.getByText("Not assessed")).toBeInTheDocument();
    expect(screen.queryByText("0%")).not.toBeInTheDocument();
  });

  it("renders assessed coverage as a percentage", () => {
    overview.profile = {
      profile_id: "atb.profile.privileged_tool_action",
      pass: true,
      coverage_score: 0.76,
      coverage_grade: "Moderate coverage",
      critical_failures: [],
      warnings: [],
    };
    trust.coverage_score = 0.76;
    trust.coverage_grade = "Moderate coverage";
    render(<ViewPage />);
    expect(screen.getByText("76%")).toBeInTheDocument();
    expect(screen.queryByText("Not assessed")).not.toBeInTheDocument();
    fireEvent.click(screen.getAllByRole("button", { name: "Trust" })[0]);
    expect(screen.getByText("Moderate coverage")).toBeInTheDocument();
  });

  it("does not claim context was supplied to the model", () => {
    render(<ViewPage />);
    fireEvent.click(screen.getAllByRole("button", { name: "Context" })[0]);
    expect(screen.getByText("Context evidence")).toBeInTheDocument();
    expect(screen.queryByText(/Context supplied to model/i)).not.toBeInTheDocument();
    expect(
      screen.getByText(/does not prove retrieval did not occur/),
    ).toBeInTheDocument();
  });

  it("opens evidence from a finding sequence", () => {
    render(<ViewPage />);
    fireEvent.click(screen.getAllByRole("button", { name: "Findings" })[0]);
    fireEvent.click(screen.getByRole("button", { name: "#2" }));
    expect(screen.getByText("Exact records")).toBeInTheDocument();
  });

  it("opens evidence from a timeline row", () => {
    render(<ViewPage />);
    fireEvent.click(screen.getAllByRole("button", { name: "Timeline" })[0]);
    fireEvent.click(screen.getByRole("button", { name: /Captured tool call/ }));
    expect(screen.getByText("Exact records")).toBeInTheDocument();
  });

  it("keeps the relationship graph collapsed", () => {
    render(<ViewPage />);
    fireEvent.click(screen.getAllByRole("button", { name: "Relationships" })[0]);
    expect(screen.queryByTestId("trace-graph")).not.toBeInTheDocument();
    expect(screen.getByText(/Optional graph of the same rows/)).toBeInTheDocument();
  });
});
