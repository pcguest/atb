"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";
import { CommandPalette, type PaletteAction } from "./components/CommandPalette";
import { RoleSelector } from "./components/role-selector/RoleSelector";
import { EmptyState, ErrorState, LoadingState, StatusBadge } from "./components/ui/investigation";
import { FindingsSurface, IncidentSurface, RecordSurface, actionClass, coverageLabel } from "./components/CoreSurfaces";
import { ContextSurface, RelationshipsSurface, TrustSurface } from "./components/AssuranceSurfaces";
import { CompactInvestigationNavigation, InvestigationNavigation, investigationSurfaces, surfaceLabel, type Surface } from "./components/InvestigationNavigation";
import { ObjectIdentityHeader } from "./components/ObjectIdentityHeader";
import { EventInspector } from "@/components/dashboard/EventInspector";
import { flattenEventPages, locateEvidence, useBundleEventsQuery, useInvestigationContextQuery, useInvestigationFindingsQuery, useInvestigationOverviewQuery, useInvestigationRelationshipsQuery, useInvestigationTimelineQuery, useInvestigationTrustQuery, useRevealFieldMutation, useRunBundleVerifyMutation } from "@/lib/api-client";
import { parseViewerLocation, viewerLocationUrl, type ViewerLocation } from "@/lib/viewer-location";

type QueryState = { isLoading?: boolean; isError?: boolean; data?: unknown; refetch?: () => unknown };

function QueryContent({ query, children }: { query: QueryState; children: ReactNode }) {
  if (query.isError) return <ErrorState onRetry={() => void query.refetch?.()}>The local evidence request failed. Try again while this viewer session is active.</ErrorState>;
  if (query.isLoading || !query.data) return <LoadingState/>;
  return children;
}

function readInitialLocation(): ViewerLocation {
  if (typeof window === "undefined") return { surface: null, seq: null, from: null, unrecognisedSurface: null };
  return parseViewerLocation(window.location.search);
}

