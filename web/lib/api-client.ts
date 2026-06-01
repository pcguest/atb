import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";
import { type ZodSchema } from "zod";

import { dashboardRoles, type DashboardRole } from "@/lib/roles";
import {
  apiErrorSchema,
  bundleEventsResponseSchema,
  bundleGraphResponseSchema,
  bundleMetaResponseSchema,
  privacyRevealRequestSchema,
  privacyRevealResponseSchema,
  profileReportSummarySchema,
  sessionsResponseSchema,
  verificationResponseSchema,
  workspaceBundlesResponseSchema,
} from "@/lib/schemas";
import type {
  BundleEventsResponse,
  BundleGraphResponse,
  BundleMetaResponse,
  EventRecord,
  PrivacyRevealRequest,
  PrivacyRevealResponse,
  ProfileReportSummary,
  SessionEntry,
  VerificationResponse,
  WorkspaceBundlesResponse,
} from "@/lib/types";

const defaultEventsPageSize = 200;
const dashboardRefetchIntervalMs = 10000;
const liveEventsRefetchIntervalMs = 5000;

// Session token is delivered in the URL fragment (#session=<token>) by the Go server
// so it is never sent automatically by the browser as part of HTTP requests.
// Re-read the hash on each request so client navigations to a new viewer session work.
function getSessionToken(): string {
  if (typeof window === "undefined") {
    return "";
  }
  const params = new URLSearchParams(window.location.hash.slice(1));
  return params.get("session") ?? "";
}

export const queryKeys = {
  verification: ["atb", "verification"] as const,
  bundleMeta: ["atb", "bundle", "meta"] as const,
  bundleGraph: ["atb", "bundle", "graph"] as const,
  bundleEvents: ["atb", "bundle", "events"] as const,
  bundleProfile: ["atb", "bundle", "profile"] as const,
  workspaceBundles: ["atb", "workspace", "bundles"] as const,
  sessions: ["atb", "sessions"] as const,
};

function parseWithSchema<T>(schema: ZodSchema<T>, payload: unknown, path: string): T {
  const parsed = schema.safeParse(payload);
  if (!parsed.success) {
    throw new Error(`Invalid API response from ${path}`);
  }
  return parsed.data;
}

async function requestJSON<T>(path: string, schema: ZodSchema<T>, init?: RequestInit): Promise<T> {
  const sessionToken = getSessionToken();
  const sessionHeader: Record<string, string> = sessionToken
    ? { "X-ATB-Session-Token": sessionToken }
    : {};
  const response = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...sessionHeader,
      ...(init?.headers ?? {}),
    },
    cache: "no-store",
  });

  const rawPayload = await response
    .json()
    .catch((): unknown => ({ error: `Unexpected response from ${path}` }));

  if (!response.ok) {
    const parsedError = apiErrorSchema.safeParse(rawPayload);
    if (parsedError.success) {
      throw new Error(parsedError.data.error);
    }
    throw new Error(`Request failed (${response.status})`);
  }

  return parseWithSchema(schema, rawPayload, path);
}

export function getVerification(): Promise<VerificationResponse> {
  return requestJSON("/api/v1/verification", verificationResponseSchema);
}

export function getBundleMeta(): Promise<BundleMetaResponse> {
  return requestJSON("/api/v1/bundle/meta", bundleMetaResponseSchema);
}

export function getBundleEvents(
  offset = 0,
  limit = defaultEventsPageSize,
): Promise<BundleEventsResponse> {
  const params = new URLSearchParams({
    offset: String(offset),
    limit: String(limit),
  });
  return requestJSON(`/api/v1/bundle/events?${params.toString()}`, bundleEventsResponseSchema);
}

export function getBundleGraph(): Promise<BundleGraphResponse> {
  return requestJSON("/api/v1/bundle/graph", bundleGraphResponseSchema);
}

export function listWorkspaceBundles(): Promise<WorkspaceBundlesResponse> {
  return requestJSON("/v1/workspace/bundles", workspaceBundlesResponseSchema);
}

export async function getSessions(): Promise<SessionEntry[]> {
  const response = await requestJSON("/api/v1/sessions", sessionsResponseSchema);
  return response.sessions;
}

export async function getBundleProfile(): Promise<ProfileReportSummary | null> {
  const sessionToken = getSessionToken();
  const sessionHeader: Record<string, string> = sessionToken
    ? { "X-ATB-Session-Token": sessionToken }
    : {};
  const response = await fetch("/api/v1/bundle/profile", {
    cache: "no-store",
    headers: sessionHeader,
  });
  if (response.status === 204) {
    return null;
  }
  if (response.status === 403) {
    const err = new Error("FORBIDDEN") as Error & { status: number };
    err.status = 403;
    throw err;
  }
  if (!response.ok) {
    const raw = await response.json().catch((): unknown => ({}));
    const parsed = apiErrorSchema.safeParse(raw);
    throw new Error(parsed.success ? parsed.data.error : `Request failed (${response.status})`);
  }
  const raw = await response.json();
  return parseWithSchema(profileReportSummarySchema, raw, "/api/v1/bundle/profile");
}

