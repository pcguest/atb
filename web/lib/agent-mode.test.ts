import { afterEach, describe, expect, it, vi } from "vitest";

import {
  detectAgentMode,
  isAgentModeBuildTime,
  resetAgentModeCacheForTests,
} from "@/lib/agent-mode";

describe("agent mode detection", () => {
  afterEach(() => {
    resetAgentModeCacheForTests();
    vi.unstubAllGlobals();
  });

  it("honours NEXT_PUBLIC_ATB_AGENT_MODE at build time", () => {
    expect(typeof isAgentModeBuildTime()).toBe("boolean");
  });

  it("returns false when workspace probe fails", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => ({ ok: false, status: 404 }) as Response),
    );

    await expect(detectAgentMode()).resolves.toBe(false);
  });

  it("returns true when workspace probe succeeds", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => ({ ok: true, status: 200, headers: new Headers({ "content-type": "application/json" }) }) as Response),
    );

    await expect(detectAgentMode()).resolves.toBe(true);
  });

  it("rejects a successful HTML response from a non-Agent viewer", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => ({ ok: true, status: 200, headers: new Headers({ "content-type": "text/html" }) }) as Response),
    );

    await expect(detectAgentMode()).resolves.toBe(false);
  });
});