export default function ViewPage() {
  // Initialise to the bounded default so the static-export HTML hydrates
  // deterministically; the URL is applied once, after mount, by the effect
  // below (reading window during render would cause a hydration mismatch).
  const [surface, setSurface] = useState<Surface>("incident");
  const [selectedSeq, setSelectedSeq] = useState<number | null>(null);
  const [selectedFinding, setSelectedFinding] = useState(0);
  const [source, setSource] = useState<Surface | null>(null);
  const [scopeGeneration, setScopeGeneration] = useState(0);
  const [focusNotice, setFocusNotice] = useState<{ code: string; message: string } | null>(null);
  const [locationNotice, setLocationNotice] = useState<string | null>(null);
  // Set once the user navigates, so a slower in-flight focus lookup cannot
  // yank the surface back to the referenced record.
  const navigatedRef = useRef(false);
  const overviewQuery = useInvestigationOverviewQuery();
  const trustQuery = useInvestigationTrustQuery();
  const valid = overviewQuery.data?.integrity_valid === true;
  const findingsQuery = useInvestigationFindingsQuery(valid);
  const timelineQuery = useInvestigationTimelineQuery(valid);
  const contextQuery = useInvestigationContextQuery(valid);
  const relationshipsQuery = useInvestigationRelationshipsQuery(valid);
  const eventsQuery = useBundleEventsQuery(valid && (surface === "evidence" || surface === "timeline"));
  const verifyMutation = useRunBundleVerifyMutation();
  const revealMutation = useRevealFieldMutation();
  const events = flattenEventPages(eventsQuery.data?.pages);
  const timeline = (timelineQuery.data?.events ?? []).filter(event => event.type !== "atb.bundle.manifest");
  const effectiveSeq = selectedSeq ?? timeline[0]?.seq ?? null;
  const selectedEvent = events.find(event => event.seq === effectiveSeq) ?? null;
  const findings = findingsQuery.data?.findings ?? [];
  const acquisitionFindings = findingsQuery.data?.acquisition_findings ?? [];

  // Presentation location is mirrored into the URL so Back/Forward and direct
  // deep links preserve investigation continuity. It never carries the session
  // token (which stays in the fragment) or revealed field values.
  function navigate(nextSurface: Surface, nextSeq: number | null, options?: { replace?: boolean; from?: Surface | null }) {
    const from = options?.from ?? null;
    navigatedRef.current = true;
    setSurface(nextSurface);
    setSelectedSeq(nextSeq);
    setSource(from);
    setLocationNotice(null);
    if (typeof window === "undefined") return;
    const url = viewerLocationUrl(window.location.href, nextSurface, nextSeq, from);
    if (options?.replace) window.history.replaceState(null, "", url);
    else window.history.pushState(null, "", url);
  }

  function selectSurface(id: Surface) { navigate(id, selectedSeq); }

  useEffect(() => {
    const apply = (markNavigated: boolean) => {
      const location = readInitialLocation();
      if (markNavigated) navigatedRef.current = true;
      setSurface(location.surface ?? "incident");
      setSelectedSeq(location.seq);
      setSource(location.from);
      setLocationNotice(
        location.unrecognisedSurface
          ? `The link named an investigation surface (“${location.unrecognisedSurface}”) that is not recognised. Showing Run.`
          : null,
      );
    };
    // Apply the URL once on mount (after hydration) without suppressing the
    // focus locator; later history changes are user-driven navigation.
    apply(false);
    const onPop = () => apply(true);
    // A session-token change swaps the whole data scope; reset selection rather
    // than render stale records against a new scope.
    const onHash = () => { setSelectedSeq(null); setSelectedFinding(0); setSource(null); setLocationNotice(null); setSurface("incident"); setScopeGeneration(value => value + 1); };
    window.addEventListener("popstate", onPop);
    window.addEventListener("hashchange", onHash);
    return () => { window.removeEventListener("popstate", onPop); window.removeEventListener("hashchange", onHash); };
  }, []);
  useEffect(() => {
    if (typeof window === "undefined") return;
    const focus = new URLSearchParams(window.location.search).get("focus");
    if (!focus) return;
    let cancelled = false;
    const controller = new AbortController();
    void locateEvidence(focus, undefined, controller.signal)
      .then((result) => {
        if (cancelled || navigatedRef.current) return;
        if (result.ok && typeof result.seq === "number") {
          setFocusNotice(null);
          navigatedRef.current = true;
          setSelectedSeq(result.seq);
          setSurface("evidence");
          setSource(null);
          window.history.replaceState(null, "", viewerLocationUrl(window.location.href, "evidence", result.seq));
          return;
        }
        setFocusNotice({
          code: result.error_code ?? "LOCATOR_ERROR",
          message: result.message ?? "The referenced evidence was not found in this bundle.",
        });
      })
      .catch((error: unknown) => {
        // An aborted lookup (for example React strict-mode remount) is not a
        // resolution failure and must not leave a stale notice on screen.
        if (cancelled || controller.signal.aborted || (error instanceof DOMException && error.name === "AbortError")) return;
        setFocusNotice({
          code: "LOCATOR_ERROR",
          message: "The evidence reference could not be resolved.",
        });
      });
    return () => { cancelled = true; controller.abort(); };
    // Resolve the referenced record once on load. Later navigation is user-driven.
  }, []);
  useEffect(() => {
    if ((surface === "evidence" || surface === "timeline") && effectiveSeq !== null && !selectedEvent && eventsQuery.hasNextPage && !eventsQuery.isFetchingNextPage && !eventsQuery.isError) void eventsQuery.fetchNextPage();
  }, [surface, effectiveSeq, selectedEvent, eventsQuery]);

  function openEvidence(seq: number) {
    const origin = surface !== "evidence" ? surface : source;
    navigate("evidence", seq, { from: origin });
  }
  function openTimeline(seq?: number) { navigate("timeline", seq !== undefined ? seq : selectedSeq); }
  function openFindings(index: number) { setSelectedFinding(index); navigate("findings", selectedSeq); }
  function selectRecord(seq: number) { navigate(surface, seq, { from: surface === "evidence" ? source : null }); }
  async function reveal(seq: number, fieldPath: string) { return (await revealMutation.mutateAsync({ seq, field_path: fieldPath, reason: "investigation_review" })).value; }
  const actions: PaletteAction[] = [
    ...investigationSurfaces.map(([id, label]) => ({ id, label: `Open ${label.toLowerCase()}`, group: "navigate" as const, run: () => navigate(id, selectedSeq) })),
    ...(!verifyMutation.isPending ? [{ id: "verify", label: "Verify bundle", group: "verify" as const, run: async () => { await verifyMutation.mutateAsync(); await Promise.all([overviewQuery.refetch(), trustQuery.refetch()]); } }] : []),
    ...(selectedEvent ? [
      { id: "inspect-selected", label: "Inspect selected evidence", group: "inspect" as const, run: () => openEvidence(selectedEvent.seq) },
      { id: "copy-digest", label: "Copy selected evidence digest", group: "copy" as const, run: () => navigator.clipboard.writeText(selectedEvent.hash) },
      { id: "raw", label: "Open raw evidence", group: "raw" as const, run: () => openEvidence(selectedEvent.seq) },
    ] : []),
  ];

  if (overviewQuery.isLoading) return <div className="w-full p-6"><LoadingState label="Loading investigation…"/></div>;
  if (overviewQuery.isError) return <div className="mx-auto w-full max-w-3xl p-6"><h1 className="mb-4 text-xl font-semibold">ATB View</h1><ErrorState onRetry={() => void overviewQuery.refetch()}>The bundle could not be read from the local viewer session. Check that its viewer window is still running, then try again. If the session has expired, reopen the link from ATB View.</ErrorState></div>;
  const overview = overviewQuery.data;
  if (!overview) return <LoadingState/>;
  const blocked = !valid && surface !== "incident" && surface !== "trust";
  const inspector = <div className="space-y-3"><div className="flex flex-wrap items-center justify-between gap-2"><h3 className="text-sm font-semibold">Exact record {effectiveSeq !== null ? `#${effectiveSeq}` : ""}</h3>{surface === "timeline" && effectiveSeq !== null && <button className={actionClass} onClick={() => openEvidence(effectiveSeq)}>Open in Evidence →</button>}{surface === "evidence" && source && <button className={actionClass} onClick={() => navigate(source, selectedSeq)}>← Return to {surfaceLabel(source)}</button>}</div>{eventsQuery.isError ? <ErrorState onRetry={() => void eventsQuery.refetch()}>This record could not be loaded. Your selection is retained.</ErrorState> : eventsQuery.isLoading || (!selectedEvent && eventsQuery.hasNextPage) ? <LoadingState label="Loading selected record…"/> : selectedEvent ? <EventInspector key={`${scopeGeneration}-${overview.bundle_path}-${selectedEvent.seq}-${selectedEvent.hash}`} event={selectedEvent} disabled={!valid || revealMutation.isPending} onReveal={reveal}/> : <EmptyState title={effectiveSeq !== null ? `Event #${effectiveSeq} unavailable` : "Select an event"}>{effectiveSeq !== null ? "The requested sequence was not found in the returned evidence. Choose another record or retry." : "Choose a recorded event to inspect its fields and hash."}</EmptyState>}</div>;
  return <div className="flex h-full min-h-0 w-full bg-background">
    <a href="#dashboard-content" className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-md focus:border focus:border-border focus:bg-card focus:px-3 focus:py-2 focus:text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">Skip to investigation content</a>
    <aside className="hidden w-44 shrink-0 flex-col border-r border-border bg-surface-1 md:flex xl:w-56" aria-label="Investigation navigation"><div className="border-b border-border px-4 py-5"><p className="font-semibold tracking-tight">ATB View</p><p className="mt-1 text-xs text-muted-foreground">Evidence investigation</p></div><InvestigationNavigation active={surface} onSelect={selectSurface} findingCount={overview.finding_count}/><div className="border-t border-border p-4 text-xs leading-5 text-muted-foreground">Local evidence<br/>Independent verification</div></aside>
    <main id="dashboard-content" className="flex min-w-0 flex-1 flex-col">
      <ObjectIdentityHeader bundlePath={overview.bundle_path} eventCount={overview.event_count} surfaceLabel={surfaceLabel(surface)} controls={<><CommandPalette actions={actions}/><RoleSelector/></>} status={<><StatusBadge tone={valid ? "verified" : "danger"}>{valid ? "Hash chain verified" : "Integrity failed"}</StatusBadge><span className="text-muted-foreground">Coverage: <span className="text-foreground">{coverageLabel(overview.integrity_valid, overview.profile)}</span></span><span className="text-muted-foreground">Custody: <span className="text-foreground">{overview.custody_state}</span></span></>}/>
    {locationNotice && <div role="status" data-testid="location-notice" className="shrink-0 border-b border-border bg-surface-1 px-4 py-2 text-xs text-muted-foreground">{locationNotice}</div>}
    {focusNotice && <div role="status" data-testid="focus-notice" className="shrink-0 border-b border-border bg-surface-1 px-4 py-2 text-xs text-muted-foreground"><span className="font-medium text-foreground">Evidence reference not focused. </span>{focusNotice.message}{focusNotice.code ? ` (${focusNotice.code})` : ""} This is a location result, not an integrity or tampering finding.</div>}
    <CompactInvestigationNavigation active={surface} onSelect={selectSurface} findingCount={overview.finding_count}/>
    <div className="min-h-0 flex-1 overflow-y-auto p-4 xl:p-6" key={`${scopeGeneration}-${overview.bundle_path}`}>
      {blocked ? <ErrorState title="Event-derived investigation is blocked"><p>Bundle integrity failed. Open Evidence status to inspect the verification boundary.</p><button className={`${actionClass} mt-3`} onClick={() => navigate("trust", selectedSeq)}>Open Evidence status</button></ErrorState> : <>
      {surface === "incident" && <IncidentSurface overview={overview} findings={findings} timeline={timeline} findingsState={valid && (findingsQuery.isLoading ? <LoadingState label="Deriving priority findings…"/> : findingsQuery.isError ? <ErrorState onRetry={() => void findingsQuery.refetch()}>Priority findings could not be loaded.</ErrorState> : undefined)} openFindings={openFindings} openEvidence={openEvidence} openTimeline={openTimeline} openTrust={() => navigate("trust", selectedSeq)}/>}
      {surface === "findings" && <QueryContent query={findingsQuery}><FindingsSurface findings={findings} acquisitionFindings={acquisitionFindings} selected={selectedFinding} select={setSelectedFinding} openEvidence={openEvidence} openTimeline={openTimeline}/></QueryContent>}
      {surface === "timeline" && <QueryContent query={timelineQuery}><RecordSurface timeline={timeline} selectedSeq={effectiveSeq} select={selectRecord} inspector={inspector} evidence={false}/></QueryContent>}
      {surface === "evidence" && <QueryContent query={timelineQuery}><RecordSurface timeline={timeline} selectedSeq={effectiveSeq} select={selectRecord} inspector={inspector} evidence/></QueryContent>}
      {surface === "context" && <QueryContent query={contextQuery}>{contextQuery.data && <ContextSurface data={contextQuery.data} onOpenEvidence={openEvidence}/>}</QueryContent>}
      {surface === "relationships" && <QueryContent query={relationshipsQuery}>{relationshipsQuery.data && <RelationshipsSurface data={relationshipsQuery.data} timeline={timeline} onOpenEvidence={openEvidence}/>}</QueryContent>}
      {surface === "trust" && <QueryContent query={trustQuery}>{trustQuery.data && <TrustSurface data={trustQuery.data} profile={overview.profile ?? undefined} timeline={timeline} onOpenEvidence={openEvidence}/>}</QueryContent>}
      </>}
    </div></main>
    {verifyMutation.isPending && <div role="status" className="fixed bottom-4 right-4 rounded-md border border-border bg-popover px-4 py-3 text-sm">Verifying bundle…</div>}
    {verifyMutation.isError && <div role="alert" className="fixed bottom-4 right-4 max-w-sm rounded-md border border-danger bg-popover px-4 py-3 text-sm">Verification could not be completed. Try Verify bundle again.</div>}
  </div>;
}
