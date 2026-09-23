import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

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

  it("overlays a revealed value onto inspector JSON after a successful reveal", async () => {
    const onReveal = vi.fn().mockResolvedValue("auditor@example.com");
    render(
      <EventInspector
        event={makeEvent("dev.session", { email: "[REDACTED]" })}
        onReveal={onReveal}
      />,
    );
    expect(screen.getByText(/\[REDACTED\]/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Reveal masked field data.email" }));
    await waitFor(() => {
      expect(onReveal).toHaveBeenCalledWith(3, "email");
    });
    await waitFor(() => {
      expect(document.querySelector("pre")?.textContent).toContain("auditor@example.com");
    });
    expect(document.querySelector("pre")?.textContent).not.toContain("[REDACTED]");
    expect(screen.queryByRole("button", { name: /Reveal masked field/ })).toBeNull();
  });

  it("handles long hashes and large JSON without expanding the layout", () => {
    const longValue = "evidence-segment-".repeat(256);
    render(
      <EventInspector
        event={makeEvent("ai.tool.exec", { payload: longValue, nested: { count: 4096 } })}
        onReveal={noReveal}
      />,
    );
    expect(screen.getAllByTitle("Click to copy full hash")[0]).toHaveTextContent(
      "aaaaaaaaaaaa…aaaaaaaaaaaa",
    );
    const json = document.querySelector("pre");
    expect(json).not.toBeNull();
    expect(json?.textContent).toContain(longValue);
    expect(json?.textContent).toContain('"prev_hash"');
    expect(json).toHaveClass("overflow-auto");
    expect(json).toHaveAttribute("tabindex", "0");
  });

  it("shows bounded acquisition provenance with honest raw-source limits", () => {
    const event = {
      ...makeEvent("ai.request.received", { request_id: "r2" }),
      acquisition: {
        mode: "retrospective",
        source_system: "chatlog",
        source_record_id: "r2",
        source_timestamp: "2026-01-01T10:00:02Z",
        acquired_at: "2026-01-02T09:00:00Z",
        adapter: "atb.chatlog.generic-jsonl",
        adapter_version: "1.0.0",
        source_digest: "sha256:" + "a".repeat(64),
        raw_source_available: false,
        checkpoint_position: "r2",
        checkpoint_status: "recorded",
      },
    } as EventRecord;
    render(<EventInspector event={event} onReveal={noReveal} />);
    const section = screen.getByTestId("acquisition-provenance");
    expect(section.textContent).toContain("retrospective");
    expect(section.textContent).toContain("chatlog");
    expect(section.textContent).toContain("r2");
    expect(section.textContent).toContain("Not retained; only the digest is recorded");
    expect(section.textContent).toContain("operational position r2");
    expect(section.textContent).toContain("not truth");
  });

  it("omits acquisition provenance when the record has none", () => {
    render(<EventInspector event={makeEvent("dev.session", { x: 1 })} onReveal={noReveal} />);
    expect(screen.queryByTestId("acquisition-provenance")).toBeNull();
  });

  it("does not apply a delayed reveal to a different selected event", async () => {
    let resolveReveal: ((value: unknown) => void) | undefined;
    const onReveal = vi.fn(() => new Promise<unknown>((resolve) => { resolveReveal = resolve; }));
    const { rerender } = render(
      <EventInspector event={makeEvent("dev.session", { email: "[REDACTED]" })} onReveal={onReveal} />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Reveal masked field data.email" }));
    rerender(
      <EventInspector
        event={{ ...makeEvent("dev.session", { email: "second@example.com" }), seq: 4, hash: "c".repeat(64) }}
        onReveal={onReveal}
      />,
    );
    resolveReveal?.("first@example.com");

    await waitFor(() => {
      expect(document.querySelector("pre")?.textContent).toContain("second@example.com");
    });
    expect(document.querySelector("pre")?.textContent).not.toContain("first@example.com");
  });
});
