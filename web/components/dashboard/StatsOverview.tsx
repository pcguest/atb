"use client";

import type { DashboardRole } from "@/lib/roles";
import { dashboardRoleLabel } from "@/lib/roles";
import type { ViewerHealthBreakdown } from "@/lib/trust-score";
import type { BundleMetaResponse, VerificationResponse } from "@/lib/types";

type StatsOverviewProps = {
  verification: VerificationResponse | null;
  meta: BundleMetaResponse | null;
  viewerHealth: ViewerHealthBreakdown;
  role: DashboardRole;
  pollingActive?: boolean;
};

function Stat({
  label,
  value,
  testId,
}: {
  label: string;
  value: React.ReactNode;
  testId?: string;
}) {
  return (
    <div className="flex flex-col gap-0.5 px-4 py-2">
      <span className="font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
        {label}
      </span>
      <span className="font-mono text-sm font-semibold text-foreground" data-testid={testId}>
        {value}
      </span>
    </div>
  );
}

function eventFamily(eventType: string): string {
  if (eventType.startsWith("ai.llm")) return "llm";
  if (eventType.startsWith("ai.tool")) return "tool";
  if (eventType.startsWith("ai.chain")) return "chain";
  if (eventType.startsWith("ai.policy")) return "policy";
  if (eventType.startsWith("ai.action")) return "action";
  if (eventType.startsWith("ai.job")) return "job";
  if (eventType.startsWith("atb.corroboration")) return "corroboration";
  return "other";
}

function eventFamilyCounts(meta: BundleMetaResponse | null): string {
  if (!meta || Object.keys(meta.type_counts).length === 0) {
    return "—";
  }

  const counts = new Map<string, number>();
  for (const [eventType, count] of Object.entries(meta.type_counts)) {
    const family = eventFamily(eventType);
    counts.set(family, (counts.get(family) ?? 0) + count);
  }

  return Array.from(counts.entries())
    .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
    .map(([family, count]) => `${family} ${count}`)
    .join(" · ");
}

export function StatsOverview({
  verification,
  meta,
  viewerHealth,
  role,
  pollingActive,
}: StatsOverviewProps) {
  return (
    <div
      className="flex flex-wrap items-center divide-x divide-border"
      data-testid="stats-overview"
    >
      <Stat label="events" value={meta?.event_count ?? "—"} testId="event-count-value" />
      <Stat
        label="families"
        value={eventFamilyCounts(meta)}
        testId="event-family-counts-value"
      />
      <Stat
        label="verification"
        value={verification?.status ?? "—"}
        testId="verification-status-value"
      />
      <Stat label="chain" value={verification?.chain_length ?? "—"} testId="chain-length-value" />
      <Stat label="health" value={`${viewerHealth.total}/100`} testId="viewer-health-value" />
      <Stat label="role" value={dashboardRoleLabel[role]} testId="role-value" />
      {pollingActive !== undefined && (
        <div className="flex flex-col gap-0.5 px-4 py-2">
          <span className="font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
            polling
          </span>
          <span
            aria-live="polite"
            aria-label={pollingActive ? "Live polling active" : "Live polling paused"}
            data-testid="polling-status-value"
            className={`font-mono text-sm font-semibold ${
              pollingActive ? "text-red-400" : "text-muted-foreground"
            }`}
          >
            {pollingActive ? "● live" : "paused"}
          </span>
        </div>
      )}
    </div>
  );
}
