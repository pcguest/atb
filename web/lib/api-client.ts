import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type QueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";
import { useEffect, useMemo, useSyncExternalStore } from "react";
import { type ZodSchema } from "zod";

import { dashboardRoles, type DashboardRole } from "@/lib/roles";
import {
  actorSessionsResponseSchema,
  apiErrorSchema,
  bundleEventsResponseSchema,
  bundleGraphResponseSchema,
  bundleMetaResponseSchema,
  privacyRevealRequestSchema,
  privacyRevealResponseSchema,
  profileReportSummarySchema,
  schemaStatusResponseSchema,
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
  SchemaStatusResponse,
  SessionEntry,
  VerificationResponse,
  WorkspaceBundlesResponse,
} from "@/lib/types";

const defaultEventsPageSize = 200;
const dashboardRefetchIntervalMs = 10000;
const liveEventsRefetchIntervalMs = 5000;

// Session token is delivered in the URL fragment (#session=<token>) by the Go server
// so it is never sent automatically by the browser as part of HTTP requests.
// Hooks capture it with the cache scope so retries cannot adopt a newer credential.
function getSessionToken(): string {
  if (typeof window === "undefined") {
    return "";
  }
  const params = new URLSearchParams(window.location.hash.slice(1));
  return params.get("session") ?? "";
}

export type QueryScope = readonly [bundleIdentity: string, sessionTokenFingerprint: string];

type RequestContext = {
  scope: QueryScope;
  sessionToken: string;
};

// This fingerprint only namespaces the in-memory query cache. It deliberately
// avoids putting the bearer token itself in React Query keys and devtools.
export function fingerprintSessionToken(token: string): string {
  if (!token) {
    return "anonymous";
  }

  let first = 0x811c9dc5;
  let second = 0x9e3779b9;
  for (let index = 0; index < token.length; index += 1) {
    const code = token.charCodeAt(index);
    first = Math.imul(first ^ code, 0x01000193);
    second = Math.imul(second ^ code, 0x85ebca6b);
  }
  return `${(first >>> 0).toString(16).padStart(8, "0")}${(second >>> 0)
    .toString(16)
    .padStart(8, "0")}`;
}

export function getQueryScope(): QueryScope {
  if (typeof window === "undefined") {
    return ["server", "anonymous"];
  }

  const params = new URLSearchParams(window.location.search);
  const sessionID = params.get("session_id");
  const bundlePath = params.get("bundle_path") ?? params.get("bundlePath");
  const identityParts = [
    sessionID ? `session:${sessionID}` : "",
    bundlePath ? `path:${bundlePath}` : "",
  ].filter(Boolean);
  const bundleIdentity =
    identityParts.length > 0 ? identityParts.join("|") : `${window.location.pathname}:current`;

  return [bundleIdentity, fingerprintSessionToken(getSessionToken())];
}

const serverRequestContextSnapshot = JSON.stringify({
  scope: ["server", "anonymous"],
  sessionToken: "",
} satisfies RequestContext);
const previousScopeByClient = new WeakMap<QueryClient, QueryScope>();

function getRequestContextSnapshot(): string {
  return JSON.stringify({
    scope: getQueryScope(),
    sessionToken: getSessionToken(),
  } satisfies RequestContext);
}

function subscribeToScopeChange(onStoreChange: () => void): () => void {
  window.addEventListener("hashchange", onStoreChange);
  window.addEventListener("popstate", onStoreChange);
  return () => {
    window.removeEventListener("hashchange", onStoreChange);
    window.removeEventListener("popstate", onStoreChange);
  };
}

function useRequestContext(): RequestContext {
  const queryClient = useQueryClient();
  const snapshot = useSyncExternalStore(
    subscribeToScopeChange,
    getRequestContextSnapshot,
    () => serverRequestContextSnapshot,
  );
  const context = useMemo(() => JSON.parse(snapshot) as RequestContext, [snapshot]);
  const { scope } = context;

  useEffect(() => {
    const previousScope = previousScopeByClient.get(queryClient);
    previousScopeByClient.set(queryClient, scope);
    if (!previousScope || (previousScope[0] === scope[0] && previousScope[1] === scope[1])) {
      return;
    }

    const previousPrefix = ["atb", ...previousScope] as const;
    void queryClient.cancelQueries({ queryKey: previousPrefix });
    queryClient.removeQueries({ queryKey: previousPrefix });
  }, [queryClient, scope]);

  return context;
}

function scopedQueryKey(scope: QueryScope, resource: readonly string[]) {
  return ["atb", ...scope, ...resource] as const;
}

