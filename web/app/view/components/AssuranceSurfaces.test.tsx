import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ContextSurface, RelationshipsSurface, TrustSurface } from "./AssuranceSurfaces";
import { investigationTrustSchema } from "@/lib/schemas/investigation";

vi.mock("@/components/dashboard/TraceGraph", () => ({
  TraceGraph: ({ graph }: { graph: { edges: unknown[] } }) => (
    <div data-testid="relationship-graph">{graph.edges.length} edges</div>
  ),
}));
afterEach(cleanup);

describe("assurance inspection", () => {
  it("keeps missing lineage bounded and links observed retrieval", () => {
    const open = vi.fn();
    render(
      <ContextSurface
        data={{
          lineage: { units: [], operations: [], warnings: ["Recorded reference unavailable"] },
          capabilities: [
            {
              name: "retrieval",
              mapping_version: "1",
              raw_event_type: "atb.event.rag_search",
              event_sequence: 7,
              fields: {},
            },
          ],
        }}
        onOpenEvidence={open}
      />,
    );
    expect(screen.getByText(/This does not prove retrieval did not occur/)).toBeVisible();
    expect(screen.getByText(/does not establish invocation binding/)).toBeVisible();
    expect(screen.getByText("Recorded reference unavailable")).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Open evidence #7" }));
    expect(open).toHaveBeenCalledWith(7);
  });

  it("keeps recorded retrieval distinct from invocation binding", () => {
    render(
      <ContextSurface
        data={{
          lineage: {
            units: [],
            warnings: [],
            operations: [
              {
                id: "assemble-1",
                type: "assemble",
                input_unit_ids: ["unit-1"],
                output_unit_ids: [],
                event_sequence: 8,
              },
            ],
          },
          capabilities: [
            {
              name: "retrieval",
              mapping_version: "1",
              raw_event_type: "atb.event.rag_retrieval",
              event_sequence: 7,
              fields: {},
            },
          ],
        }}
        onOpenEvidence={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: /Assembled context/ }));
    expect(screen.getByText(/No invocation binding was recorded/)).toBeVisible();
    expect(screen.queryByText(/Bound to recorded invocation/)).toBeNull();
  });

  it("offers both relationship endpoints and filters the graph with the rows", () => {
    const open = vi.fn();
    render(
      <RelationshipsSurface
        data={{
          relationships: [
            {
              id: "r",
              source_seq: 2,
              target_seq: 5,
              kind: "shared_identifier",
              evidence_value: "request-42",
              strength: "derived",
            },
          ],
        }}
        timeline={[]}
        onOpenEvidence={open}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Open evidence #2" }));
    fireEvent.click(screen.getByRole("button", { name: "Open evidence #5" }));
    expect(open.mock.calls).toEqual([[2], [5]]);
    expect(screen.queryByTestId("relationship-graph")).toBeNull();
    fireEvent.click(screen.getByText(/Open optional graph/));
    expect(screen.getByTestId("relationship-graph")).toHaveTextContent("1 edges");
    fireEvent.change(screen.getByRole("textbox", { name: "Filter relationships" }), {
      target: { value: "missing" },
    });
    expect(screen.getByText("No matching relationships")).toBeVisible();
    expect(screen.getByTestId("relationship-graph")).toHaveTextContent("0 edges");
  });

  it("offers record paths from relationship rows and bounds an empty relationship set", () => {
    const open = vi.fn();
    const { rerender } = render(
      <RelationshipsSurface
        data={{
          relationships: [
            {
              id: "r",
              source_seq: 2,
              target_seq: 5,
              kind: "request_id",
              evidence_value: "request-42",
              strength: "semantic",
            },
          ],
        }}
        timeline={[]}
        onOpenEvidence={open}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Source #2" }));
    fireEvent.click(screen.getByRole("button", { name: "Target #5" }));
    expect(open.mock.calls).toEqual([[2], [5]]);
    rerender(<RelationshipsSurface data={{ relationships: [] }} timeline={[]} onOpenEvidence={open} />);
    expect(screen.getByText("No derived relationships")).toBeVisible();
  });

  it("does not invent supporting record links for trust summaries", () => {
    const data = investigationTrustSchema.parse({
      proof_statement: "Recorded integrity and order",
      integrity_valid: true,
      canonicalisation: "rfc8785",
      signature_status: "absent",
      anchor_status: "absent",
      profile_pass: false,
      assurance_valid: false,
      external_corroboration: false,
      custody_state: "Local only",
      limitations: ["Not complete capture"],
    });
    render(<TrustSurface data={data} timeline={[]} onOpenEvidence={vi.fn()} />);
    expect(screen.getAllByText("Absent from recorded evidence")).not.toHaveLength(0);
    expect(screen.getByText("Not assessed")).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: /Corroboration/ }));
    expect(screen.queryByRole("button", { name: /Open evidence/ })).toBeNull();
  });

  it("keeps corroboration links and secondary assurance states subordinate", () => {
    const open = vi.fn();
    const data = investigationTrustSchema.parse({
      proof_statement: "Recorded integrity and order",
      integrity_valid: true,
      canonicalisation: "rfc8785",
      signature_status: "verified",
      anchor_status: "absent",
      profile_pass: false,
      assurance_valid: false,
      external_corroboration: true,
      custody_state: "External receipt recorded",
      limitations: [],
    });
    render(
      <TrustSurface
        data={data}
        timeline={[
          {
            seq: 11,
            type: "atb.corroboration.external",
            label: "External receipt",
            hash: "sha256:11",
            family: "corroboration",
            causal_edge: false,
          },
        ]}
        onOpenEvidence={open}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: /Corroboration/ }));
    fireEvent.click(screen.getByRole("button", { name: "Open evidence #11" }));
    expect(open).toHaveBeenCalledWith(11);
    expect(screen.getByText("verified")).toBeVisible();
    expect(screen.getAllByText("Absent from recorded evidence")).not.toHaveLength(0);
    expect(screen.getAllByText("External receipt recorded")).not.toHaveLength(0);
    expect(screen.getByText(/Bundle canonicalisation contract/)).toBeVisible();
  });
});
