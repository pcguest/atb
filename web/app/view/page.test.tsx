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
const findings = [finding];
const context = {
  lineage: { units: [], operations: [], warnings: [] },
  capabilities: [] as Array<{
    name: string;
    mapping_version: string;
    event_sequence: number;
    raw_event_type: string;
    fields: Record<string, unknown>;
  }>,
};
let overviewLoading = false;
let overviewError = false;
let eventPages: Array<{ events: Array<{ seq: number; type: string; hash: string; prev_hash: string; data: Record<string, unknown> }> }> = [];
let eventsHasNextPage = false;
const fetchNextPage = vi.fn();
const trust = {
  proof_statement: "ATB proves the integrity and order of records presented in a bundle.",
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
  useInvestigationOverviewQuery: () => ({
    data: overview,
    isLoading: overviewLoading,
    isError: overviewError,
  }),
  useInvestigationTrustQuery: () => ({
    data: trust,
  }),
  useInvestigationFindingsQuery: () => ({
    data: { findings },
    isLoading: false,
    isError: false,
  }),
  useInvestigationTimelineQuery: () => ({
    data: {
      events: [
        { seq: 0, type: "atb.bundle.manifest", label: "Manifest", timestamp: "2026-09-01T00:00:00Z", hash: "manifest", family: "bundle", causal_edge: false },
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
  useInvestigationContextQuery: () => ({ data: context }),
  useInvestigationRelationshipsQuery: () => ({ data: { relationships: [] } }),
  useBundleEventsQuery: () => ({ data: { pages: eventPages }, hasNextPage: eventsHasNextPage, fetchNextPage, isLoading: false, isError: false, isFetchingNextPage: false }),
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
  trust.integrity_valid = true;
  trust.external_corroboration = false;
  trust.custody_state = "Local only";
  overview.custody_state = "Local only";
  overview.finding_count = 1;
  findings.splice(0, findings.length, finding);
  context.capabilities.splice(0);
  overviewLoading = false;
  overviewError = false;
  eventPages = [];
  eventsHasNextPage = false;
  fetchNextPage.mockReset();
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
    expect(screen.getAllByText("Hash chain verified")).not.toHaveLength(0);
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
    expect(screen.getByRole("region", { name: "Context evidence" })).toBeInTheDocument();
    expect(screen.queryByText(/Context supplied to model/i)).not.toBeInTheDocument();
    expect(screen.getByText(/does not prove retrieval did not occur/)).toBeInTheDocument();
  });

  it("opens evidence from a finding sequence", () => {
    render(<ViewPage />);
    fireEvent.click(screen.getAllByRole("button", { name: "Findings" })[0]);
    fireEvent.click(screen.getByRole("button", { name: "#2" }));
    expect(screen.getByText("Exact records")).toBeInTheDocument();
  });

  it("retains an unavailable selected finding event as an explicit evidence state", () => {
    findings[0] = { ...finding, event_seqs: [99] };
    render(<ViewPage />);
    fireEvent.click(screen.getAllByRole("button", { name: "Findings" })[0]);
    fireEvent.click(screen.getByRole("button", { name: "#99" }));
    expect(screen.getByText("Event #99 unavailable")).toBeInTheDocument();
  });

  it("loads subsequent event pages for a selected finding event", () => {
    findings[0] = { ...finding, event_seqs: [99] };
    eventsHasNextPage = true;
    render(<ViewPage />);
    fireEvent.click(screen.getAllByRole("button", { name: "Findings" })[0]);
    fireEvent.click(screen.getByRole("button", { name: "#99" }));
    expect(fetchNextPage).toHaveBeenCalled();
    expect(screen.getByText("Loading selected record…")).toBeInTheDocument();
  });

  it("opens evidence from a timeline row", () => {
    render(<ViewPage />);
    fireEvent.click(screen.getAllByRole("button", { name: "Timeline" })[0]);
    fireEvent.click(screen.getByRole("button", { name: /Captured tool call/ }));
    fireEvent.click(screen.getByRole("button", { name: "Open in Evidence →" }));
    expect(screen.getByText("Exact records")).toBeInTheDocument();
  });

  it("keeps the relationship graph collapsed", () => {
    render(<ViewPage />);
    fireEvent.click(screen.getAllByRole("button", { name: "Relationships" })[0]);
    expect(screen.queryByTestId("trace-graph")).not.toBeInTheDocument();
    expect(screen.getByText(/Open optional graph/)).toBeInTheDocument();
  });

  it("makes failed integrity visibly untrusted", () => {
    overview.integrity_valid = false;
    trust.integrity_valid = false;
    render(<ViewPage />);
    expect(screen.getByText("Integrity failed")).toBeInTheDocument();
    expect(screen.getByText("Untrusted")).toBeInTheDocument();
    fireEvent.click(screen.getAllByRole("button", { name: "Trust" })[0]);
    expect(screen.getAllByText("Hash chain failed")).not.toHaveLength(0);
  });

  it("blocks event-derived findings and timeline when integrity is invalid", () => {
    overview.integrity_valid = false;
    trust.integrity_valid = false;
    render(<ViewPage />);
    fireEvent.click(screen.getAllByRole("button", { name: "Findings" })[0]);
    expect(screen.getByText("Event-derived investigation is blocked")).toBeInTheDocument();
    fireEvent.click(screen.getAllByRole("button", { name: "Timeline" })[0]);
    expect(screen.getByText("Event-derived investigation is blocked")).toBeInTheDocument();
  });

  it("states the boundary when no findings are derived", () => {
    findings.splice(0);
    overview.finding_count = 0;
    render(<ViewPage />);
    expect(screen.getByText("No findings in recorded evidence")).toBeInTheDocument();
    expect(
      screen.getByText(/does not prove complete capture or universal absence/),
    ).toBeInTheDocument();
  });

  it("keeps a long finding readable and its evidence reachable", () => {
    const longDetail = "A bounded forensic explanation ".repeat(24).trim();
    findings[0] = { ...finding, detail: longDetail };
    render(<ViewPage />);
    expect(screen.getByText(longDetail)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "#2" })).toBeInTheDocument();
  });

  it("distinguishes retrieval evidence from missing structured lineage", () => {
    context.capabilities.push({
      name: "retrieval_query",
      mapping_version: "1",
      event_sequence: 2,
      raw_event_type: "atb.event.rag_query",
      fields: {},
    });
    render(<ViewPage />);
    fireEvent.click(screen.getAllByRole("button", { name: "Context" })[0]);
    expect(
      screen.getByText("Retrieval was recorded; structured lineage units were not."),
    ).toBeInTheDocument();
    expect(screen.getByText("Recorded retrieval capabilities")).toBeInTheDocument();
  });

  it("reports absent and present external custody without scoring it", () => {
    trust.external_corroboration = true;
    trust.custody_state = "External receipt recorded";
    overview.custody_state = "External receipt recorded";
    render(<ViewPage />);
    fireEvent.click(screen.getAllByRole("button", { name: "Trust" })[0]);
    expect(screen.getByText("External evidence present")).toBeInTheDocument();
    expect(screen.getAllByText("External receipt recorded")).not.toHaveLength(0);
    expect(screen.queryByText(/health score/i)).not.toBeInTheDocument();
  });

  it("renders explicit loading and recoverable error states", () => {
    overviewLoading = true;
    const { rerender } = render(<ViewPage />);
    expect(screen.getByRole("status")).toHaveTextContent("Loading investigation");
    overviewLoading = false;
    overviewError = true;
    rerender(<ViewPage />);
    expect(screen.getByRole("alert")).toHaveTextContent(/local viewer session/);
  });
});
