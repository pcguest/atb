/**
 * Investigation UI location state.
 *
 * These helpers manage *presentation* location only — which investigation
 * surface is open, which record sequence is selected, and where a selection was
 * opened from. They are deliberately separate from the canonical evidence
 * identity (`?focus=<atb-locator>`), the evidence hashes, and the viewer
 * session token (which lives in the URL fragment and must never be moved into
 * the query string).
 *
 * URL slugs are human-legible (`run`, `status`) and mapped to the internal
 * surface ids (`incident`, `trust`) so the wire format does not leak internal
 * naming.
 */

export const SURFACE_IDS = [
  "incident",
  "findings",
  "timeline",
  "context",
  "relationships",
  "evidence",
  "trust",
] as const;

export type SurfaceId = (typeof SURFACE_IDS)[number];

const SLUG_BY_SURFACE: Record<SurfaceId, string> = {
  incident: "run",
  findings: "findings",
  timeline: "timeline",
  context: "context",
  relationships: "relationships",
  evidence: "evidence",
  trust: "status",
};

const SURFACE_BY_SLUG: Record<string, SurfaceId> = Object.fromEntries(
  (Object.entries(SLUG_BY_SURFACE) as Array<[SurfaceId, string]>).map(([surface, slug]) => [slug, surface]),
);

export function isSurfaceId(value: string | null | undefined): value is SurfaceId {
  return typeof value === "string" && (SURFACE_IDS as readonly string[]).includes(value);
}

export type ViewerLocation = {
  surface: SurfaceId | null;
  seq: number | null;
  /** Surface a selected record was opened from, for a stable return path. */
  from: SurfaceId | null;
  /** A `surface` value that was present but not a recognised slug. */
  unrecognisedSurface: string | null;
};

/**
 * Parse the presentation location from a query string. Unknown surfaces and
 * invalid sequences fail safe to null so callers can fall back to a bounded
 * default rather than rendering an inconsistent state.
 *
 * A sequence is only accepted alongside a valid surface, so an invalid surface
 * cannot leave an invisible selection armed. An unrecognised surface is
 * reported so the caller can be honest about a bad link rather than silently
 * pretending the link was absent.
 */
export function parseViewerLocation(search: string): ViewerLocation {
  const params = new URLSearchParams(search);
  const slug = params.get("surface");
  const surface = slug ? SURFACE_BY_SLUG[slug] ?? null : null;
  const unrecognisedSurface = slug && !surface ? slug : null;
  const fromSlug = params.get("from");
  const from = fromSlug ? SURFACE_BY_SLUG[fromSlug] ?? null : null;
  const rawSeq = params.get("seq");
  let seq: number | null = null;
  if (surface && rawSeq !== null && /^\d+$/.test(rawSeq)) {
    const parsed = Number.parseInt(rawSeq, 10);
    if (Number.isSafeInteger(parsed) && parsed >= 0) seq = parsed;
  }
  return { surface, seq, from, unrecognisedSurface };
}

/**
 * Build a same-origin URL that carries presentation location while preserving
 * the existing fragment (session token), dropping the now-resolved `focus`
 * reference, and recording the origin surface for a stable return path.
 */
export function viewerLocationUrl(
  href: string,
  surface: SurfaceId,
  seq: number | null,
  from: SurfaceId | null = null,
): string {
  const url = new URL(href);
  url.searchParams.set("surface", SLUG_BY_SURFACE[surface]);
  if (seq === null) url.searchParams.delete("seq");
  else url.searchParams.set("seq", String(seq));
  if (from && from !== surface) url.searchParams.set("from", SLUG_BY_SURFACE[from]);
  else url.searchParams.delete("from");
  url.searchParams.delete("focus");
  return `${url.pathname}${url.search}${url.hash}`;
}
