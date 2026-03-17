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
  verificationResponseSchema,
} from "@/lib/schemas";
import type {
  BundleEventsResponse,
  BundleGraphResponse,
  BundleMetaResponse,
  EventRecord,
  PrivacyRevealRequest,
  PrivacyRevealResponse,
  VerificationResponse,
} from "@/lib/types";

const defaultEventsPageSize = 200;
const dashboardRefetchIntervalMs = 10000;
const liveEventsRefetchIntervalMs = 5000;

export const queryKeys = {
  verification: ["atb", "verification"] as const,
  bundleMeta: ["atb", "bundle", "meta"] as const,
  bundleGraph: ["atb", "bundle", "graph"] as const,
  bundleEvents: ["atb", "bundle", "events"] as const,
};

function parseWithSchema<T>(schema: ZodSchema<T>, payload: unknown, path: string): T {
  const parsed = schema.safeParse(payload);
  if (!parsed.success) {
    throw new Error(`Invalid API response from ${path}`);
  }
  return parsed.data;
}

async function requestJSON<T>(path: string, schema: ZodSchema<T>, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
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
