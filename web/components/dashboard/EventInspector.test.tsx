import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { EventInspector } from "@/components/dashboard/EventInspector";
import type { EventRecord } from "@/lib/types";

const noReveal = async () => ({});

function makeEvent(type: string, data: Record<string, unknown>): EventRecord {
  return {
    seq: 3,
    type,
    hash: "a".repeat(64),
    prev_hash: "b".repeat(64),
    timestamp: "2026-05-28T09:00:00Z",
    data,
  } as EventRecord;
}

describe("EventInspector", () => {
  it("shows a one-line summary for forensic events", () => {
    render(
      <EventInspector
        event={makeEvent("ai.action.error", { action_id: "toolu_1", error_class: "failed" })}
        onReveal={noReveal}
      />,
    );
    const summary = screen.getByTestId("event-summary");
    expect(summary.textContent).toContain("action=toolu_1 error_class=failed");
  });

  it("omits the summary line when there is nothing concise to say", () => {
    render(<EventInspector event={makeEvent("dev.session", { x: 1 })} onReveal={noReveal} />);
    expect(screen.queryByTestId("event-summary")).toBeNull();
  });
});
