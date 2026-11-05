import {
  parseViewerLocation,
  surfaceSlug,
  type SurfaceId,
} from "@/lib/viewer-location";

export const LOCAL_VIEWER_ROUTES = ["/workspace", "/sessions", "/view"] as const;
export type LocalViewerRoute = (typeof LOCAL_VIEWER_ROUTES)[number];

export type LocalViewerField =
  | "bundle_path"
  | "session_id"
  | "surface"
  | "seq"
  | "from"
  | "focus";

export type LocalViewerContext = {
  route: LocalViewerRoute | null;
  bundlePath: string | null;
  sessionId: string | null;
  surface: SurfaceId | null;
  seq: number | null;
  from: SurfaceId | null;
  focus: string | null;
  fragment: string;
  discarded: LocalViewerField[];
};

export type LocalViewerNavigation = {
  href: string;
  discarded: LocalViewerField[];
};

const ROUTE_FIELDS: Record<LocalViewerRoute, readonly LocalViewerField[]> = {
  "/workspace": ["bundle_path", "session_id"],
  "/sessions": ["bundle_path", "session_id"],
  "/view": ["bundle_path", "session_id", "surface", "seq", "from", "focus"],
};

const FIELD_ORDER: readonly LocalViewerField[] = [
  "bundle_path",
  "session_id",
  "surface",
  "seq",
  "from",
  "focus",
];

const routeBase = "http://atb.local";

function routeFromPath(pathname: string): LocalViewerRoute | null {
  const normalized = pathname.length > 1 ? pathname.replace(/\/+$/, "") : pathname;
  return (LOCAL_VIEWER_ROUTES as readonly string[]).includes(normalized)
    ? (normalized as LocalViewerRoute)
    : null;
}

function safeValue(raw: string | null, maxLength: number): { value: string | null; invalid: boolean } {
  if (raw === null) return { value: null, invalid: false };
  if (raw.length === 0 || raw.length > maxLength || /[\u0000-\u001f\u007f]/.test(raw)) {
    return { value: null, invalid: true };
  }
  return { value: raw, invalid: false };
}

function addDiscarded(discarded: LocalViewerField[], field: LocalViewerField) {
  if (!discarded.includes(field)) discarded.push(field);
}

export function parseLocalViewerContext(href: string): LocalViewerContext {
  const url = new URL(href, routeBase);
  const params = url.searchParams;
  const discarded: LocalViewerField[] = [];

  const rawBundlePath = params.has("bundle_path")
    ? params.get("bundle_path")
    : params.get("bundlePath");
  const bundlePath = safeValue(rawBundlePath, 4096);
  if (bundlePath.invalid) addDiscarded(discarded, "bundle_path");

  const sessionId = safeValue(params.get("session_id"), 512);
  if (sessionId.invalid) addDiscarded(discarded, "session_id");

  const focus = safeValue(params.get("focus"), 8192);
  if (focus.invalid) addDiscarded(discarded, "focus");

  const location = parseViewerLocation(url.search);
  if (params.has("surface") && !location.surface) addDiscarded(discarded, "surface");
  if (params.has("seq") && location.seq === null) addDiscarded(discarded, "seq");
  if (params.has("from") && !location.from) addDiscarded(discarded, "from");

  return {
    route: routeFromPath(url.pathname),
    bundlePath: bundlePath.value,
    sessionId: sessionId.value,
    surface: location.surface,
    seq: location.seq,
    from: location.from,
    focus: focus.value,
    fragment: url.hash,
    discarded,
  };
}

function contextValue(context: LocalViewerContext, field: LocalViewerField): string | null {
  switch (field) {
    case "bundle_path":
      return context.bundlePath;
    case "session_id":
      return context.sessionId;
    case "surface":
      return context.surface === null ? null : surfaceSlug(context.surface);
    case "seq":
      return context.seq === null ? null : String(context.seq);
    case "from":
      return context.from === null ? null : surfaceSlug(context.from);
    case "focus":
      return context.focus;
  }
}

function mergeContext(
  context: LocalViewerContext,
  overrides?: Partial<Pick<LocalViewerContext, "bundlePath" | "sessionId" | "surface" | "seq" | "from" | "focus">>,
): LocalViewerContext {
  if (!overrides) return context;
  return {
    ...context,
    bundlePath: overrides.bundlePath !== undefined ? overrides.bundlePath : context.bundlePath,
    sessionId: overrides.sessionId !== undefined ? overrides.sessionId : context.sessionId,
    surface: overrides.surface !== undefined ? overrides.surface : context.surface,
    seq: overrides.seq !== undefined ? overrides.seq : context.seq,
    from: overrides.from !== undefined ? overrides.from : context.from,
    focus: overrides.focus !== undefined ? overrides.focus : context.focus,
  };
}

export function localViewerNavigation(
  target: LocalViewerRoute,
  context: LocalViewerContext,
  overrides?: Partial<Pick<LocalViewerContext, "bundlePath" | "sessionId" | "surface" | "seq" | "from" | "focus">>,
): LocalViewerNavigation {
  const merged = mergeContext(context, overrides);
  const allowed = ROUTE_FIELDS[target];
  const discarded: LocalViewerField[] = [...context.discarded];
  for (const field of FIELD_ORDER) {
    if (!allowed.includes(field) && contextValue(merged, field) !== null) addDiscarded(discarded, field);
  }

  const url = new URL(`${target}/`, routeBase);
  for (const field of FIELD_ORDER) {
    if (!allowed.includes(field)) continue;
    let value = contextValue(merged, field);
    if (value === null) continue;
    if (field === "bundle_path" || field === "session_id" || field === "focus") {
      const max = field === "bundle_path" ? 4096 : field === "session_id" ? 512 : 8192;
      const safe = safeValue(value, max);
      if (safe.invalid) {
        addDiscarded(discarded, field);
        continue;
      }
      value = safe.value;
    }
    if (field === "seq" && merged.surface === null) {
      addDiscarded(discarded, "seq");
      continue;
    }
    if (value !== null) url.searchParams.set(field, value);
  }
  url.hash = merged.fragment;
  return { href: `${url.pathname}${url.search}${url.hash}`, discarded };
}

export function localViewerHref(
  target: LocalViewerRoute,
  context: LocalViewerContext,
  overrides?: Partial<Pick<LocalViewerContext, "bundlePath" | "sessionId" | "surface" | "seq" | "from" | "focus">>,
): string {
  return localViewerNavigation(target, context, overrides).href;
}
