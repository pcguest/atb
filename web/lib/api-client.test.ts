import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type PropsWithChildren } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  fingerprintSessionToken,
  getQueryScope,
  queryKeys,
  type QueryScope,
  useVerificationQuery,
} from "@/lib/api-client";

const verificationPayload = {
  status: "valid",
  message: "verified",
  bundle_path: "/bundles/example.atb",
  chain_length: 3,
};

function jsonResponse(payload: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => payload,
  } as Response;
}

function requestToken(fetchMock: ReturnType<typeof vi.fn>, callIndex: number): string | null {
  const init = fetchMock.mock.calls[callIndex]?.[1] as RequestInit | undefined;
  return new Headers(init?.headers).get("X-ATB-Session-Token");
}

function testClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { retryDelay: 0 },
    },
  });
}

function wrapperFor(client: QueryClient) {
  return function QueryWrapper({ children }: PropsWithChildren) {
    return createElement(QueryClientProvider, { client }, children);
  };
}

describe("React Query cache scoping", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    window.history.replaceState({}, "", "/view/");
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("fingerprints the session token without exposing it", () => {
    const token = "viewer-secret-token";
    const fingerprint = fingerprintSessionToken(token);

    expect(fingerprint).toMatch(/^[0-9a-f]{16}$/);
    expect(fingerprint).not.toContain(token);
    expect(fingerprintSessionToken(token)).toBe(fingerprint);
    expect(fingerprintSessionToken("another-token")).not.toBe(fingerprint);
  });

  it("derives bundle identity from the viewer route", () => {
    window.history.replaceState(
      {},
      "",
      "/view/?session_id=session-1&bundle_path=%2Fbundles%2Fone.atb#session=token-1",
    );

    expect(getQueryScope()).toEqual([
      "session:session-1|path:/bundles/one.atb",
      fingerprintSessionToken("token-1"),
    ]);
  });

  it("produces distinct cache keys when the bundle or session changes", () => {
    const firstScope: QueryScope = ["path:/bundles/one.atb", fingerprintSessionToken("token-1")];
    const nextBundleScope: QueryScope = [
      "path:/bundles/two.atb",
      fingerprintSessionToken("token-1"),
    ];
    const nextSessionScope: QueryScope = [
      "path:/bundles/one.atb",
      fingerprintSessionToken("token-2"),
    ];

    const firstKey = queryKeys.bundleMeta(firstScope);
    expect(queryKeys.bundleMeta(nextBundleScope)).not.toEqual(firstKey);
    expect(queryKeys.bundleMeta(nextSessionScope)).not.toEqual(firstKey);
    expect(queryKeys.bundleEvents(nextSessionScope)).not.toEqual(
      queryKeys.bundleEvents(firstScope),
    );
  });

  it("keeps a stale query retry bound to the token that owns its cache key", async () => {
    window.history.replaceState({}, "", "/view/#session=token-one");
    const fetchMock = vi
      .fn()
      .mockRejectedValueOnce(new Error("temporary failure"))
      .mockResolvedValueOnce(jsonResponse(verificationPayload));
    vi.stubGlobal("fetch", fetchMock);

    const client = testClient();
    const { result } = renderHook(() => useVerificationQuery(), {
      wrapper: wrapperFor(client),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(requestToken(fetchMock, 0)).toBe("token-one");
    expect(requestToken(fetchMock, 1)).toBe("token-one");
    expect(
      JSON.stringify(
        client
          .getQueryCache()
          .getAll()
          .map((query) => query.queryKey),
      ),
    ).not.toContain("token-one");
  });

  it("does not let a failed old-scope request overwrite data after a token switch", async () => {
    window.history.replaceState({}, "", "/view/#session=token-one");
    let rejectOldRequest: ((reason?: unknown) => void) | undefined;
    const fetchMock = vi.fn((_path: string, init?: RequestInit) => {
      const token = new Headers(init?.headers).get("X-ATB-Session-Token");
      if (token === "token-one") {
        return new Promise<Response>((_resolve, reject) => {
          rejectOldRequest = reject;
        });
      }
      return Promise.resolve(jsonResponse({ ...verificationPayload, message: "token-two data" }));
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = testClient();
    const { result } = renderHook(() => useVerificationQuery(), {
      wrapper: wrapperFor(client),
    });
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));

    act(() => {
      window.history.replaceState({}, "", "/view/#session=token-two");
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });

    await waitFor(() => expect(result.current.data?.message).toBe("token-two data"));
    await act(async () => {
      rejectOldRequest?.(new Error("old request failed"));
      await Promise.resolve();
    });

    expect(result.current.data?.message).toBe("token-two data");
    expect(requestToken(fetchMock, 0)).toBe("token-one");
    expect(requestToken(fetchMock, 1)).toBe("token-two");
    await waitFor(() => {
      expect(
        client.getQueryData(
          queryKeys.verification(["/view/:current", fingerprintSessionToken("token-one")]),
        ),
      ).toBeUndefined();
    });
  });

  it("removes authenticated cache data and makes an anonymous request on logout", async () => {
    window.history.replaceState({}, "", "/view/#session=token-one");
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(verificationPayload))
      .mockResolvedValueOnce(jsonResponse({ ...verificationPayload, message: "anonymous data" }));
    vi.stubGlobal("fetch", fetchMock);

    const client = testClient();
    const authenticatedScope = getQueryScope();
    const { result } = renderHook(() => useVerificationQuery(), {
      wrapper: wrapperFor(client),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(client.getQueryData(queryKeys.verification(authenticatedScope))).toBeDefined();

    act(() => {
      window.history.replaceState({}, "", "/view/");
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });

    await waitFor(() => expect(result.current.data?.message).toBe("anonymous data"));
    await waitFor(() => {
      expect(client.getQueryData(queryKeys.verification(authenticatedScope))).toBeUndefined();
    });
    expect(requestToken(fetchMock, 0)).toBe("token-one");
    expect(requestToken(fetchMock, 1)).toBeNull();
  });
});
