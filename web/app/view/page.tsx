"use client";

import {
  AlertTriangle,
  Boxes,
  CheckCircle2,
  Clock3,
  FileJson2,
  Fingerprint,
  GitBranch,
  Info,
  ListChecks,
  RefreshCw,
  ShieldCheck,
} from "lucide-react";
import { useMemo, useState } from "react";

import { CommandPalette, type PaletteAction } from "@/app/view/components/CommandPalette";
import { RoleSelector } from "@/app/view/components/role-selector/RoleSelector";
import { Skeleton } from "@/app/view/components/ui/skeleton";
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

function StatePill({ valid, yes, no }: { valid: boolean; yes: string; no: string }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full border px-2 py-1 text-xs ${valid ? "border-verified/40 bg-verified/10 text-verified" : "border-danger/40 bg-danger/10 text-danger"}`}
    >
      {valid ? <CheckCircle2 className="h-3.5 w-3.5" /> : <AlertTriangle className="h-3.5 w-3.5" />}
      {valid ? yes : no}
    </span>
  );
}

function EmptyState({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
      {children}
    </div>
  );
}

function FindingCard({
  finding,
  onOpenEvidence,
}: {
  finding: InvestigationFinding;
  onOpenEvidence?: (seq: number) => void;
}) {
  return (
    <article className="rounded-lg border border-border bg-card p-4">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <h3 className="font-medium">{finding.title}</h3>
        <span className="rounded border border-border px-2 py-0.5 font-mono text-[10px] uppercase text-muted-foreground">
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
                  className="text-primary underline-offset-2 hover:underline"
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
        className={`grid w-full grid-cols-[4rem_1fr] gap-3 text-left hover:bg-muted ${dense ? "py-2" : "py-3"}`}
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
  const selectedEvent = events.find((event) => event.seq === selectedSeq) ?? events[0] ?? null;
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
    { id: "findings", label: "Open findings", group: "navigate", run: () => setSurface("findings") },
    { id: "timeline", label: "Open timeline", group: "navigate", run: () => setSurface("timeline") },
    { id: "context", label: "Open context lineage", group: "navigate", run: () => setSurface("context") },
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

  if (overviewQuery.isLoading)
    return (
      <div className="p-6">
        <Skeleton className="h-48 w-full" />
      </div>
    );
  if (overviewQuery.isError) throw overviewQuery.error;
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
        className="sticky top-0 hidden h-screen w-60 shrink-0 flex-col border-r border-border bg-surface-1 md:flex"
        aria-label="Investigation navigation"
      >
        <div className="border-b border-border p-4">
          <p className="text-sm font-semibold tracking-wide">ATB View</p>
          <p
            className="mt-1 truncate font-mono text-[11px] text-muted-foreground"
            title={overview.bundle_path}
          >
            {displayBundlePath(overview.bundle_path)}
          </p>
        </div>
        <nav className="flex-1 space-y-1 p-2">
          {surfaces.map(([id, label, Icon]) => (
            <button
              key={id}
              type="button"
              onClick={() => setSurface(id)}
              aria-current={surface === id ? "page" : undefined}
              className={`flex w-full items-center gap-3 rounded-md px-3 py-2 text-left text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${surface === id ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
            >
              <Icon className="h-4 w-4" aria-hidden="true" />
              {label}
              {id === "findings" && overview.finding_count > 0 && (
                <span className="ml-auto rounded-full bg-warning/15 px-2 font-mono text-[10px] text-warning">
                  {overview.finding_count}
                </span>
              )}
            </button>
          ))}
        </nav>
        <div className="border-t border-border p-3 text-xs text-muted-foreground">
          Presentation mode only. Authorisation is enforced separately.
        </div>
      </aside>
      <main id="dashboard-content" className="min-w-0 flex-1 overflow-x-hidden">
        <header className="sticky top-0 z-20 flex flex-wrap items-center justify-between gap-3 border-b border-border bg-background/95 px-4 py-3 backdrop-blur md:px-6">
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
        <div className="border-b border-border px-4 py-2 md:hidden">
          <div className="flex gap-1 overflow-x-auto">
            {surfaces.map(([id, label]) => (
              <button
                key={id}
                type="button"
                onClick={() => setSurface(id)}
                className={`whitespace-nowrap rounded px-2 py-1 text-xs ${surface === id ? "bg-primary/10 text-primary" : "text-muted-foreground"}`}
              >
                {label}
              </button>
            ))}
          </div>
        </div>
        <div className="mx-auto max-w-7xl space-y-5 p-4 md:p-6">
          {surface === "incident" && (
            <IncidentSurface
              overview={overview}
              findings={findingsQuery.data?.findings ?? []}
              openFindings={() => setSurface("findings")}
              openEvidence={openEvidence}
            />
          )}
          {surface === "findings" && (
            <section className="space-y-3">
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
            <section className="rounded-lg border border-border bg-card px-4">
              <ol>
                {timelineQuery.data?.events.map((event) => (
                  <TimelineRow
                    key={event.seq}
                    event={event}
                    dense={isDensePresentation(role)}
                    onSelect={openEvidence}
                  />
                ))}
              </ol>
            </section>
          )}
          {surface === "context" && <ContextSurface context={contextQuery.data} />}
          {surface === "relationships" && (
            <RelationshipsSurface
              graph={graphQuery.data ?? null}
              relationships={relationshipsQuery.data?.relationships ?? []}
              disabled={!integrityValid || graphQuery.isFetching}
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
              loadMore={() => void eventsQuery.fetchNextPage()}
            />
          )}
          {surface === "trust" && trustQuery.data && <TrustSurface trust={trustQuery.data} />}
          {!integrityValid && surface !== "incident" && surface !== "trust" && (
            <EmptyState>
              Bundle integrity failed. Event-derived investigation surfaces are blocked; open Trust
              for the verified boundary.
            </EmptyState>
          )}
        </div>
      </main>
      {verifyMutation.isPending && (
        <div className="fixed bottom-4 right-4 flex items-center gap-2 rounded-md border border-border bg-popover px-3 py-2 text-sm shadow-lg">
          <RefreshCw className="h-4 w-4 animate-spin" />
          Verifying bundle…
        </div>
      )}
    </div>
  );
}

