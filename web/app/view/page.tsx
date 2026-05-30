"use client";

import { useEffect, useMemo, useState } from "react";

import { AuditorCompliancePanel } from "@/app/view/components/AuditorCompliancePanel";
import { ExecutiveSummaryPanel } from "@/app/view/components/ExecutiveSummaryPanel";
import { Skeleton } from "@/app/view/components/ui/skeleton";
import { EventInspector } from "@/components/dashboard/EventInspector";
import { ProfileCAS } from "@/components/dashboard/ProfileCAS";
import { StatsOverview } from "@/components/dashboard/StatsOverview";
import { TraceGraph } from "@/components/dashboard/TraceGraph";
import { TraceTimeline } from "@/components/dashboard/TraceTimeline";
import {
  flattenEventPages,
  useBundleEventsQuery,
  useBundleGraphQuery,
  useBundleMetaQuery,
  useRevealFieldMutation,
  useVerificationQuery,
} from "@/lib/api-client";
import { canViewExecutiveSummary, canViewRawData } from "@/lib/roles";
import { useUIStore } from "@/lib/state/ui-store";
import { calculateViewerHealthScore } from "@/lib/trust-score";
import { displayBundlePath } from "@/lib/display-path";

export default function ViewPage() {
  const role = useUIStore((state) => state.role);
  const [selectedSeq, setSelectedSeq] = useState<number | null>(null);
  const [pollingPaused, setPollingPaused] = useState(false);

  const verificationQuery = useVerificationQuery();
  const verification = verificationQuery.data ?? null;
  const verificationValid = verification?.status === "valid";

  const metaQuery = useBundleMetaQuery(verificationValid);
  const shouldLoadRawEventData = verificationValid && canViewRawData(role);
  const livePollingEnabled = shouldLoadRawEventData && !pollingPaused;
  const eventsQuery = useBundleEventsQuery(livePollingEnabled);
  const graphQuery = useBundleGraphQuery(livePollingEnabled);
  const revealFieldMutation = useRevealFieldMutation();

  if (verificationQuery.isError) {
    throw verificationQuery.error;
  }
  if (verificationValid && metaQuery.isError) {
    throw metaQuery.error;
  }
  if (shouldLoadRawEventData && eventsQuery.isError) {
    throw eventsQuery.error;
  }
  if (shouldLoadRawEventData && graphQuery.isError) {
    throw graphQuery.error;
  }

  const events = useMemo(
    () => flattenEventPages(eventsQuery.data?.pages),
    [eventsQuery.data?.pages],
  );
  const eventTotal = eventsQuery.data?.pages?.at(-1)?.total ?? 0;

  const activeSelectedSeq = useMemo(() => {
    if (events.length === 0) return null;
    if (selectedSeq !== null && events.some((e) => e.seq === selectedSeq)) return selectedSeq;
    return events[0].seq;
  }, [events, selectedSeq]);

  const selectedEvent = useMemo(
    () => events.find((e) => e.seq === activeSelectedSeq) ?? null,
    [activeSelectedSeq, events],
  );

  useEffect(() => {
    if (!shouldLoadRawEventData) return;
    let mounted = true;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const onScroll = () => {
      setPollingPaused(true);
      if (timer) clearTimeout(timer);
      timer = setTimeout(() => {
        if (mounted) setPollingPaused(false);
      }, 1200);
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      mounted = false;
      window.removeEventListener("scroll", onScroll);
    };
  }, [shouldLoadRawEventData]);

  const viewerHealth = useMemo(
    () => calculateViewerHealthScore({ verification, lastTimestamp: metaQuery.data?.last_timestamp }),
    [metaQuery.data?.last_timestamp, verification],
  );

  async function handleReveal(seq: number, fieldPath: string): Promise<unknown> {
    const response = await revealFieldMutation.mutateAsync({
      seq,
      field_path: fieldPath,
      reason: "dashboard_reveal",
    });
    return response.value;
  }

  return (
    <div className="flex h-full w-full overflow-hidden">
      {/* ── LEFT RAIL ─────────────────────────────────────────────────── */}
      <aside className="flex w-64 shrink-0 flex-col border-r border-border">
        <div className="border-b border-border px-3 py-2">
          <p className="truncate font-mono text-xs uppercase tracking-widest text-muted-foreground">
            ATB · {displayBundlePath(metaQuery.data?.bundle_path)}
          </p>
        </div>

        <div className="flex-1 overflow-hidden">
          {eventsQuery.isLoading && shouldLoadRawEventData ? (
            <div className="space-y-2 p-2">
              {Array.from({ length: 8 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : (
            <TraceTimeline
              events={events}
              total={eventTotal}
              hasMore={Boolean(eventsQuery.hasNextPage)}
              loadingMore={eventsQuery.isFetchingNextPage}
              selectedSeq={activeSelectedSeq}
              onSelect={setSelectedSeq}
              onLoadMore={() => void eventsQuery.fetchNextPage()}
              disabled={!verificationValid || eventsQuery.isFetching}
            />
          )}
        </div>

        <ProfileCAS className="border-t border-border" />
      </aside>

      {/* ── CENTRE COLUMN ─────────────────────────────────────────────── */}
      <main id="dashboard-content" className="flex flex-1 flex-col overflow-y-auto">
        <h1 className="sr-only">ATB Trust Dashboard</h1>

        {/* DAG graph */}
        <section className="h-[340px] shrink-0 border-b border-border bg-grid">
          {graphQuery.isLoading ? (
            <div className="flex h-full items-center justify-center">
              <Skeleton className="h-full w-full rounded-none" />
            </div>
          ) : (
            <TraceGraph
              graph={graphQuery.data ?? null}
              disabled={!verificationValid || graphQuery.isFetching}
              onSelectSeq={setSelectedSeq}
              layout="dagre-top-down"
            />
          )}
        </section>

        {/* Stats strip */}
        <section className="shrink-0 border-b border-border">
          <StatsOverview
            verification={verification}
            meta={metaQuery.data ?? null}
            viewerHealth={viewerHealth}
            role={role}
            pollingActive={shouldLoadRawEventData ? !pollingPaused : undefined}
          />
        </section>

        {/* Role-conditional panels */}
        {role === "auditor" && (
          <div className="p-4">
            <AuditorCompliancePanel
              verification={verification}
              meta={metaQuery.data ?? null}
              viewerHealth={viewerHealth}
            />
          </div>
        )}
        {canViewExecutiveSummary(role) && (
          <div className="p-4">
            <ExecutiveSummaryPanel
              verification={verification}
              meta={metaQuery.data ?? null}
              viewerHealth={viewerHealth}
            />
          </div>
        )}
      </main>

      {/* ── RIGHT RAIL ────────────────────────────────────────────────── */}
      <aside
        aria-label="Event inspector"
        className="flex w-[420px] shrink-0 flex-col overflow-y-auto border-l border-border"
      >
        <div className="border-b border-border px-3 py-2">
          <p className="font-mono text-xs uppercase tracking-widest text-muted-foreground">
            Inspector
          </p>
        </div>
        <EventInspector
          event={selectedEvent}
          disabled={!verificationValid || revealFieldMutation.isPending}
          onReveal={handleReveal}
        />
      </aside>
    </div>
  );
}