export const queryKeys = {
  verification: (scope: QueryScope) => scopedQueryKey(scope, ["verification"]),
  bundleMeta: (scope: QueryScope) => scopedQueryKey(scope, ["bundle", "meta"]),
  bundleGraph: (scope: QueryScope) => scopedQueryKey(scope, ["bundle", "graph"]),
  bundleEvents: (scope: QueryScope) => scopedQueryKey(scope, ["bundle", "events"]),
  bundleProfile: (scope: QueryScope) => scopedQueryKey(scope, ["bundle", "profile"]),
  workspaceBundles: (scope: QueryScope) => scopedQueryKey(scope, ["workspace", "bundles"]),
  sessions: (scope: QueryScope) => scopedQueryKey(scope, ["sessions"]),
  actorSessions: (scope: QueryScope) => scopedQueryKey(scope, ["sessions", "by-actor"]),
  schemaStatus: (scope: QueryScope) => scopedQueryKey(scope, ["schema", "status"]),
};

function parseWithSchema<T>(schema: ZodSchema<T>, payload: unknown, path: string): T {
  const parsed = schema.safeParse(payload);
  if (!parsed.success) {
    throw new Error(`Invalid API response from ${path}`);
  }
  return parsed.data;
}

async function requestJSON<T>(
  path: string,
  schema: ZodSchema<T>,
  init?: RequestInit,
  sessionToken = getSessionToken(),
): Promise<T> {
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

export function getVerification(
  sessionToken = getSessionToken(),
  signal?: AbortSignal,
): Promise<VerificationResponse> {
  return requestJSON("/api/v1/verification", verificationResponseSchema, { signal }, sessionToken);
}

export function getBundleMeta(
  sessionToken = getSessionToken(),
  signal?: AbortSignal,
): Promise<BundleMetaResponse> {
  return requestJSON("/api/v1/bundle/meta", bundleMetaResponseSchema, { signal }, sessionToken);
}

export function getBundleEvents(
  offset = 0,
  limit = defaultEventsPageSize,
  sessionToken = getSessionToken(),
  signal?: AbortSignal,
): Promise<BundleEventsResponse> {
  const params = new URLSearchParams({
    offset: String(offset),
    limit: String(limit),
  });
  return requestJSON(
    `/api/v1/bundle/events?${params.toString()}`,
    bundleEventsResponseSchema,
    { signal },
    sessionToken,
  );
}

export function getBundleGraph(
  sessionToken = getSessionToken(),
  signal?: AbortSignal,
): Promise<BundleGraphResponse> {
  return requestJSON("/api/v1/bundle/graph", bundleGraphResponseSchema, { signal }, sessionToken);
}

export function listWorkspaceBundles(
  sessionToken = getSessionToken(),
  signal?: AbortSignal,
): Promise<WorkspaceBundlesResponse> {
  return requestJSON(
    "/v1/workspace/bundles",
    workspaceBundlesResponseSchema,
    { signal },
    sessionToken,
  );
}

export async function getSessions(
  sessionToken = getSessionToken(),
  signal?: AbortSignal,
): Promise<SessionEntry[]> {
  const response = await requestJSON(
    "/api/v1/sessions",
    sessionsResponseSchema,
    { signal },
    sessionToken,
  );
  return response.sessions;
}

export async function getActorSessions(
  sessionToken = getSessionToken(),
  signal?: AbortSignal,
): Promise<Record<string, SessionEntry[]>> {
  const response = await requestJSON(
    "/api/v1/sessions/by-actor",
    actorSessionsResponseSchema,
    { signal },
    sessionToken,
  );
  return response.actors;
}

export function getSchemaStatus(
  sessionToken = getSessionToken(),
  signal?: AbortSignal,
): Promise<SchemaStatusResponse> {
  return requestJSON("/api/v1/schema/status", schemaStatusResponseSchema, { signal }, sessionToken);
}

export async function getBundleProfile(
  sessionToken = getSessionToken(),
  signal?: AbortSignal,
): Promise<ProfileReportSummary | null> {
  const sessionHeader: Record<string, string> = sessionToken
    ? { "X-ATB-Session-Token": sessionToken }
    : {};
  const response = await fetch("/api/v1/bundle/profile", {
    cache: "no-store",
    headers: sessionHeader,
    signal,
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

export function runBundleVerify(sessionToken = getSessionToken()): Promise<ProfileReportSummary> {
  return requestJSON(
    "/api/v1/bundle/verify",
    profileReportSummarySchema,
    { method: "POST" },
    sessionToken,
  );
}

export function useBundleProfileQuery(
  enabled: boolean,
): UseQueryResult<ProfileReportSummary | null, Error> {
  const { scope, sessionToken } = useRequestContext();
  return useQuery({
    queryKey: queryKeys.bundleProfile(scope),
    queryFn: ({ signal }) => getBundleProfile(sessionToken, signal),
    enabled,
    retry: (failureCount, error) =>
      (error as Error & { status?: number }).status !== 403 && failureCount < 1,
    staleTime: 15000,
  });
}

export function useRunBundleVerifyMutation(): UseMutationResult<ProfileReportSummary, Error, void> {
  const queryClient = useQueryClient();
  const { scope, sessionToken } = useRequestContext();
  return useMutation({
    mutationKey: [...queryKeys.bundleProfile(scope), "verify"],
    mutationFn: () => runBundleVerify(sessionToken),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.bundleProfile(scope) });
    },
  });
}

export function revealField(
  payload: PrivacyRevealRequest,
  sessionToken = getSessionToken(),
): Promise<PrivacyRevealResponse> {
  const request = parseWithSchema(
    privacyRevealRequestSchema,
    payload,
    "/api/v1/privacy/reveal payload",
  );
  return requestJSON(
    "/api/v1/privacy/reveal",
    privacyRevealResponseSchema,
    {
      method: "POST",
      body: JSON.stringify(request),
    },
    sessionToken,
  );
}

export function useVerificationQuery(): UseQueryResult<VerificationResponse, Error> {
  const { scope, sessionToken } = useRequestContext();
  return useQuery({
    queryKey: queryKeys.verification(scope),
    queryFn: ({ signal }) => getVerification(sessionToken, signal),
    refetchInterval: dashboardRefetchIntervalMs,
    retry: 1,
    staleTime: 5000,
  });
}

export function useBundleMetaQuery(enabled: boolean): UseQueryResult<BundleMetaResponse, Error> {
  const { scope, sessionToken } = useRequestContext();
  return useQuery({
    queryKey: queryKeys.bundleMeta(scope),
    queryFn: ({ signal }) => getBundleMeta(sessionToken, signal),
    enabled,
    refetchInterval: enabled ? dashboardRefetchIntervalMs : false,
    retry: 1,
  });
}

export function useBundleGraphQuery(enabled: boolean): UseQueryResult<BundleGraphResponse, Error> {
  const { scope, sessionToken } = useRequestContext();
  return useQuery({
    queryKey: queryKeys.bundleGraph(scope),
    queryFn: ({ signal }) => getBundleGraph(sessionToken, signal),
    enabled,
    refetchInterval: enabled ? dashboardRefetchIntervalMs : false,
    retry: 1,
  });
}

export function useWorkspaceBundlesQuery(
  enabled: boolean,
): UseQueryResult<WorkspaceBundlesResponse, Error> {
  const { scope, sessionToken } = useRequestContext();
  return useQuery({
    queryKey: queryKeys.workspaceBundles(scope),
    queryFn: ({ signal }) => listWorkspaceBundles(sessionToken, signal),
    enabled,
    retry: false,
    staleTime: 30000,
  });
}

export function useSessionsQuery(enabled: boolean): UseQueryResult<SessionEntry[], Error> {
  const { scope, sessionToken } = useRequestContext();
  return useQuery({
    queryKey: queryKeys.sessions(scope),
    queryFn: ({ signal }) => getSessions(sessionToken, signal),
    enabled,
    retry: false,
    staleTime: 30000,
  });
}

export function useActorSessionsQuery(
  enabled: boolean,
): UseQueryResult<Record<string, SessionEntry[]>, Error> {
  const { scope, sessionToken } = useRequestContext();
  return useQuery({
    queryKey: queryKeys.actorSessions(scope),
    queryFn: ({ signal }) => getActorSessions(sessionToken, signal),
    enabled,
    retry: false,
    staleTime: 30000,
  });
}

export function useSchemaStatusQuery(
  enabled: boolean,
): UseQueryResult<SchemaStatusResponse, Error> {
  const { scope, sessionToken } = useRequestContext();
  return useQuery({
    queryKey: queryKeys.schemaStatus(scope),
    queryFn: ({ signal }) => getSchemaStatus(sessionToken, signal),
    enabled,
    retry: false,
    staleTime: 30000,
  });
}

export function useBundleEventsQuery(enabled: boolean, pageSize = defaultEventsPageSize) {
  const { scope, sessionToken } = useRequestContext();
  return useInfiniteQuery({
    queryKey: [...queryKeys.bundleEvents(scope), pageSize],
    queryFn: ({ pageParam, signal }) =>
      getBundleEvents(Number(pageParam), pageSize, sessionToken, signal),
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
  const { scope, sessionToken } = useRequestContext();
  return useMutation({
    mutationKey: [...queryKeys.bundleEvents(scope), "reveal"],
    mutationFn: (payload) => revealField(payload, sessionToken),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.bundleEvents(scope) });
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