export function runBundleVerify(): Promise<ProfileReportSummary> {
  return requestJSON("/api/v1/bundle/verify", profileReportSummarySchema, { method: "POST" });
}

export function useBundleProfileQuery(
  enabled: boolean,
): UseQueryResult<ProfileReportSummary | null, Error> {
  return useQuery({
    queryKey: queryKeys.bundleProfile,
    queryFn: getBundleProfile,
    enabled,
    retry: (failureCount, error) =>
      (error as Error & { status?: number }).status !== 403 && failureCount < 1,
    staleTime: 15000,
  });
}

export function useRunBundleVerifyMutation(): UseMutationResult<ProfileReportSummary, Error, void> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => runBundleVerify(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.bundleProfile });
    },
  });
}

export function revealField(payload: PrivacyRevealRequest): Promise<PrivacyRevealResponse> {
  const request = parseWithSchema(
    privacyRevealRequestSchema,
    payload,
    "/api/v1/privacy/reveal payload",
  );
  return requestJSON("/api/v1/privacy/reveal", privacyRevealResponseSchema, {
    method: "POST",
    body: JSON.stringify(request),
  });
}

export function useVerificationQuery(): UseQueryResult<VerificationResponse, Error> {
  return useQuery({
    queryKey: queryKeys.verification,
    queryFn: getVerification,
    refetchInterval: dashboardRefetchIntervalMs,
    retry: 1,
    staleTime: 5000,
  });
}

export function useBundleMetaQuery(enabled: boolean): UseQueryResult<BundleMetaResponse, Error> {
  return useQuery({
    queryKey: queryKeys.bundleMeta,
    queryFn: getBundleMeta,
    enabled,
    refetchInterval: enabled ? dashboardRefetchIntervalMs : false,
    retry: 1,
  });
}

export function useBundleGraphQuery(enabled: boolean): UseQueryResult<BundleGraphResponse, Error> {
  return useQuery({
    queryKey: queryKeys.bundleGraph,
    queryFn: getBundleGraph,
    enabled,
    refetchInterval: enabled ? dashboardRefetchIntervalMs : false,
    retry: 1,
  });
}

export function useWorkspaceBundlesQuery(
  enabled: boolean,
): UseQueryResult<WorkspaceBundlesResponse, Error> {
  return useQuery({
    queryKey: queryKeys.workspaceBundles,
    queryFn: listWorkspaceBundles,
    enabled,
    retry: false,
    staleTime: 30000,
  });
}

export function useSessionsQuery(enabled: boolean): UseQueryResult<SessionEntry[], Error> {
  return useQuery({
    queryKey: queryKeys.sessions,
    queryFn: getSessions,
    enabled,
    retry: false,
    staleTime: 30000,
  });
}

export function useBundleEventsQuery(enabled: boolean, pageSize = defaultEventsPageSize) {
  return useInfiniteQuery({
    queryKey: [...queryKeys.bundleEvents, pageSize],
    queryFn: ({ pageParam }) => getBundleEvents(Number(pageParam), pageSize),
    initialPageParam: 0,
    getNextPageParam: (lastPage, allPages) => {
      const loadedCount = allPages.reduce((count, page) => count + page.events.length, 0);
      return loadedCount < lastPage.total ? loadedCount : undefined;
    },
    enabled,
    retry: 1,
    refetchInterval: enabled ? liveEventsRefetchIntervalMs : false,
    structuralSharing: true,
  });
}

export function useRevealFieldMutation(): UseMutationResult<
  PrivacyRevealResponse,
  Error,
  PrivacyRevealRequest
> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: revealField,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.bundleEvents });
    },
  });
}

export function flattenEventPages(pages: BundleEventsResponse[] | undefined): EventRecord[] {
  if (!pages) {
    return [];
  }

  const eventsBySeq = new Map<number, EventRecord>();
  for (const page of pages) {
    for (const event of page.events) {
      if (!eventsBySeq.has(event.seq)) {
        eventsBySeq.set(event.seq, event);
      }
    }
  }

  return Array.from(eventsBySeq.values()).sort((a, b) => a.seq - b.seq);
}

export function isDashboardRole(value: string): value is DashboardRole {
  return dashboardRoles.includes(value as DashboardRole);
}
