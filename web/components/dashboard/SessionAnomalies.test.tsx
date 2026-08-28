import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  SessionAnomaliesBanner,
  summariseAnomalies,
} from "@/components/dashboard/SessionAnomalies";
import type { SessionEntry } from "@/lib/types";

function session(id: string, flags: string[]): SessionEntry {
  return {
    session_id: id,
    actor: { display_name: "agent", email: "" },
    started_at: "2026-05-28T09:00:00Z",
    closed_at: null,
    exchange_count: 1,
    inferred_profile: "atb.profile.privileged_tool_action",
    cas_grade: "Insufficient",
    anomaly_flags: flags,
    bundle_path: "/tmp/b.atb",
  };
}

describe("summariseAnomalies", () => {
  it("counts sessions per distinct flag, most frequent first", () => {
    const summary = summariseAnomalies([
      session("a", ["tool_without_approval"]),
      session("b", ["tool_without_approval", "session_not_closed"]),
      session("c", []),
    ]);
    expect(summary[0]).toMatchObject({ flag: "tool_without_approval", sessions: 2 });
    expect(summary.find((s) => s.flag === "session_not_closed")?.sessions).toBe(1);
  });

  it("counts a duplicated flag within one session once", () => {
    const summary = summariseAnomalies([
      session("a", ["tool_without_approval", "tool_without_approval"]),
    ]);
    expect(summary).toEqual([
      {
        flag: "tool_without_approval",
        label: "No matching earlier approval in captured evidence",
        sessions: 1,
      },
    ]);
  });

  it("returns empty for clean sessions", () => {
    expect(summariseAnomalies([session("a", [])])).toEqual([]);
  });
});

describe("SessionAnomaliesBanner", () => {
  it("renders a banner listing anomalies", () => {
    render(<SessionAnomaliesBanner sessions={[session("a", ["tool_without_approval"])]} />);
    const banner = screen.getByTestId("session-anomalies");
    expect(banner.textContent).toContain("No matching earlier approval in captured evidence");
    expect(banner.textContent).toContain("1 session");
  });

  it("renders nothing when there are no anomalies", () => {
    const { container } = render(<SessionAnomaliesBanner sessions={[session("a", [])]} />);
    expect(container.firstChild).toBeNull();
  });

  it("labels the ai.* detection anomalies", () => {
    render(
      <SessionAnomaliesBanner
        sessions={[session("a", ["policy_denied_executed", "action_failed"])]}
      />,
    );
    const banner = screen.getByTestId("session-anomalies");
    expect(banner.textContent).toContain("Policy denied but action executed");
    expect(banner.textContent).toContain("Privileged action failed");
  });
});
