"use client";

import AnomalyBadge from "@/components/AnomalyBadge";
import { useSessionsQuery } from "@/lib/api-client";
import type { SessionEntry } from "@/lib/types";

const ANOMALY_LABELS: Record<string, string> = {
  tool_without_approval: "No matching earlier approval in captured evidence",
  policy_denied_executed: "Policy denied but action executed",
  action_failed: "Privileged action failed",
  unresolved_identity: "Unresolved identity",
  session_not_closed: "Session not closed",
  load_error: "Bundle load error",
};

export interface AnomalySummary {
  flag: string;
  label: string;
  sessions: number;
}

/**
 * summariseAnomalies counts how many of the bundle's sessions carry each
 * distinct anomaly flag, most-frequent first. Duplicate flags within one
 * session are counted once.
 */
export function summariseAnomalies(sessions: SessionEntry[]): AnomalySummary[] {
  const counts = new Map<string, number>();
  for (const session of sessions) {
    for (const flag of new Set(session.anomaly_flags ?? [])) {
      counts.set(flag, (counts.get(flag) ?? 0) + 1);
    }
  }
  return Array.from(counts.entries())
    .map(([flag, sessions]) => ({ flag, label: ANOMALY_LABELS[flag] ?? flag, sessions }))
    .sort((a, b) => b.sessions - a.sessions || a.flag.localeCompare(b.flag));
}

/** Presentational banner; renders nothing when there are no anomalies. */
export function SessionAnomaliesBanner({ sessions }: { sessions: SessionEntry[] }) {
  const summary = summariseAnomalies(sessions);
  if (summary.length === 0) {
    return null;
  }
  return (
    <div
      data-testid="session-anomalies"
      role="status"
      className="flex flex-wrap items-center gap-x-4 gap-y-1 border-b border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-200"
    >
      <span className="font-medium uppercase tracking-wide text-amber-300">
        Oversight anomalies
      </span>
      {summary.map((item) => (
        <span key={item.flag} className="inline-flex items-center gap-1">
          <AnomalyBadge flag={item.flag} />
          <span>{item.label}</span>
          <span className="text-amber-400/80">
            ({item.sessions} session{item.sessions === 1 ? "" : "s"})
          </span>
        </span>
      ))}
    </div>
  );
}

/** Container: fetches the bundle's sessions and surfaces their anomaly flags. */
export function SessionAnomalies({ enabled = true }: { enabled?: boolean }) {
  const { data } = useSessionsQuery(enabled);
  if (!data) {
    return null;
  }
  return <SessionAnomaliesBanner sessions={data} />;
}
