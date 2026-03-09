"use client";

import type { BundleMetaResponse } from "@/lib/types";

type StatsOverviewProps = {
  meta: BundleMetaResponse | null;
};

export function StatsOverview({ meta }: StatsOverviewProps) {
  if (!meta) {
    return null;
  }

  const eventTypes = Object.keys(meta.type_counts ?? {}).length;

  return (
    <div className="grid gap-3 sm:grid-cols-3">
      <div className="rounded-lg border border-slate-800 bg-slate-900 p-3">
        <div className="text-xs uppercase tracking-wide text-slate-400">Events</div>
        <div className="mt-1 text-xl font-semibold text-slate-100">{meta.event_count}</div>
      </div>
      <div className="rounded-lg border border-slate-800 bg-slate-900 p-3">
        <div className="text-xs uppercase tracking-wide text-slate-400">Event Types</div>
        <div className="mt-1 text-xl font-semibold text-slate-100">{eventTypes}</div>
      </div>
      <div className="rounded-lg border border-slate-800 bg-slate-900 p-3">
        <div className="text-xs uppercase tracking-wide text-slate-400">Verification</div>
        <div className="mt-1 text-xl font-semibold text-slate-100">
          {meta.verified ? "Verified" : "Invalid"}
        </div>
      </div>
    </div>
  );
}
