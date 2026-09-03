"use client";

import {
  AlertTriangle,
  Boxes,
  Clock3,
  FileJson2,
  Fingerprint,
  GitBranch,
  Info,
  ListChecks,
  RefreshCw,
  ShieldCheck,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { CommandPalette, type PaletteAction } from "@/app/view/components/CommandPalette";
import { RoleSelector } from "@/app/view/components/role-selector/RoleSelector";
import {
  CopyAction,
  EmptyState,
  ErrorState,
  LoadingState,
  StatusBadge,
  SurfaceHeader,
} from "@/app/view/components/ui/investigation";
import { EventInspector } from "@/components/dashboard/EventInspector";
import { TraceGraph } from "@/components/dashboard/TraceGraph";
import {
  flattenEventPages,
  useBundleEventsQuery,
  useBundleGraphQuery,
  useInvestigationContextQuery,
  useInvestigationFindingsQuery,
  useInvestigationOverviewQuery,
  useInvestigationRelationshipsQuery,
  useInvestigationTimelineQuery,
  useInvestigationTrustQuery,
  useRevealFieldMutation,
  useRunBundleVerifyMutation,
} from "@/lib/api-client";
import { displayBundlePath } from "@/lib/display-path";
import { isDensePresentation } from "@/lib/roles";
import { useUIStore } from "@/lib/state/ui-store";
import type { InvestigationFinding, TimelineEvent } from "@/lib/types";

const surfaces = [
  ["incident", "Incident", Info],
  ["findings", "Findings", ListChecks],
  ["timeline", "Timeline", Clock3],
  ["context", "Context", Boxes],
  ["relationships", "Relationships", GitBranch],
  ["evidence", "Evidence", FileJson2],
  ["trust", "Trust", ShieldCheck],
] as const;
type Surface = (typeof surfaces)[number][0];

function FindingCard({
  finding,
  onOpenEvidence,
}: {
  finding: InvestigationFinding;
  onOpenEvidence?: (seq: number) => void;
}) {
  return (
    <article className="rounded-lg border border-border bg-card p-4 transition-colors hover:border-primary/30">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <h3 className="font-medium">{finding.title}</h3>
        <span className="rounded border border-border bg-muted/45 px-2 py-0.5 font-mono text-[10px] uppercase text-muted-foreground">
          {finding.severity || "inconclusive"}
        </span>
      </div>
      <p className="mt-2 text-sm text-secondary-foreground">{finding.detail}</p>
      <dl className="mt-4 grid gap-3 text-sm md:grid-cols-2">
        <div>
          <dt className="text-xs uppercase tracking-wide text-muted-foreground">
            ATB can conclude
          </dt>
          <dd className="mt-1">{finding.what_atb_can_conclude}</dd>
        </div>
        <div>
          <dt className="text-xs uppercase tracking-wide text-muted-foreground">
            ATB cannot conclude
          </dt>
          <dd className="mt-1 text-muted-foreground">{finding.what_atb_cannot_conclude}</dd>
        </div>
      </dl>
      {finding.event_seqs.length > 0 && (
        <p className="mt-3 font-mono text-xs text-muted-foreground">
          Supporting events:{" "}
          {finding.event_seqs.map((seq, index) => (
            <span key={seq}>
              {index > 0 ? ", " : ""}
              {onOpenEvidence ? (
                <button
                  type="button"
                  onClick={() => onOpenEvidence(seq)}
                  className="rounded-sm text-primary underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  #{seq}
                </button>
              ) : (
                `#${seq}`
              )}
            </span>
          ))}
        </p>
      )}
    </article>
  );
}

function TimelineRow({
  event,
  dense,
  onSelect,
}: {
  event: TimelineEvent;
  dense: boolean;
  onSelect: (seq: number) => void;
}) {
  return (
    <li className="border-b border-border last:border-0">
      <button
        type="button"
        onClick={() => onSelect(event.seq)}
        className={`grid w-full grid-cols-[3.5rem_1fr] gap-3 px-3 text-left transition-colors hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring ${dense ? "py-2" : "py-3"}`}
      >
        <span className="font-mono text-xs text-muted-foreground">#{event.seq}</span>
        <div className="min-w-0">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span>{event.label}</span>
            <time className="font-mono text-xs text-muted-foreground">
              {event.timestamp || "time unavailable"}
            </time>
          </div>
          <p className="mt-1 truncate font-mono text-[11px] text-muted-foreground">
            {event.type} · {event.hash}
          </p>
        </div>
      </button>
    </li>
  );
}

export default function ViewPage() {
  const role = useUIStore((state) => state.role);
  const [surface, setSurface] = useState<Surface>("incident");
  const [selectedSeq, setSelectedSeq] = useState<number | null>(null);
  const overviewQuery = useInvestigationOverviewQuery();
  const trustQuery = useInvestigationTrustQuery();
  const integrityValid = overviewQuery.data?.integrity_valid === true;
  const findingsQuery = useInvestigationFindingsQuery(integrityValid);
  const timelineQuery = useInvestigationTimelineQuery(integrityValid);
  const contextQuery = useInvestigationContextQuery(integrityValid);
  const relationshipsQuery = useInvestigationRelationshipsQuery(integrityValid);
  const eventsQuery = useBundleEventsQuery(integrityValid && surface === "evidence");
  const graphQuery = useBundleGraphQuery(integrityValid && surface === "relationships");
  const verifyMutation = useRunBundleVerifyMutation();
  const revealMutation = useRevealFieldMutation();
  const events = useMemo(
    () => flattenEventPages(eventsQuery.data?.pages),
    [eventsQuery.data?.pages],
  );
  const selectedEvent =
    selectedSeq === null
      ? (events[0] ?? null)
      : (events.find((event) => event.seq === selectedSeq) ?? null);

  useEffect(() => {
    if (
      surface === "evidence" &&
      selectedSeq !== null &&
      !selectedEvent &&
      eventsQuery.hasNextPage &&
      !eventsQuery.isFetchingNextPage
    ) {
      void eventsQuery.fetchNextPage();
    }
  }, [eventsQuery, selectedEvent, selectedSeq, surface]);
  const actions: PaletteAction[] = [
    {
      id: "verify",
      label: "Verify bundle",
      hint: "operation",
      group: "operate",
      run: async () => {
        await verifyMutation.mutateAsync();
      },
    },
    {
      id: "findings",
      label: "Open findings",
      group: "navigate",
      run: () => setSurface("findings"),
    },
    {
      id: "timeline",
      label: "Open timeline",
      group: "navigate",
      run: () => setSurface("timeline"),
    },
    {
      id: "context",
      label: "Open context lineage",
      group: "navigate",
      run: () => setSurface("context"),
    },
    { id: "raw", label: "Show raw evidence", group: "navigate", run: () => setSurface("evidence") },
    { id: "trust", label: "Open Trust", group: "navigate", run: () => setSurface("trust") },
    {
      id: "copy-digest",
      label: "Copy selected evidence digest",
      group: "copy",
      run: async () => {
        if (selectedEvent?.hash && navigator.clipboard)
          await navigator.clipboard.writeText(selectedEvent.hash);
      },
    },
  ];

  if (overviewQuery.isLoading) return <LoadingState label="Loading investigation…" />;
  if (overviewQuery.isError)
    return (
      <div className="p-6">
        <ErrorState>
          Check that the bundle exists and that the local viewer session is still active, then
          reload.
        </ErrorState>
      </div>
    );
  const overview = overviewQuery.data;
  if (!overview) return null;
  async function reveal(seq: number, fieldPath: string) {
    const response = await revealMutation.mutateAsync({
      seq,
      field_path: fieldPath,
      reason: "investigation_review",
    });
    return response.value;
  }
  function openEvidence(seq: number) {
    setSelectedSeq(seq);
    setSurface("evidence");
  }

  return (
    <div className="flex min-h-full w-full bg-background">
      <aside
        className="sticky top-[var(--view-banner-offset)] hidden h-[calc(100vh-var(--view-banner-offset))] w-64 shrink-0 flex-col border-r border-border bg-surface-1 md:flex"
        aria-label="Investigation navigation"
      >
        <div className="border-b border-border px-4 py-5">
          <div className="flex items-center gap-2.5">
            <span className="grid h-8 w-8 place-items-center rounded-md border border-primary/35 bg-primary/10 font-mono text-xs font-semibold text-primary">
              A
            </span>
            <div>
              <p className="text-sm font-semibold tracking-wide">ATB View</p>
              <p className="text-[11px] text-muted-foreground">Forensic investigation</p>
            </div>
          </div>
          <p
            className="mt-4 truncate rounded-md border border-border bg-background/35 px-2.5 py-2 font-mono text-[11px] text-muted-foreground"
            title={overview.bundle_path}
          >
            {displayBundlePath(overview.bundle_path)}
          </p>
        </div>
        <nav className="flex-1 space-y-1 p-2" aria-label="Investigation sequence">
          {surfaces.map(([id, label, Icon], index) => (
            <button
              key={id}
              type="button"
              onClick={() => setSurface(id)}
              aria-label={label}
              aria-current={surface === id ? "page" : undefined}
              className={`group flex w-full items-center gap-3 rounded-md border px-3 py-2.5 text-left text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${surface === id ? "border-primary/25 bg-primary/10 text-foreground" : "border-transparent text-muted-foreground hover:bg-muted hover:text-foreground"}`}
            >
              <span className="w-4 font-mono text-[10px] text-muted-foreground">{index + 1}</span>
              <Icon
                className={`h-4 w-4 ${surface === id ? "text-primary" : ""}`}
                aria-hidden="true"
              />
              {label}
              {id === "findings" && overview.finding_count > 0 && (
                <span className="ml-auto rounded-full bg-warning/15 px-2 font-mono text-[10px] text-warning">
                  {overview.finding_count}
                </span>
              )}
            </button>
          ))}
        </nav>
        <div className="border-t border-border p-4 text-xs leading-5 text-muted-foreground">
          Evidence is read-only here. Authorisation is enforced separately.
        </div>
      </aside>
      <main id="dashboard-content" className="min-w-0 flex-1 overflow-x-hidden">
        <header className="sticky top-[var(--view-banner-offset)] z-20 flex min-h-16 flex-wrap items-center justify-between gap-3 border-b border-border bg-background/95 px-4 py-3 backdrop-blur md:px-7">
          <div>
            <h1 className="text-lg font-semibold">
              {surfaces.find(([id]) => id === surface)?.[1]}
            </h1>
            <p className="text-xs text-muted-foreground">
              Evidence-led investigation · detail on demand
            </p>
          </div>
          <div className="flex items-center gap-2">
            <CommandPalette actions={actions} />
            <RoleSelector />
          </div>
        </header>
        <div className="border-b border-border bg-surface-1 px-4 py-2 md:hidden">
          <div
            className="flex gap-1 overflow-x-auto"
            role="tablist"
            aria-label="Investigation surfaces"
          >
            {surfaces.map(([id, label]) => (
              <button
                key={id}
                type="button"
                role="tab"
                aria-selected={surface === id}
                aria-current={surface === id ? "page" : undefined}
                onClick={() => setSurface(id)}
                className={`min-h-9 whitespace-nowrap rounded-md px-3 py-1.5 text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${surface === id ? "bg-primary/10 text-primary" : "text-muted-foreground"}`}
              >
                {label}
              </button>
            ))}
          </div>
        </div>
        <div className="mx-auto max-w-[88rem] space-y-6 p-4 md:p-7">
          {surface === "incident" && (
            <IncidentSurface
              overview={overview}
              findings={findingsQuery.data?.findings ?? []}
              findingsLoading={findingsQuery.isLoading}
              findingsError={findingsQuery.isError}
              openFindings={() => setSurface("findings")}
              openEvidence={openEvidence}
            />
          )}
          {surface === "findings" && (
            <section className="space-y-4">
              <SurfaceHeader
                eyebrow="Triage"
                title="Findings"
                description="Bounded conclusions derived from the records in this bundle. Open a supporting event to inspect the exact evidence."
              />
              {findingsQuery.isLoading && <LoadingState label="Deriving findings…" />}
              {findingsQuery.isError && (
                <ErrorState>
                  Reload the investigation or inspect the bundle with the CLI.
                </ErrorState>
              )}
              {findingsQuery.data?.findings.map((finding) => (
                <FindingCard
                  key={`${finding.flag}-${finding.session_id}`}
                  finding={finding}
                  onOpenEvidence={openEvidence}
                />
              ))}
              {findingsQuery.data?.findings.length === 0 && (
                <EmptyState>
                  No findings. This does not prove complete capture or universal absence.
                </EmptyState>
              )}
            </section>
          )}
          {surface === "timeline" && (
            <section className="space-y-4">
              <SurfaceHeader
                eyebrow="Sequence"
                title="Timeline"
                description="Recorded events in order. Select a row to open the exact record and hash."
              />
              {timelineQuery.isLoading && <LoadingState label="Loading timeline…" />}
              {timelineQuery.isError && (
                <ErrorState>
                  The ordered event view is unavailable. Reload or use atb inspect.
                </ErrorState>
              )}
              <ol className="overflow-hidden rounded-lg border border-border bg-card">
                {timelineQuery.data?.events.map((event) => (
                  <TimelineRow
                    key={event.seq}
                    event={event}
                    dense={isDensePresentation(role)}
                    onSelect={openEvidence}
                  />
                ))}
              </ol>
              {timelineQuery.data?.events.length === 0 && (
                <EmptyState title="No timeline events">
                  No event-derived timeline is available for this bundle.
                </EmptyState>
              )}
            </section>
          )}
          {surface === "context" && (
            <ContextSurface
              context={contextQuery.data}
              loading={contextQuery.isLoading}
              error={contextQuery.isError}
            />
          )}
          {surface === "relationships" && (
            <RelationshipsSurface
              graph={graphQuery.data ?? null}
              relationships={relationshipsQuery.data?.relationships ?? []}
              disabled={!integrityValid || graphQuery.isFetching}
              loading={relationshipsQuery.isLoading}
              error={relationshipsQuery.isError}
              select={openEvidence}
            />
          )}
          {surface === "evidence" && (
            <EvidenceSurface
              events={events}
              selected={selectedEvent}
              selectedSeq={selectedSeq}
              select={setSelectedSeq}
              reveal={reveal}
              disabled={!integrityValid || revealMutation.isPending}
              hasMore={Boolean(eventsQuery.hasNextPage)}
              loading={
                eventsQuery.isLoading ||
                Boolean(selectedSeq !== null && !selectedEvent && eventsQuery.isFetchingNextPage)
              }
              error={eventsQuery.isError}
              loadMore={() => void eventsQuery.fetchNextPage()}
            />
          )}
          {surface === "trust" && (
            <TrustSurface
              trust={trustQuery.data}
              loading={trustQuery.isLoading}
              error={trustQuery.isError}
            />
          )}
          {!integrityValid && surface !== "incident" && surface !== "trust" && (
            <EmptyState>
              Bundle integrity failed. Event-derived investigation surfaces are blocked; open Trust
              for the verified boundary.
            </EmptyState>
          )}
        </div>
      </main>
      {verifyMutation.isPending && (
        <div
          className="fixed bottom-4 right-4 flex items-center gap-2 rounded-md border border-border bg-popover px-3 py-2 text-sm shadow-lg"
          role="status"
          aria-live="polite"
        >
          <RefreshCw className="h-4 w-4 animate-spin" />
          Verifying bundle…
        </div>
      )}
      {verifyMutation.isError && (
        <div
          className="fixed bottom-4 right-4 max-w-sm rounded-md border border-danger/40 bg-popover px-4 py-3 text-sm shadow-lg"
          role="alert"
        >
          Verification could not be completed. Check the selected profile and try again.
        </div>
      )}
    </div>
  );
}

function evidenceCoverageLabel(
  integrityValid: boolean,
  profile:
    | {
        coverage_score?: number;
        coverage_grade?: string;
      }
    | null
    | undefined,
): string {
  if (!integrityValid) {
    return "Untrusted";
  }
  const grade = profile?.coverage_grade?.trim();
  if (!grade) {
    return "Not assessed";
  }
  const score = profile?.coverage_score;
  if (typeof score === "number" && Number.isFinite(score)) {
    return `${Math.round(score * 100)}%`;
  }
  return grade;
}

function IncidentSurface({
  overview,
  findings,
  findingsLoading,
  findingsError,
  openFindings,
  openEvidence,
}: {
  overview: NonNullable<ReturnType<typeof useInvestigationOverviewQuery>["data"]>;
  findings: InvestigationFinding[];
  findingsLoading: boolean;
  findingsError: boolean;
  openFindings: () => void;
  openEvidence: (seq: number) => void;
}) {
  return (
    <section className="space-y-6" aria-labelledby="incident-heading">
      <div className="rounded-xl border border-border bg-card p-5 md:p-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="max-w-3xl">
            <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
              What happened?
            </p>
            <h2
              id="incident-heading"
              className="mt-2 text-xl font-semibold leading-8 tracking-tight md:text-2xl"
            >
              {overview.summary}
            </h2>
            <p className="mt-3 text-sm text-muted-foreground">
              {overview.event_count} recorded event{overview.event_count === 1 ? "" : "s"} ·
              conclusions are bounded to this bundle
            </p>
          </div>
          <StatusBadge tone={overview.integrity_valid ? "verified" : "danger"}>
            {overview.integrity_valid ? "Hash chain verified" : "Integrity failed"}
          </StatusBadge>
        </div>
      </div>
      <dl className="grid overflow-hidden rounded-lg border border-border bg-card sm:grid-cols-2 xl:grid-cols-4">
        {[
          ["Profile", overview.profile?.profile_id || "Not selected"],
          ["Evidence coverage", evidenceCoverageLabel(overview.integrity_valid, overview.profile)],
          ["Important findings", String(overview.finding_count)],
          ["Custody", overview.custody_state],
        ].map(([label, value]) => (
          <div
            key={label}
            className="border-b border-border p-4 last:border-0 sm:border-r sm:[&:nth-child(2)]:border-r-0 xl:border-b-0 xl:[&:nth-child(2)]:border-r xl:last:border-r-0"
          >
            <dt className="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
              {label}
            </dt>
            <dd className="mt-2 break-words text-sm font-medium">{value}</dd>
          </div>
        ))}
      </dl>
      <div>
        <div className="mb-3 flex items-end justify-between border-b border-border pb-3">
          <div>
            <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
              Triage
            </p>
            <h2 className="mt-1 font-semibold">Priority findings</h2>
          </div>
          <button
            type="button"
            onClick={openFindings}
            className="rounded-sm text-sm font-medium text-primary underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            View all
          </button>
        </div>
        <div className="space-y-3">
          {findingsLoading && <LoadingState label="Deriving priority findings…" />}
          {findingsError && (
            <ErrorState>Open Findings to retry or inspect the bundle with the CLI.</ErrorState>
          )}
          {findings.slice(0, 3).map((finding) => (
            <FindingCard
              key={`${finding.flag}-${finding.session_id}`}
              finding={finding}
              onOpenEvidence={openEvidence}
            />
          ))}
          {!findingsLoading && !findingsError && findings.length === 0 && (
            <EmptyState title="No findings in recorded evidence">
              No findings. This does not prove complete capture or universal absence.
            </EmptyState>
          )}
        </div>
      </div>
    </section>
  );
}

function ContextSurface({
  context,
  loading,
  error,
}: {
  context: ReturnType<typeof useInvestigationContextQuery>["data"];
  loading: boolean;
  error: boolean;
}) {
  const header = (
    <SurfaceHeader
      eyebrow="Provenance"
      title="Context evidence"
      description="Observable sources and transformations only. ATB does not store hidden model reasoning. “Supplied to model” applies only when an invocation bind is proven."
    />
  );
  if (loading)
    return (
      <section className="space-y-5">
        {header}
        <LoadingState label="Loading context evidence…" />
      </section>
    );
  if (error)
    return (
      <section className="space-y-5">
        {header}
        <ErrorState>Reload the investigation or inspect context records with the CLI.</ErrorState>
      </section>
    );
  return (
    <section className="space-y-5">
      {header}
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        {context?.lineage.units.map((unit) => (
          <article key={unit.id} className="rounded-lg border border-border bg-card p-4">
            <p className="text-xs uppercase tracking-wide text-muted-foreground">
              {unit.kind.replaceAll("_", " ")}
            </p>
            <h3 className="mt-2">{unit.source || unit.id}</h3>
            <div className="mt-3 flex items-start gap-2">
              <p className="min-w-0 flex-1 break-all font-mono text-[11px] leading-5 text-muted-foreground">
                {unit.digest}
              </p>
              <CopyAction value={unit.digest} label={`Copy digest for ${unit.source || unit.id}`} />
            </div>
          </article>
        ))}
        {context?.lineage.units.length === 0 && (
          <EmptyState title="No structured context lineage">
            {(context?.capabilities.length ?? 0) > 0
              ? "Retrieval was recorded; structured lineage units were not."
              : "No context lineage units or retrieval capabilities were recorded. This does not prove retrieval did not occur."}
          </EmptyState>
        )}
      </div>
      {(context?.capabilities.length ?? 0) > 0 && (
        <div className="space-y-2">
          <h3 className="text-sm font-medium">Recorded retrieval capabilities</h3>
          <ul className="space-y-2">
            {context?.capabilities.map((capability) => (
              <li
                key={`${capability.event_sequence}-${capability.raw_event_type}`}
                className="rounded-lg border border-border bg-card p-3 text-sm"
              >
                <p className="font-medium">{capability.name.replaceAll("_", " ")}</p>
                {capability.raw_event_type.startsWith("atb.event.rag_") && (
                  <p className="mt-1 text-xs text-muted-foreground">PageIndex adapter</p>
                )}
                <p className="mt-1 font-mono text-xs text-muted-foreground">
                  {capability.raw_event_type} · event #{capability.event_sequence}
                </p>
              </li>
            ))}
          </ul>
        </div>
      )}
      <div className="space-y-3">
        {context?.lineage.operations.map((operation) => (
          <article key={operation.id} className="rounded-lg border border-border bg-card p-4">
            <div className="flex justify-between gap-3">
              <h3 className="capitalize">{operation.type.replaceAll("_", " ")}</h3>
              <span className="font-mono text-xs text-muted-foreground">
                event #{operation.event_sequence}
              </span>
            </div>
            <p className="mt-2 text-sm text-muted-foreground">
              {operation.input_unit_ids.length} input unit(s) → {operation.output_unit_ids.length}{" "}
              output unit(s){operation.method ? ` · ${operation.method}` : ""}
            </p>
            {operation.input_token_count !== undefined &&
              operation.output_token_count !== undefined && (
                <p className="mt-2 text-sm">
                  Before {operation.input_token_count} tokens · After {operation.output_token_count}{" "}
                  tokens · Reduction{" "}
                  {Math.max(
                    0,
                    Math.round(
                      (1 -
                        operation.output_token_count / Math.max(1, operation.input_token_count)) *
                        100,
                    ),
                  )}
                  %
                </p>
              )}
          </article>
        ))}
      </div>
    </section>
  );
}

function RelationshipsSurface({
  graph,
  relationships,
  disabled,
  loading,
  error,
  select,
}: {
  graph: Parameters<typeof TraceGraph>[0]["graph"];
  relationships: Array<{
    id: string;
    source_seq: number;
    target_seq: number;
    kind: string;
    evidence_value: string;
  }>;
  disabled: boolean;
  loading: boolean;
  error: boolean;
  select: (seq: number) => void;
}) {
  const [graphOpen, setGraphOpen] = useState(false);
  const header = (
    <SurfaceHeader
      eyebrow="Links"
      title="Relationships"
      description="Shared identifiers derived from recorded fields. These rows do not assert causation."
    />
  );
  if (loading)
    return (
      <section className="space-y-4">
        {header}
        <LoadingState label="Deriving relationships…" />
      </section>
    );
  if (error)
    return (
      <section className="space-y-4">
        {header}
        <ErrorState>
          Reload the investigation or inspect relationship fields in Evidence.
        </ErrorState>
      </section>
    );
  return (
    <section className="space-y-4">
      {header}
      <div className="rounded-lg border border-border bg-card">
        <ul>
          {relationships.map((relationship) => (
            <li key={relationship.id}>
              <button
                type="button"
                onClick={() => select(relationship.source_seq)}
                className="grid w-full gap-1 border-b border-border p-3 text-left text-sm last:border-0 hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring md:grid-cols-[10rem_1fr_auto]"
              >
                <span className="font-mono text-xs">
                  #{relationship.source_seq} → #{relationship.target_seq}
                </span>
                <span>{relationship.kind.replaceAll("_", " ")}</span>
                <span className="truncate font-mono text-xs text-muted-foreground">
                  {relationship.evidence_value}
                </span>
              </button>
            </li>
          ))}
          {relationships.length === 0 && (
            <li className="p-4">
              <EmptyState title="No derived relationships">
                No shared-identifier relationships were derived.
              </EmptyState>
            </li>
          )}
        </ul>
      </div>
      <details
        className="rounded-lg border border-border bg-surface-1"
        onToggle={(event) => setGraphOpen(event.currentTarget.open)}
      >
        <summary className="cursor-pointer px-4 py-3 text-sm font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring">
          Open optional graph{" "}
          <span className="font-normal text-muted-foreground">· same rows, not causation</span>
        </summary>
        {graphOpen && (
          <div className="h-[420px] overflow-hidden border-t border-border">
            <TraceGraph
              graph={graph}
              disabled={disabled}
              onSelectSeq={select}
              layout="dagre-top-down"
            />
          </div>
        )}
      </details>
    </section>
  );
}

function EvidenceSurface({
  events,
  selected,
  selectedSeq,
  select,
  reveal,
  disabled,
  hasMore,
  loading,
  error,
  loadMore,
}: {
  events: ReturnType<typeof flattenEventPages>;
  selected: ReturnType<typeof flattenEventPages>[number] | null;
  selectedSeq: number | null;
  select: (seq: number) => void;
  reveal: (seq: number, path: string) => Promise<unknown>;
  disabled: boolean;
  hasMore: boolean;
  loading: boolean;
  error: boolean;
  loadMore: () => void;
}) {
  return (
    <section className="space-y-4">
      <SurfaceHeader
        eyebrow="Source records"
        title="Evidence"
        description="Select a record to inspect its captured fields and hash. Masked values require an explicit, audited reveal."
        action={
          selected?.hash ? (
            <CopyAction value={selected.hash} label="Copy selected evidence digest" />
          ) : undefined
        }
      />
      {error && (
        <ErrorState>
          Reload the investigation or use <span className="font-mono">atb inspect</span> for the
          exact records.
        </ErrorState>
      )}
      <div className="grid min-h-[620px] gap-4 lg:grid-cols-[minmax(18rem,0.85fr)_minmax(24rem,1.15fr)]">
        <div className="overflow-hidden rounded-lg border border-border bg-card">
          <div className="border-b border-border px-4 py-3">
            <h2 className="font-medium">Exact records</h2>
            <p className="text-xs text-muted-foreground">
              Raw records remain the verifiable source of truth.
            </p>
          </div>
          <ol className="max-h-[70vh] overflow-y-auto">
            {events.map((event) => (
              <li key={event.seq}>
                <button
                  type="button"
                  onClick={() => select(event.seq)}
                  aria-current={selected?.seq === event.seq ? "true" : undefined}
                  className={`w-full border-b border-border p-3 text-left transition-colors hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring ${selected?.seq === event.seq ? "bg-primary/10 shadow-[inset_3px_0_0_hsl(var(--primary))]" : ""}`}
                >
                  <span className="font-mono text-xs text-muted-foreground">#{event.seq}</span>
                  <p className="mt-1 truncate text-sm">{event.type}</p>
                  <p className="mt-1 truncate font-mono text-[10px] text-muted-foreground">
                    {event.hash}
                  </p>
                </button>
              </li>
            ))}
          </ol>
          {events.length === 0 && !loading && !error && (
            <div className="p-3">
              <EmptyState title="No evidence records">
                This bundle contains no event records to inspect.
              </EmptyState>
            </div>
          )}
          {loading && (
            <div
              className="border-t border-border px-3 py-3 text-xs text-muted-foreground"
              role="status"
            >
              Loading requested record…
            </div>
          )}
          {hasMore && !loading && (
            <button
              type="button"
              onClick={loadMore}
              className="m-3 rounded-md border border-border px-3 py-2 text-sm hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              Load more evidence
            </button>
          )}
        </div>
        <div className="overflow-hidden rounded-lg border border-border bg-card">
          {selected ? (
            <EventInspector event={selected} disabled={disabled} onReveal={reveal} />
          ) : (
            <EmptyState title={loading ? "Locating evidence" : "Select a record"}>
              {loading
                ? `Loading pages until event #${selectedSeq ?? ""} is available.`
                : "Choose an evidence row to inspect its fields and hash."}
            </EmptyState>
          )}
        </div>
      </div>
    </section>
  );
}

function TrustSurface({
  trust,
  loading,
  error,
}: {
  trust: ReturnType<typeof useInvestigationTrustQuery>["data"];
  loading: boolean;
  error: boolean;
}) {
  const header = (
    <SurfaceHeader
      eyebrow="Assurance boundary"
      title="Trust"
      description="Three independent questions. Integrity, profile coverage, and external custody are not collapsed into one score."
    />
  );
  if (loading)
    return (
      <section className="space-y-5">
        {header}
        <LoadingState label="Loading trust evidence…" />
      </section>
    );
  if (error || !trust)
    return (
      <section className="space-y-5">
        {header}
        <ErrorState>Run verification again or inspect the trust report with the CLI.</ErrorState>
      </section>
    );
  const coverage = !trust.integrity_valid ? "Untrusted" : trust.coverage_grade || "Not assessed";
  const questions = [
    {
      label: "Integrity",
      title: "Is recorded evidence intact?",
      value: trust.integrity_valid ? "Hash chain verified" : "Hash chain failed",
      detail: "RFC 8785 canonical hashes and sequence. Coverage cannot repair a broken chain.",
    },
    {
      label: "Coverage",
      title: "Does the selected evidence profile pass?",
      value: coverage,
      detail:
        "Profile-scoped completeness of recorded evidence, not proof that everything was captured.",
    },
    {
      label: "Corroboration",
      title: "Is external or organisational custody recorded?",
      value: trust.external_corroboration
        ? "External evidence present"
        : "No external corroboration",
      detail: "Independent records outside this bundle. Absence is not a fail score.",
    },
  ];
  return (
    <section className="space-y-5">
      {header}
      <div className="rounded-xl border border-border bg-card p-5 md:p-6">
        <div className="flex gap-3">
          <Fingerprint className="mt-0.5 h-5 w-5 shrink-0 text-primary" />
          <div>
            <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
              What ATB proves
            </p>
            <h2 className="mt-2 text-lg font-semibold leading-7">{trust.proof_statement}</h2>
          </div>
        </div>
      </div>
      <div className="grid gap-3 lg:grid-cols-3">
        {questions.map((question) => (
          <article key={question.title} className="rounded-lg border border-border bg-card p-5">
            <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
              {question.label}
            </p>
            <h2 className="mt-2 text-sm font-semibold">{question.title}</h2>
            <p className="mt-3 font-semibold">{question.value}</p>
            <p className="mt-2 text-sm text-muted-foreground">{question.detail}</p>
          </article>
        ))}
      </div>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {[
          ["Signature", trust.signature_status],
          ["Anchor", trust.anchor_status],
          ["Custody", trust.custody_state],
          ["Canonicalisation", trust.canonicalisation],
        ].map(([label, value]) => (
          <div key={label} className="rounded-lg border border-border bg-card p-4">
            <p className="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
              {label}
            </p>
            <p className="mt-2 text-sm font-medium capitalize">{value}</p>
          </div>
        ))}
      </div>
      <div className="rounded-lg border border-warning/30 bg-warning/5 p-5">
        <h2 className="font-medium">What ATB does not prove</h2>
        <ul className="mt-3 grid gap-2 text-sm text-muted-foreground md:grid-cols-2">
          {trust.limitations.map((limitation) => (
            <li key={limitation} className="flex gap-2">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-warning" />
              {limitation}
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