function IncidentSurface({
  overview,
  findings,
  openFindings,
  openEvidence,
}: {
  overview: NonNullable<ReturnType<typeof useInvestigationOverviewQuery>["data"]>;
  findings: InvestigationFinding[];
  openFindings: () => void;
  openEvidence: (seq: number) => void;
}) {
  return (
    <>
      <section className="rounded-xl border border-border bg-card p-5">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="max-w-3xl">
            <p className="text-xs uppercase tracking-widest text-muted-foreground">
              What happened?
            </p>
            <h2 className="mt-2 text-xl font-medium">{overview.summary}</h2>
          </div>
          <StatePill
            valid={overview.integrity_valid}
            yes="Hash chain verified"
            no="Integrity failed"
          />
        </div>
      </section>
      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {[
          ["Profile", overview.profile?.profile_id || "Not selected"],
          [
            "Evidence coverage",
            overview.integrity_valid && overview.profile
              ? `${Math.round(overview.profile.coverage_score * 100)}%`
              : overview.integrity_valid
                ? "Not assessed"
                : "Untrusted",
          ],
          ["Important findings", String(overview.finding_count)],
          ["Custody", overview.custody_state],
        ].map(([label, value]) => (
          <div key={label} className="rounded-lg border border-border bg-card p-4">
            <p className="text-xs uppercase tracking-wide text-muted-foreground">{label}</p>
            <p className="mt-2 text-lg">{value}</p>
          </div>
        ))}
      </section>
      <section>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="font-medium">Findings</h2>
          <button
            type="button"
            onClick={openFindings}
            className="text-sm text-primary hover:underline"
          >
            View all
          </button>
        </div>
        <div className="space-y-3">
          {findings.slice(0, 3).map((finding) => (
            <FindingCard
              key={`${finding.flag}-${finding.session_id}`}
              finding={finding}
              onOpenEvidence={openEvidence}
            />
          ))}
          {findings.length === 0 && (
            <EmptyState>
              No findings. This does not prove complete capture or universal absence.
            </EmptyState>
          )}
        </div>
      </section>
    </>
  );
}

