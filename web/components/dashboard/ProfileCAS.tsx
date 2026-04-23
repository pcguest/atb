"use client";

import { ChevronDown, ChevronRight, RotateCcw, ShieldOff } from "lucide-react";
import { useState } from "react";

import { useBundleProfileQuery, useRunBundleVerifyMutation } from "@/lib/api-client";
import type { FailureDTO, ProfileReportSummary } from "@/lib/types";

// ─── Grade chip colours ───────────────────────────────────────────────────────

const GRADE_STYLE: Record<string, string> = {
  High: "bg-green-900/60 text-green-300 border-green-700/40",
  Medium: "bg-teal-900/60 text-teal-300 border-teal-700/40",
  Low: "bg-yellow-900/60 text-yellow-300 border-yellow-700/40",
  Insufficient: "bg-red-900/60 text-red-300 border-red-700/40",
};

// ─── Sub-score bar ────────────────────────────────────────────────────────────

function SubScoreBar({ label, value }: { label: string; value: number }) {
  const pct = Math.round(value * 100);
  return (
    <div className="flex items-center gap-2">
      <span className="w-6 shrink-0 font-mono text-[10px] uppercase text-muted-foreground">
        {label}
      </span>
      <div className="flex-1 overflow-hidden rounded-full bg-muted" style={{ height: 4 }}>
        <div
          className="h-full rounded-full bg-primary/40 transition-all"
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="w-8 shrink-0 text-right font-mono text-[10px] text-muted-foreground">
        {pct}%
      </span>
    </div>
  );
}

// ─── Failure / warning card ───────────────────────────────────────────────────

function FailureCard({
  failure,
  variant,
}: {
  failure: FailureDTO;
  variant: "critical" | "warning";
}) {
  const styles =
    variant === "critical"
      ? "border-red-800/30 bg-red-950/40"
      : "border-amber-800/30 bg-amber-950/30";

  return (
    <li className={`rounded border px-3 py-1.5 ${styles}`}>
      <span className="font-mono text-xs font-semibold text-red-300">
        {variant === "critical" ? failure.kind : ""}
      </span>
      {variant === "critical" && <span className="mx-1 text-xs text-muted-foreground">—</span>}
      <span className="text-xs text-muted-foreground">{failure.detail}</span>
    </li>
  );
}

// ─── Full report body ─────────────────────────────────────────────────────────

