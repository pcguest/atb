"use client";

import { Boxes, Clock3, FileJson2, GitBranch, Info, ListChecks, ShieldCheck } from "lucide-react";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { CommandPalette, type PaletteAction } from "./components/CommandPalette";
import { RoleSelector } from "./components/role-selector/RoleSelector";
import { CopyAction, EmptyState, ErrorState, LoadingState, StatusBadge } from "./components/ui/investigation";
import { FindingsSurface, IncidentSurface, RecordSurface, actionClass } from "./components/CoreSurfaces";
import { ContextSurface, RelationshipsSurface, TrustSurface } from "./components/AssuranceSurfaces";
import { EventInspector } from "@/components/dashboard/EventInspector";
import { flattenEventPages, useBundleEventsQuery, useInvestigationContextQuery, useInvestigationFindingsQuery, useInvestigationOverviewQuery, useInvestigationRelationshipsQuery, useInvestigationTimelineQuery, useInvestigationTrustQuery, useRevealFieldMutation, useRunBundleVerifyMutation } from "@/lib/api-client";
import { displayBundlePath } from "@/lib/display-path";

const surfaces = [["incident", "Incident", Info], ["findings", "Findings", ListChecks], ["timeline", "Timeline", Clock3], ["context", "Context", Boxes], ["relationships", "Relationships", GitBranch], ["evidence", "Evidence", FileJson2], ["trust", "Trust", ShieldCheck]] as const;
type Surface = typeof surfaces[number][0];
type QueryState = { isLoading?: boolean; isError?: boolean; data?: unknown; refetch?: () => unknown };

function QueryContent({ query, children }: { query: QueryState; children: ReactNode }) {
  if (query.isError) return <ErrorState onRetry={() => void query.refetch?.()}>The local evidence request failed. Try again while this viewer session is active.</ErrorState>;
  if (query.isLoading || !query.data) return <LoadingState/>;
  return children;
}

