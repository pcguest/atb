"use client";

import { Download, ShieldAlert, TrendingUp } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { BundleMetaPanel } from "@/app/view/components/bundle-meta/BundleMetaPanel";
import { TrustScoreCard } from "@/app/view/components/trust-score/TrustScoreCard";
import { Button } from "@/app/view/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/app/view/components/ui/card";
import { Skeleton } from "@/app/view/components/ui/skeleton";
import { EventInspector } from "@/components/dashboard/EventInspector";
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
import {
  canExportEvidence,
  canViewExecutiveSummary,
  canViewRawData,
  dashboardRoleLabel,
} from "@/lib/roles";
import { useUIStore } from "@/lib/state/ui-store";
import { calculateTrustScore, type TrustScoreBreakdown } from "@/lib/trust-score";
import type { BundleMetaResponse, VerificationResponse } from "@/lib/types";

type RoleInsightsProps = {
  verification: VerificationResponse | null;
  meta: BundleMetaResponse | null;
  trustScore: TrustScoreBreakdown;
};

function downloadReviewSummary(payload: Record<string, unknown>): void {
  const evidenceBlob = new Blob([JSON.stringify(payload, null, 2)], {
    type: "application/json",
  });
  const url = URL.createObjectURL(evidenceBlob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `atb-review-summary-${Date.now()}.json`;
  link.click();
  URL.revokeObjectURL(url);
}

function AuditorCompliancePanel({ verification, meta, trustScore }: RoleInsightsProps) {
  const reviewSummaryEnabled = canExportEvidence("auditor");

  function handleDownloadReviewSummary(): void {
    if (!verification || !meta) {
      return;
    }

    downloadReviewSummary({
      generated_at: new Date().toISOString(),
      trust_score: trustScore.total,
      verification_status: verification.status,
      chain_length: verification.chain_length,
      head_hash: verification.head_hash ?? null,
      genesis_hash: meta.genesis_hash ?? null,
      verified_at: meta.verified_at ?? null,
      event_count: meta.event_count,
      type_counts: meta.type_counts,
      bundle_path: meta.bundle_path,
    });
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Auditor Workspace</CardTitle>
        <CardDescription>
          Review summary mode is active. Raw payload and event-debug sections are hidden.
        </CardDescription>
      </CardHeader>
      <CardContent className="grid gap-3 md:grid-cols-3">
        <div className="rounded-md border border-border bg-background/70 p-3">
          <p className="text-xs uppercase tracking-wide text-muted-foreground">Verification</p>
          <p className="mt-1 text-sm font-medium capitalize text-foreground">
            {verification?.status ?? "unknown"}
          </p>
        </div>
        <div className="rounded-md border border-border bg-background/70 p-3">
          <p className="text-xs uppercase tracking-wide text-muted-foreground">Event Count</p>
          <p className="mt-1 text-sm font-medium text-foreground">{meta?.event_count ?? 0}</p>
        </div>
        <div className="rounded-md border border-border bg-background/70 p-3">
          <p className="text-xs uppercase tracking-wide text-muted-foreground">Trust Score</p>
          <p className="mt-1 text-sm font-medium text-foreground">{trustScore.total}</p>
        </div>
        <div className="md:col-span-3">
          <Button
            type="button"
            variant="outline"
            data-testid="export-evidence-btn"
            className="inline-flex items-center gap-2"
            onClick={handleDownloadReviewSummary}
            disabled={!reviewSummaryEnabled || !verification || !meta}
            aria-label="Download review summary"
          >
            <Download className="h-4 w-4" aria-hidden="true" />
            Download review summary
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

function ExecutiveSummaryPanel({ verification, meta, trustScore }: RoleInsightsProps) {
  const topEventTypes = Object.entries(meta?.type_counts ?? {})
    .sort((a, b) => b[1] - a[1])
    .slice(0, 3);

  return (
    <Card data-testid="executive-summary">
      <CardHeader>
        <CardTitle>Executive Summary</CardTitle>
        <CardDescription>
          High-level trust posture view. Detailed payload inspection is intentionally hidden.
        </CardDescription>
      </CardHeader>
      <CardContent className="grid gap-3 md:grid-cols-3">
        <div className="rounded-md border border-border bg-background/70 p-3">
          <p className="text-xs uppercase tracking-wide text-muted-foreground">Trust Posture</p>
          <p className="mt-1 text-sm font-medium text-foreground">{trustScore.total} / 100</p>
        </div>
        <div className="rounded-md border border-border bg-background/70 p-3 md:col-span-2">
          <p className="text-xs uppercase tracking-wide text-muted-foreground">Trend Summary</p>
          <p className="mt-1 inline-flex items-center gap-1 text-sm text-foreground">
            <TrendingUp className="h-4 w-4 text-primary" aria-hidden="true" />
            {verification?.status === "valid"
              ? "Integrity checks passing and telemetry is stable."
              : "Integrity checks are failing or pending verification."}
          </p>
        </div>
        <div className="rounded-md border border-border bg-background/70 p-3 md:col-span-3">
          <p className="text-xs uppercase tracking-wide text-muted-foreground">Top Event Types</p>
          <p className="mt-1 text-sm text-foreground">
            {topEventTypes.length > 0
              ? topEventTypes.map(([eventType, count]) => `${eventType} (${count})`).join(", ")
              : "No event aggregates available."}
          </p>
        </div>
      </CardContent>
    </Card>
  );
}

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
    if (events.length === 0) {
      return null;
    }
    if (selectedSeq !== null && events.some((event) => event.seq === selectedSeq)) {
      return selectedSeq;
    }
    return events[0].seq;
  }, [events, selectedSeq]);
  const selectedEvent = useMemo(
    () => events.find((event) => event.seq === activeSelectedSeq) ?? null,
    [activeSelectedSeq, events],
  );

  useEffect(() => {
    let mounted = true;

    if (!shouldLoadRawEventData) {
      return;
    }

    let timer: ReturnType<typeof setTimeout> | undefined;
    const onScroll = () => {
      setPollingPaused(true);
      if (timer) {
        clearTimeout(timer);
      }
      timer = setTimeout(() => {
        if (mounted) {
          setPollingPaused(false);
        }
      }, 1200);
    };

    window.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      mounted = false;
      window.removeEventListener("scroll", onScroll);
    };
  }, [shouldLoadRawEventData]);

  const trustScore = useMemo(
    () =>
      calculateTrustScore({
        verification,
        lastTimestamp: metaQuery.data?.last_timestamp,
      }),
    [metaQuery.data?.last_timestamp, verification],
  );

  const shellLoading = verificationQuery.isLoading || (verificationValid && metaQuery.isLoading);
  const roleLabel = dashboardRoleLabel[role];

  async function handleReveal(seq: number, fieldPath: string): Promise<unknown> {
    const response = await revealFieldMutation.mutateAsync({
      seq,
      field_path: fieldPath,
      reason: "dashboard_reveal",
    });
    return response.value;
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-lg">Operational Status</CardTitle>
          <CardDescription>
            Role-based rendering and trust telemetry for ATB v1.1.0.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col items-start justify-between gap-3 sm:flex-row sm:items-center">
          <div>
            <p className="text-xs uppercase tracking-wide text-muted-foreground">Active Role</p>
            <p className="text-base font-semibold">{roleLabel}</p>
          </div>
          <div className="flex flex-col items-start gap-2 sm:items-end">
            <p className="text-sm text-muted-foreground">
              Verification:{" "}
              <span className="font-medium text-foreground">
                {verification?.status ?? "Checking..."}
              </span>
            </p>
            {shouldLoadRawEventData && (
              <p
                className="text-xs font-medium text-red-300"
                aria-live="polite"
                aria-label={pollingPaused ? "Live polling paused" : "Live polling active"}
              >
                {pollingPaused ? "Polling paused" : "🔴 Live"}
              </p>
            )}
            <Button
              type="button"
              variant="outline"
              onClick={() => void verificationQuery.refetch()}
            >
              Verify bundle first
            </Button>
          </div>
        </CardContent>
      </Card>

      {!verificationValid && !verificationQuery.isLoading && (
        <Card className="border-red-500/40 bg-red-500/10">
          <CardContent className="flex items-center gap-3 py-5 text-sm text-red-100">
            <ShieldAlert className="h-5 w-5 shrink-0 text-red-300" aria-hidden="true" />
            Interaction is limited until the bundle hash chain validates successfully.
          </CardContent>
        </Card>
      )}

      <section className="grid gap-4 xl:grid-cols-5">
        <div className="xl:col-span-2">
          <TrustScoreCard loading={shellLoading} breakdown={shellLoading ? null : trustScore} />
        </div>
        <div className="xl:col-span-3">
          <BundleMetaPanel
            loading={shellLoading}
            verification={verification}
            meta={metaQuery.data ?? null}
            chainLength={verification?.chain_length ?? 0}
          />
        </div>
      </section>

      {role === "auditor" && (
        <AuditorCompliancePanel
          verification={verification}
          meta={metaQuery.data ?? null}
          trustScore={trustScore}
        />
      )}

      {canViewExecutiveSummary(role) && (
        <ExecutiveSummaryPanel
          verification={verification}
          meta={metaQuery.data ?? null}
          trustScore={trustScore}
        />
      )}

      {role === "engineer" && (
        <>
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>Engineer Debug Context</CardTitle>
              <CardDescription>
                Raw event payload inspection and graph debugging are enabled in this role.
              </CardDescription>
            </CardHeader>
            <CardContent className="grid gap-3 md:grid-cols-3">
              <div className="rounded-md border border-border bg-background/70 p-3">
                <p className="text-xs uppercase tracking-wide text-muted-foreground">
                  Selected Seq
                </p>
                <p className="mt-1 text-sm font-medium text-foreground">
                  {activeSelectedSeq ?? "None"}
                </p>
              </div>
              <div className="rounded-md border border-border bg-background/70 p-3">
                <p className="text-xs uppercase tracking-wide text-muted-foreground">
                  Loaded Events
                </p>
                <p className="mt-1 text-sm font-medium text-foreground">{events.length}</p>
              </div>
              <div className="rounded-md border border-border bg-background/70 p-3">
                <p className="text-xs uppercase tracking-wide text-muted-foreground">Reveal API</p>
                <p className="mt-1 text-sm font-medium text-foreground">
                  {revealFieldMutation.isPending ? "Pending" : "Idle"}
                </p>
              </div>
            </CardContent>
          </Card>

          <section className="grid gap-4 lg:grid-cols-12">
            <div className="lg:col-span-4">
              {eventsQuery.isLoading ? (
                <Card>
                  <CardHeader>
                    <CardTitle>Timeline</CardTitle>
                    <CardDescription>Loading event pages.</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    {Array.from({ length: 7 }).map((_, index) => (
                      <Skeleton key={index} className="h-12 w-full" />
                    ))}
                  </CardContent>
                </Card>
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
            <div className="lg:col-span-8">
              {eventsQuery.isLoading ? (
                <Card data-testid="raw-data-inspector">
                  <CardHeader>
                    <CardTitle>Event Inspector</CardTitle>
                    <CardDescription>Loading selected event payload.</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    <Skeleton className="h-4 w-1/3" />
                    <Skeleton className="h-4 w-2/3" />
                    <Skeleton className="h-56 w-full" />
                  </CardContent>
                </Card>
              ) : (
                <div data-testid="raw-data-inspector">
                  <EventInspector
                    event={selectedEvent}
                    disabled={!verificationValid || revealFieldMutation.isPending}
                    onReveal={handleReveal}
                  />
                </div>
              )}
            </div>
          </section>

          <section>
            {graphQuery.isLoading ? (
              <Card>
                <CardHeader>
                  <CardTitle>Trace Graph</CardTitle>
                  <CardDescription>
                    Hydrating relationship graph from verified data.
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <Skeleton className="h-[320px] w-full" />
                </CardContent>
              </Card>
            ) : (
              <TraceGraph
                graph={graphQuery.data ?? null}
                disabled={!verificationValid || graphQuery.isFetching}
                onSelectSeq={setSelectedSeq}
              />
            )}
          </section>
        </>
      )}
    </div>
  );
}