function ReportBody({ report }: { report: ProfileReportSummary }) {
  const [warningsOpen, setWarningsOpen] = useState(false);
  const casPercent = Math.round(report.cas_score * 100);
  const effectivePct = report.effective_score
    ? Math.round(report.effective_score * 100)
    : undefined;
  const bonusPct =
    report.corroboration_bonus && report.corroboration_bonus > 0
      ? Math.round(report.corroboration_bonus * 100)
      : undefined;

  const subEntries = Object.entries(report.sub_scores) as [string, number][];

  return (
    <div className="space-y-3 pb-2">
      {/* CAS score bar */}
      <div className="space-y-1">
        <div className="flex items-center justify-between">
          <span className="font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
            completeness (CAS)
          </span>
          <span className="font-mono text-xs text-foreground">{casPercent}%</span>
        </div>
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
          <div
            className="h-full rounded-full bg-primary/70 transition-all"
            style={{ width: `${casPercent}%` }}
          />
        </div>
        {effectivePct !== undefined && bonusPct !== undefined && (
          <p className="font-mono text-[10px] text-muted-foreground">
            effective {effectivePct}% (with corroboration +{bonusPct}%)
          </p>
        )}
      </div>

      {/* Sub-score grid — 2 columns */}
      <div className="grid grid-cols-2 gap-x-4 gap-y-1.5">
        {subEntries.map(([key, val]) => (
          <SubScoreBar key={key} label={key.toUpperCase()} value={val} />
        ))}
      </div>

      {/* Critical failures */}
      {report.critical_failures.length > 0 && (
        <div className="space-y-1">
          <p className="font-mono text-[10px] uppercase tracking-widest text-red-400">
            critical failures ({report.critical_failures.length})
          </p>
          <ul className="space-y-1">
            {report.critical_failures.map((f, i) => (
              <FailureCard key={i} failure={f} variant="critical" />
            ))}
          </ul>
        </div>
      )}

      {/* Warnings — collapsible */}
      {report.warnings.length > 0 && (
        <div>
          <button
            type="button"
            onClick={() => setWarningsOpen((v) => !v)}
            aria-expanded={warningsOpen}
            className="flex items-center gap-1 font-mono text-[10px] uppercase tracking-widest text-amber-500 hover:text-amber-400"
          >
            {warningsOpen ? (
              <ChevronDown className="h-3 w-3" aria-hidden="true" />
            ) : (
              <ChevronRight className="h-3 w-3" aria-hidden="true" />
            )}
            {report.warnings.length} warning{report.warnings.length !== 1 ? "s" : ""}
          </button>
          {warningsOpen && (
            <ul className="mt-1.5 space-y-1">
              {report.warnings.map((w, i) => (
                <li
                  key={i}
                  className="rounded border border-amber-800/30 bg-amber-950/30 px-3 py-1.5"
                >
                  <span className="text-xs text-muted-foreground">{w}</span>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}

// ─── Main component ───────────────────────────────────────────────────────────

type ProfileCASProps = {
  className?: string;
};

export function ProfileCAS({ className }: ProfileCASProps) {
  const [open, setOpen] = useState(false);
  const profileQuery = useBundleProfileQuery(true);
  const verifyMutation = useRunBundleVerifyMutation();

  const isForbidden =
    profileQuery.isError &&
    (profileQuery.error as Error & { status?: number }).status === 403;

  const report = profileQuery.data ?? null;

  return (
    <div className={`shrink-0 ${className ?? ""}`}>
      {/* Header row — always visible, toggles body */}
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="flex w-full items-center justify-between px-3 py-2"
      >
        <div className="flex min-w-0 items-center gap-2">
          <span className="font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
            Profile &amp; CAS
          </span>
          {report && (
            <>
              <span className="max-w-[10ch] truncate font-mono text-[10px] text-foreground/70">
                {report.profile_id}
              </span>
              <span
                className={`rounded border px-1.5 py-0.5 font-mono text-[10px] font-bold ${
                  report.pass
                    ? "border-green-700/40 bg-green-900/60 text-green-300"
                    : "border-red-700/40 bg-red-900/60 text-red-300"
                }`}
              >
                {report.pass ? "PASS" : "FAIL"}
              </span>
              <span
                className={`rounded border px-1.5 py-0.5 font-mono text-[10px] font-bold ${
                  GRADE_STYLE[report.cas_grade] ?? "text-muted-foreground"
                }`}
              >
                {report.cas_grade}
              </span>
            </>
          )}
        </div>
        {open ? (
          <ChevronDown className="h-3.5 w-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
        )}
      </button>

      {/* Collapsible body */}
      {open && (
        <div className="border-t border-border px-3 pt-3">
          {/* Forbidden / tamper */}
          {isForbidden && (
            <div className="flex items-center gap-2 py-2 text-xs text-red-400">
              <ShieldOff className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
              <span className="font-mono">
                Verification required before profile report is available.
              </span>
            </div>
          )}

          {/* Loading */}
          {profileQuery.isLoading && (
            <div className="space-y-2 py-2">
              {Array.from({ length: 4 }).map((_, i) => (
                <div key={i} className="h-2.5 animate-pulse rounded bg-muted" />
              ))}
            </div>
          )}

          {/* No report yet */}
          {!profileQuery.isLoading && !isForbidden && report === null && (
            <div className="flex flex-col gap-2 py-2">
              <span className="font-mono text-xs text-muted-foreground">
                No profile report — run verification first.
              </span>
              <button
                type="button"
                disabled={verifyMutation.isPending}
                onClick={() => verifyMutation.mutate()}
                className="flex w-fit items-center gap-1.5 rounded border border-border bg-card px-2.5 py-1 font-mono text-xs text-foreground hover:border-ring disabled:cursor-not-allowed disabled:opacity-50"
              >
                <RotateCcw
                  className={`h-3 w-3 ${verifyMutation.isPending ? "animate-spin" : ""}`}
                  aria-hidden="true"
                />
                {verifyMutation.isPending ? "Running…" : "Run Verify"}
              </button>
              {verifyMutation.isError && (
                <span className="font-mono text-xs text-destructive">
                  {verifyMutation.error.message}
                </span>
              )}
            </div>
          )}

          {/* Report present */}
          {!profileQuery.isLoading && !isForbidden && report !== null && (
            <>
              <ReportBody report={report} />
              <div className="border-t border-border pt-2">
                <button
                  type="button"
                  disabled={verifyMutation.isPending}
                  onClick={() => verifyMutation.mutate()}
                  aria-label="Re-run bundle verification"
                  className="flex items-center gap-1.5 rounded border border-border bg-card px-2.5 py-1 font-mono text-xs text-muted-foreground hover:border-ring hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <RotateCcw
                    className={`h-3 w-3 ${verifyMutation.isPending ? "animate-spin" : ""}`}
                    aria-hidden="true"
                  />
                  {verifyMutation.isPending ? "Running…" : "Re-run Verify"}
                </button>
                {verifyMutation.isError && (
                  <span className="mt-1 block font-mono text-xs text-destructive">
                    {verifyMutation.error.message}
                  </span>
                )}
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}
