"use client";

import type { DashboardRole } from "@/lib/roles";
import { dashboardRoleLabel } from "@/lib/roles";
import type { TrustScoreBreakdown } from "@/lib/trust-score";
import type { BundleMetaResponse, VerificationResponse } from "@/lib/types";

type StatsOverviewProps = {
  verification: VerificationResponse | null;
  meta: BundleMetaResponse | null;
  trustScore: TrustScoreBreakdown;
  role: DashboardRole;
  pollingActive?: boolean;
};

function Stat({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5 px-4 py-2">
      <span className="font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
        {label}
      </span>
      <span className="font-mono text-sm font-semibold text-foreground">{value}</span>
    </div>
  );
}

export function StatsOverview({
  verification,
  meta,
  trustScore,
  role,
  pollingActive,
}: StatsOverviewProps) {
  return (
    <div className="flex flex-wrap items-center divide-x divide-border">
      <Stat label="events" value={meta?.event_count ?? "—"} />
      <Stat label="chain" value={verification?.chain_length ?? "—"} />
      <Stat label="trust" value={`${trustScore.total}/100`} />
      <Stat label="role" value={dashboardRoleLabel[role]} />
      {pollingActive !== undefined && (
        <div className="flex flex-col gap-0.5 px-4 py-2">
          <span className="font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
            polling
          </span>
          <span
            aria-live="polite"
            aria-label={pollingActive ? "Live polling active" : "Live polling paused"}
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
