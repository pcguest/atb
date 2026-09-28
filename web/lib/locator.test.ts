import { afterEach, describe, expect, it, vi } from "vitest";

import { locateEvidence } from "@/lib/api-client";

const head = "a".repeat(64);

function jsonResponse(payload: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => payload,
  } as Response;
}

describe("locateEvidence", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sends the locator and returns an identity resolution", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) =>
      jsonResponse({
        ok: true,
        canonical: `atb://evidence/1/${head}?seq=2`,
        seq: 2,
        record_hash_matched: false,
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const locator = `atb://evidence/1/${head}?seq=2`;
    const result = await locateEvidence(locator, "tok");

    expect(result.ok).toBe(true);
    expect(result.seq).toBe(2);
    const called = fetchMock.mock.calls[0]?.[0] as string;
    expect(called).toContain("/api/v1/bundle/locate?");
    expect(new URL(called, "http://localhost").searchParams.get("locator")).toBe(locator);
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined;
    expect(new Headers(init?.headers).get("X-ATB-Session-Token")).toBe("tok");
  });

  it("surfaces a location failure code without throwing", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          ok: false,
          seq: 0,
          record_hash_matched: false,
          error_code: "EVENT_NOT_FOUND",
          message: "no such record",
        }),
      ),
    );

    const result = await locateEvidence(`atb://evidence/1/${head}?seq=9`, "tok");

    expect(result.ok).toBe(false);
    expect(result.error_code).toBe("EVENT_NOT_FOUND");
  });
});