export default function ViewPage() {
  const [surface, setSurface] = useState<Surface>("incident");
  const [selectedSeq, setSelectedSeq] = useState<number | null>(null);
  const [selectedFinding, setSelectedFinding] = useState(0);
  const [source, setSource] = useState<Surface | null>(null);
  const [scopeGeneration, setScopeGeneration] = useState(0);
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
  const events = useMemo(() => flattenEventPages(eventsQuery.data?.pages), [eventsQuery.data?.pages]);
  const timeline = timelineQuery.data?.events ?? [];
  const effectiveSeq = selectedSeq ?? timeline[0]?.seq ?? null;
  const selectedEvent = events.find(event => event.seq === effectiveSeq) ?? null;
  const findings = findingsQuery.data?.findings ?? [];

  useEffect(() => {
    const reset = () => { setSelectedSeq(null); setSelectedFinding(0); setSource(null); setSurface("incident"); setScopeGeneration(value => value + 1); };
    window.addEventListener("popstate", reset);
    window.addEventListener("hashchange", reset);
    return () => { window.removeEventListener("popstate", reset); window.removeEventListener("hashchange", reset); };
  }, []);
  useEffect(() => {
    if ((surface === "evidence" || surface === "timeline") && effectiveSeq !== null && !selectedEvent && eventsQuery.hasNextPage && !eventsQuery.isFetchingNextPage && !eventsQuery.isError) void eventsQuery.fetchNextPage();
  }, [surface, effectiveSeq, selectedEvent, eventsQuery]);

  function openEvidence(seq: number) { setSelectedSeq(seq); if (surface !== "evidence") setSource(surface); setSurface("evidence"); }
  function openTimeline(seq?: number) { if (seq !== undefined) setSelectedSeq(seq); setSurface("timeline"); }
  function openFindings(index: number) { setSelectedFinding(index); setSurface("findings"); }
  async function reveal(seq: number, fieldPath: string) { return (await revealMutation.mutateAsync({ seq, field_path: fieldPath, reason: "investigation_review" })).value; }
  const actions: PaletteAction[] = [
    ...surfaces.map(([id, label]) => ({ id, label: `Open ${label.toLowerCase()}`, group: "navigate" as const, run: () => setSurface(id) })),
    ...(!verifyMutation.isPending ? [{ id: "verify", label: "Verify bundle", group: "verify" as const, run: async () => { await verifyMutation.mutateAsync(); } }] : []),
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
  const inspector = <div className="space-y-3"><div className="flex flex-wrap items-center justify-between gap-2"><h3 className="text-sm font-semibold">Exact record {effectiveSeq !== null ? `#${effectiveSeq}` : ""}</h3>{surface === "timeline" && effectiveSeq !== null && <button className={actionClass} onClick={() => openEvidence(effectiveSeq)}>Open in Evidence →</button>}{surface === "evidence" && source && <button className={actionClass} onClick={() => setSurface(source)}>← Return to {source}</button>}</div>{eventsQuery.isError ? <ErrorState onRetry={() => void eventsQuery.refetch()}>This record could not be loaded. Your selection is retained.</ErrorState> : eventsQuery.isLoading || (!selectedEvent && eventsQuery.hasNextPage) ? <LoadingState label="Loading selected record…"/> : selectedEvent ? <EventInspector key={`${scopeGeneration}-${overview.bundle_path}-${selectedEvent.seq}-${selectedEvent.hash}`} event={selectedEvent} disabled={!valid || revealMutation.isPending} onReveal={reveal}/> : <EmptyState title={effectiveSeq !== null ? `Event #${effectiveSeq} unavailable` : "Select an event"}>{effectiveSeq !== null ? "The requested sequence was not found in the returned evidence. Choose another record or retry." : "Choose a recorded event to inspect its fields and hash."}</EmptyState>}</div>;
  return <div className="flex h-full min-h-0 w-full bg-background">
    <aside className="hidden w-44 shrink-0 flex-col border-r border-border bg-surface-1 md:flex xl:w-56" aria-label="Investigation navigation"><div className="border-b border-border px-4 py-5"><p className="font-semibold tracking-tight">ATB View</p><p className="mt-1 text-xs text-muted-foreground">Forensic workspace</p></div><nav className="flex-1 space-y-1 p-2" aria-label="Investigation sequence">{surfaces.map(([id, label, Icon]) => <button key={id} onClick={() => setSurface(id)} aria-label={label} aria-current={surface === id ? "page" : undefined} className={`flex w-full items-center gap-3 rounded-md border px-3 py-2.5 text-left text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${surface === id ? "border-primary/30 bg-primary/10 text-foreground" : "border-transparent text-muted-foreground hover:bg-muted hover:text-foreground"}`}><Icon className="h-4 w-4 shrink-0" aria-hidden="true"/>{label}{id === "findings" && overview.finding_count > 0 && <span className="ml-auto text-xs">{overview.finding_count}</span>}</button>)}</nav><div className="border-t border-border p-4 text-xs leading-5 text-muted-foreground">Local evidence<br/>Independent verification</div></aside>
    <main id="dashboard-content" className="flex min-w-0 flex-1 flex-col"><header className="shrink-0 border-b border-border bg-surface-1 px-4 py-3 xl:px-6"><div className="flex flex-wrap items-start justify-between gap-3"><div className="min-w-0 flex-1"><div className="flex items-center gap-2"><h1 className="min-w-0 truncate text-sm font-semibold" title={overview.bundle_path}>{displayBundlePath(overview.bundle_path)}</h1><CopyAction value={overview.bundle_path} label="Copy bundle path"/></div><p className="mt-1 break-words text-xs text-muted-foreground">Bundle investigation <span aria-hidden="true">/</span> {surfaces.find(([id]) => id === surface)?.[1]}</p></div><div className="flex items-center gap-2"><CommandPalette actions={actions}/><RoleSelector/></div></div><div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 text-xs"><StatusBadge tone={valid ? "verified" : "danger"}>{valid ? "Hash chain verified" : "Integrity failed"}</StatusBadge><span className="break-all text-muted-foreground">Profile: <span className="text-foreground">{overview.profile?.profile_id || "Not selected"}</span></span><span className="text-muted-foreground">Custody: <span className="text-foreground">{overview.custody_state}</span></span></div></header>
    <nav aria-label="Compact investigation navigation" className="flex shrink-0 gap-1 overflow-x-auto border-b border-border p-2 md:hidden">{surfaces.map(([id, label]) => <button key={id} onClick={() => setSurface(id)} aria-current={surface === id ? "page" : undefined} className={`rounded px-3 py-2 text-sm ${surface === id ? "bg-primary/10 text-primary" : "text-muted-foreground"}`}>{label}</button>)}</nav>
    <div className="min-h-0 flex-1 overflow-y-auto p-4 xl:p-6" key={`${scopeGeneration}-${overview.bundle_path}`}>
      {blocked ? <ErrorState title="Event-derived investigation is blocked"><p>Bundle integrity failed. Open Trust to inspect the verification boundary.</p><button className={`${actionClass} mt-3`} onClick={() => setSurface("trust")}>Open Trust</button></ErrorState> : <>
      {surface === "incident" && <IncidentSurface overview={overview} findings={findings} timeline={timeline} findingsState={valid && (findingsQuery.isLoading ? <LoadingState label="Deriving priority findings…"/> : findingsQuery.isError ? <ErrorState onRetry={() => void findingsQuery.refetch()}>Priority findings could not be loaded.</ErrorState> : undefined)} openFindings={openFindings} openEvidence={openEvidence} openTimeline={openTimeline} openTrust={() => setSurface("trust")}/>}
      {surface === "findings" && <QueryContent query={findingsQuery}><FindingsSurface findings={findings} selected={selectedFinding} select={setSelectedFinding} openEvidence={openEvidence} openTimeline={openTimeline}/></QueryContent>}
      {surface === "timeline" && <QueryContent query={timelineQuery}><RecordSurface timeline={timeline} selectedSeq={effectiveSeq} select={setSelectedSeq} inspector={inspector} evidence={false}/></QueryContent>}
      {surface === "evidence" && <QueryContent query={timelineQuery}><RecordSurface timeline={timeline} selectedSeq={effectiveSeq} select={setSelectedSeq} inspector={inspector} evidence/></QueryContent>}
      {surface === "context" && <QueryContent query={contextQuery}>{contextQuery.data && <ContextSurface data={contextQuery.data} onOpenEvidence={openEvidence}/>}</QueryContent>}
      {surface === "relationships" && <QueryContent query={relationshipsQuery}>{relationshipsQuery.data && <RelationshipsSurface data={relationshipsQuery.data} timeline={timeline} onOpenEvidence={openEvidence}/>}</QueryContent>}
      {surface === "trust" && <QueryContent query={trustQuery}>{trustQuery.data && <TrustSurface data={trustQuery.data} profile={overview.profile ?? undefined} timeline={timeline} onOpenEvidence={openEvidence}/>}</QueryContent>}
      </>}
    </div></main>
    {verifyMutation.isPending && <div role="status" className="fixed bottom-4 right-4 rounded-md border border-border bg-popover px-4 py-3 text-sm">Verifying bundle…</div>}
    {verifyMutation.isError && <div role="alert" className="fixed bottom-4 right-4 max-w-sm rounded-md border border-danger bg-popover px-4 py-3 text-sm">Verification could not be completed. Try Verify bundle again.</div>}
  </div>;
}