function ContextSurface({
  context,
}: {
  context: ReturnType<typeof useInvestigationContextQuery>["data"];
}) {
  return (
    <section className="space-y-5">
      <div>
        <h2 className="font-medium">Context evidence</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Observable sources and transformations only. ATB does not store hidden
          model reasoning. “Supplied to model” only applies when an invocation
          bind is proven.
        </p>
      </div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        {context?.lineage.units.map((unit) => (
          <article key={unit.id} className="rounded-lg border border-border bg-card p-4">
            <p className="text-xs uppercase tracking-wide text-muted-foreground">
              {unit.kind.replaceAll("_", " ")}
            </p>
            <h3 className="mt-2">{unit.source || unit.id}</h3>
            <p className="mt-3 break-all font-mono text-[11px] text-muted-foreground">
              {unit.digest}
            </p>
          </article>
        ))}
        {context?.lineage.units.length === 0 && (
          <EmptyState>
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
  select: (seq: number) => void;
}) {
  const [graphOpen, setGraphOpen] = useState(false);
  return (
    <section className="space-y-4">
      <div className="rounded-lg border border-border bg-card">
        <ul>
          {relationships.map((relationship) => (
            <li key={relationship.id}>
              <button
                type="button"
                onClick={() => select(relationship.source_seq)}
                className="grid w-full gap-1 border-b border-border p-3 text-left text-sm last:border-0 hover:bg-muted md:grid-cols-[10rem_1fr_auto]"
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
              <EmptyState>No shared-identifier relationships were derived.</EmptyState>
            </li>
          )}
        </ul>
      </div>
      <details
        className="rounded-lg border border-border bg-surface-1"
        onToggle={(event) => setGraphOpen(event.currentTarget.open)}
      >
        <summary className="cursor-pointer px-4 py-3 text-sm">
          Optional graph of the same rows. Shared identifiers, not causation.
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
  loadMore,
}: {
  events: ReturnType<typeof flattenEventPages>;
  selected: ReturnType<typeof flattenEventPages>[number] | null;
  selectedSeq: number | null;
  select: (seq: number) => void;
  reveal: (seq: number, path: string) => Promise<unknown>;
  disabled: boolean;
  hasMore: boolean;
  loadMore: () => void;
}) {
  return (
    <section className="grid min-h-[620px] gap-4 lg:grid-cols-[minmax(18rem,0.9fr)_minmax(24rem,1.1fr)]">
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
                className={`w-full border-b border-border p-3 text-left hover:bg-muted ${selectedSeq === event.seq || (selectedSeq === null && selected?.seq === event.seq) ? "bg-primary/10" : ""}`}
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
        {hasMore && (
          <button
            type="button"
            onClick={loadMore}
            className="m-3 rounded border border-border px-3 py-2 text-sm"
          >
            Load more evidence
          </button>
        )}
      </div>
      <div className="overflow-hidden rounded-lg border border-border bg-card">
        <EventInspector event={selected} disabled={disabled} onReveal={reveal} />
      </div>
    </section>
  );
}

function TrustSurface({
  trust,
}: {
  trust: NonNullable<ReturnType<typeof useInvestigationTrustQuery>["data"]>;
}) {
  const coverage = !trust.integrity_valid
    ? "Untrusted"
    : trust.coverage_grade || "Not assessed";
  const questions = [
    {
      title: "Integrity",
      value: trust.integrity_valid ? "Hash chain verified" : "Hash chain failed",
      detail: "RFC 8785 canonical hashes and sequence. Coverage cannot repair a broken chain.",
    },
    {
      title: "Coverage",
      value: coverage,
      detail: "Profile-scoped completeness of recorded evidence, not proof that everything was captured.",
    },
    {
      title: "Corroboration",
      value: trust.external_corroboration ? "External evidence present" : "No external corroboration",
      detail: "Independent records outside this bundle. Absence is not a fail score.",
    },
  ];
  return (
    <section className="space-y-5">
      <div className="rounded-xl border border-border bg-card p-5">
        <Fingerprint className="h-5 w-5 text-primary" />
        <h2 className="mt-3 text-xl">{trust.proof_statement}</h2>
      </div>
      <div className="grid gap-3 lg:grid-cols-3">
        {questions.map((question) => (
          <article key={question.title} className="rounded-lg border border-border bg-card p-4">
            <p className="text-xs uppercase tracking-wide text-muted-foreground">{question.title}</p>
            <p className="mt-2 text-lg">{question.value}</p>
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
            <p className="text-xs uppercase text-muted-foreground">{label}</p>
            <p className="mt-2 capitalize">{value}</p>
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
