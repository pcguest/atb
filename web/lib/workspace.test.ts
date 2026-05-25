import { describe, expect, it } from "vitest";

import { workspaceBundlesResponseSchema } from "@/lib/schemas/workspace";
import { bundleDisplayId, bundleViewHref } from "@/lib/workspace-nav";

describe("workspaceBundlesResponseSchema", () => {
  it("accepts agent workspace bundle list payloads", () => {
    const parsed = workspaceBundlesResponseSchema.parse({
      bundles: [
        {
          id: "sess_abc123",
          session_id: "sess_abc123",
          bundle_path: "/tmp/sessions/sess_abc123/bundle.atb",
          profile_id: "atb.profile.rag_answer",
          head_hash: "deadbeef".padEnd(64, "0"),
          event_count: 7,
          opened_at: "2026-05-25T06:00:00.000000000Z",
          closed_at: "2026-05-25T06:01:23.000000000Z",
        },
      ],
    });

    expect(parsed.bundles).toHaveLength(1);
    expect(parsed.bundles[0]?.session_id).toBe("sess_abc123");
  });
});

describe("workspace navigation helpers", () => {
  const bundle = {
    id: "sess_abc123",
    session_id: "sess_abc123",
    bundle_path: "/tmp/sessions/sess_abc123/bundle.atb",
    profile_id: "atb.profile.rag_answer",
    head_hash: "deadbeef".padEnd(64, "0"),
    event_count: 3,
    opened_at: "2026-05-25T06:00:00.000000000Z",
    closed_at: "2026-05-25T06:01:23.000000000Z",
  };

  it("builds agent-mode view links with session_id query params", () => {
    expect(bundleViewHref(bundle, true)).toBe(
      "/view/?session_id=sess_abc123&bundle_path=%2Ftmp%2Fsessions%2Fsess_abc123%2Fbundle.atb",
    );
  });

  it("builds single-bundle view links without session_id", () => {
    expect(bundleViewHref(bundle, false)).toBe(
      "/view/?bundle_path=%2Ftmp%2Fsessions%2Fsess_abc123%2Fbundle.atb",
    );
  });

  it("prefers session_id for display labels", () => {
    expect(bundleDisplayId(bundle)).toBe("sess_abc123");
  });
});
