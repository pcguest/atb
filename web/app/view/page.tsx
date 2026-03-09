"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { getBundleEvents, getBundleGraph, getBundleMeta, getVerification, revealField } from "@/lib/api-client";
import { EventInspector } from "@/components/dashboard/EventInspector";
import { StatsOverview } from "@/components/dashboard/StatsOverview";
import { TraceGraph } from "@/components/dashboard/TraceGraph";
import { TraceTimeline } from "@/components/dashboard/TraceTimeline";
import { VerificationBanner } from "@/components/dashboard/VerificationBanner";
import type { BundleGraphResponse, BundleMetaResponse, EventRecord, VerificationResponse } from "@/lib/types";

const eventsPageSize = 200;
const connectivityPollMs = 5000;

function toMessage(err: unknown, fallback: string): string {
  if (err instanceof Error && err.message.trim() !== "") {
    return err.message;
  }
  return fallback;
}

export default function ViewPage() {
  const [verification, setVerification] = useState<VerificationResponse | null>(null);
  const [verificationLoading, setVerificationLoading] = useState(true);
  const [verificationError, setVerificationError] = useState<string | null>(null);

  const [meta, setMeta] = useState<BundleMetaResponse | null>(null);
  const [events, setEvents] = useState<EventRecord[]>([]);
  const [eventTotal, setEventTotal] = useState(0);
  const [graph, setGraph] = useState<BundleGraphResponse | null>(null);
  const [dataLoading, setDataLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [dataError, setDataError] = useState<string | null>(null);

  const [selectedSeq, setSelectedSeq] = useState<number | null>(null);
  const verificationStatusRef = useRef<string>("unknown");

  const clearData = useCallback(() => {
    setMeta(null);
    setEvents([]);
    setEventTotal(0);
    setGraph(null);
    setSelectedSeq(null);
  }, []);

  const loadBundleData = useCallback(async () => {
    setDataLoading(true);
    setDataError(null);
    try {
      const [metaResp, eventsResp, graphResp] = await Promise.all([
        getBundleMeta(),
        getBundleEvents(0, eventsPageSize),
        getBundleGraph(),
      ]);
      setMeta(metaResp);
      setEvents(eventsResp.events);
      setEventTotal(eventsResp.total);
      setGraph(graphResp);
      setSelectedSeq((current) => current ?? eventsResp.events[0]?.seq ?? null);
    } catch (err) {
      setDataError(toMessage(err, "Failed to load dashboard data"));
    } finally {
      setDataLoading(false);
    }
  }, []);

  const loadVerification = useCallback(async () => {
    setVerificationLoading(true);
    try {
      const verificationResp = await getVerification();
      setVerification(verificationResp);
      verificationStatusRef.current = verificationResp.status;
      setVerificationError(null);
      if (verificationResp.status === "valid") {
        await loadBundleData();
      } else {
        clearData();
      }
    } catch (err) {
      clearData();
      verificationStatusRef.current = "unknown";
      setVerificationError(toMessage(err, "Connection lost to local viewer server"));
    } finally {
      setVerificationLoading(false);
    }
  }, [clearData, loadBundleData]);

  const loadMoreEvents = useCallback(async () => {
    if (dataLoading || loadingMore) {
      return;
    }
    if (verification?.status !== "valid") {
      return;
    }
    if (events.length >= eventTotal) {
      return;
    }

    setLoadingMore(true);
    try {
      const nextPage = await getBundleEvents(events.length, eventsPageSize);
      setEventTotal(nextPage.total);
      setEvents((current) => {
        const seen = new Set(current.map((event) => event.seq));
        const appended = nextPage.events.filter((event) => !seen.has(event.seq));
        return current.concat(appended);
      });
      setSelectedSeq((current) => current ?? nextPage.events[0]?.seq ?? null);
    } catch (err) {
      setDataError(toMessage(err, "Failed to load more events"));
    } finally {
      setLoadingMore(false);
    }
  }, [dataLoading, eventTotal, events.length, loadingMore, verification?.status]);

  useEffect(() => {
    void loadVerification();
  }, [loadVerification]);

  useEffect(() => {
    const interval = window.setInterval(async () => {
      try {
        const nextVerification = await getVerification();
        setVerification(nextVerification);
        const previousStatus = verificationStatusRef.current;
        verificationStatusRef.current = nextVerification.status;
        setVerificationError(null);

        if (nextVerification.status !== "valid") {
          clearData();
          return;
        }

        if (previousStatus !== "valid") {
          await loadBundleData();
        }
      } catch (err) {
        verificationStatusRef.current = "unknown";
        setVerificationError(toMessage(err, "Connection lost to local viewer server"));
      }
    }, connectivityPollMs);

    return () => {
      window.clearInterval(interval);
    };
  }, [clearData, loadBundleData]);

  const selectedEvent = useMemo(() => {
    if (selectedSeq === null) {
      return null;
    }
    return events.find((event) => event.seq === selectedSeq) ?? null;
  }, [events, selectedSeq]);

  const hasMoreEvents = events.length < eventTotal;
  const interactionsDisabled =
    verificationLoading || verificationError !== null || verification?.status !== "valid";

  async function handleReveal(seq: number, fieldPath: string): Promise<unknown> {
    const response = await revealField({
      seq,
      field_path: fieldPath,
      reason: "dashboard_reveal",
    });
    return response.value;
  }

  return (
    <main className="min-h-screen bg-slate-950 px-4 py-5 text-slate-100 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-7xl space-y-4">
        <h1 className="text-xl font-semibold tracking-tight sm:text-2xl">ATB Viewer</h1>

        <VerificationBanner
          verification={verification}
          loading={verificationLoading}
          error={verificationError}
        />

        {verification?.status !== "valid" && !verificationLoading && (
          <div className="rounded-lg border border-red-500/40 bg-red-950/40 p-4 text-sm text-red-200">
            Interaction is disabled until the bundle hash chain is valid.
          </div>
        )}

        {dataError && (
          <div className="rounded-lg border border-amber-500/40 bg-amber-950/30 p-4 text-sm text-amber-200">
            {dataError}
          </div>
        )}

        <StatsOverview meta={meta} />

        <section className="grid gap-4 lg:grid-cols-12">
          <div className="lg:col-span-4">
            <TraceTimeline
              events={events}
              total={eventTotal}
              hasMore={hasMoreEvents}
              loadingMore={loadingMore}
              selectedSeq={selectedSeq}
              onSelect={setSelectedSeq}
              onLoadMore={loadMoreEvents}
              disabled={interactionsDisabled || dataLoading}
            />
          </div>
          <div className="lg:col-span-8">
            <EventInspector
              event={selectedEvent}
              disabled={interactionsDisabled || dataLoading}
              onReveal={handleReveal}
            />
          </div>
        </section>

        <section>
          <TraceGraph
            graph={graph}
            disabled={interactionsDisabled || dataLoading}
            onSelectSeq={setSelectedSeq}
          />
        </section>
      </div>
    </main>
  );
}
