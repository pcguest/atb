import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { FindingsSurface, RecordSurface } from "./CoreSurfaces";
import type { AcquisitionFinding, InvestigationFinding, TimelineEvent } from "@/lib/types";

const findings: InvestigationFinding[] = [
  {
    flag: "missing_approval",
    severity: "high",
    title: "Approval was not recorded",
    detail: "No matching approval exists in the recorded evidence.",
    basis: "event sequence",
    boundedness: "recorded_evidence_only",
    what_atb_can_conclude: "No recorded approval matches this action.",
    what_atb_cannot_conclude: "ATB cannot prove universal absence.",
    event_seqs: [42],
  },
  {
    flag: "review",
    severity: "low",
    title: "Review required",
    detail: "A reviewer should inspect the captured context.",
    basis: "event sequence",
    boundedness: "recorded_evidence_only",
    what_atb_can_conclude: "Review is warranted.",
    what_atb_cannot_conclude: "ATB cannot assess omitted records.",
    event_seqs: [],
  },
];

const timeline: TimelineEvent[] = [
  { seq: 41, type: "atb.context.unit", label: "Context unit recorded", hash: "sha256:one", family: "context", causal_edge: false },
  { seq: 42, type: "atb.tool.call", label: "Captured tool call", timestamp: "2026-09-01T00:00:00Z", hash: "sha256:two", family: "tool", causal_edge: false },
];

describe("forensic core surfaces", () => {
  it("selects a finding and opens its supporting record in Evidence or Timeline", () => {
    const select = vi.fn();
    const openEvidence = vi.fn();
    const openTimeline = vi.fn();
    render(<FindingsSurface findings={findings} selected={0} select={select} openEvidence={openEvidence} openTimeline={openTimeline} />);

    fireEvent.click(screen.getByRole("button", { name: /Review required/ }));
    expect(select).toHaveBeenCalledWith(1);
    expect(screen.getByText("ATB can conclude")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "#42" }));
    fireEvent.click(screen.getByRole("button", { name: "Inspect in timeline →" }));
    expect(openEvidence).toHaveBeenCalledWith(42);
    expect(openTimeline).toHaveBeenCalledWith(42);
  });

  it("searches and filters findings only by their recorded severity", () => {
    render(<FindingsSurface findings={findings} selected={0} select={vi.fn()} openEvidence={vi.fn()} openTimeline={vi.fn()} />);
    fireEvent.change(screen.getByRole("textbox", { name: "Search findings" }), { target: { value: "review" } });
    expect(screen.getByRole("button", { name: /Review required/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Approval was not recorded/ })).not.toBeInTheDocument();
    fireEvent.change(screen.getByRole("combobox", { name: "Finding severity" }), { target: { value: "high" } });
    expect(screen.getByText("No matching findings")).toBeInTheDocument();
  });

  it("searches event identity and type, filters types, and marks missing recorded time", () => {
    const select = vi.fn();
    render(<RecordSurface timeline={timeline} selectedSeq={41} select={select} evidence={false} inspector={<div>Exact inspector</div>} />);
    expect(screen.getByText("time unavailable")).toBeInTheDocument();
    fireEvent.change(screen.getByRole("textbox", { name: "Search event identity, type or hash" }), { target: { value: "sha256:two" } });
    expect(screen.getByText("Captured tool call")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Captured tool call/ }));
    expect(select).toHaveBeenCalledWith(42);
    fireEvent.change(screen.getByRole("combobox", { name: "Event type" }), { target: { value: "atb.context.unit" } });
    expect(screen.getByText("No matching records")).toBeInTheDocument();
  });

  it("moves keyboard focus to the selected timeline record", () => {
    render(<RecordSurface timeline={timeline} selectedSeq={42} select={vi.fn()} evidence={false} inspector={<div>Exact inspector</div>} />);
    expect(screen.getByRole("button", { name: /Captured tool call/ })).toHaveFocus();
  });

  it("shows acquisition continuity findings as bounded observations", () => {
    const acquisitionFindings: AcquisitionFinding[] = [{
      flag: "source_record_changed",
      severity: "high",
      title: "Source record representation changed",
      detail: "Source record chatlog:r2 was previously acquired with a different representation digest.",
      source_system: "chatlog",
      source_record_id: "r2",
      previous_digest: "sha256:prev",
      current_digest: "sha256:curr",
      previous_acquired_at: "2026-01-01T09:00:00Z",
      current_acquired_at: "2026-01-02T09:00:00Z",
      adapter: "atb.chatlog.generic-jsonl",
      adapter_version: "1.0.0",
      event_seq: 7,
      boundedness: "bounded",
      what_atb_can_conclude: "The source representation differs between acquisitions.",
      what_atb_cannot_conclude: "ATB does not establish why it changed or that it was malicious.",
    }];
    render(<FindingsSurface findings={findings} acquisitionFindings={acquisitionFindings} selected={0} select={vi.fn()} openEvidence={vi.fn()} openTimeline={vi.fn()} />);
    const section = screen.getByTestId("acquisition-findings");
    expect(section.textContent).toContain("Acquisition continuity findings");
    expect(section.textContent).toContain("chatlog:r2");
    expect(section.textContent).toContain("sha256:prev");
    expect(section.textContent).toContain("sha256:curr");
    expect(section.textContent).toContain("not proof of tampering");
  });
});
