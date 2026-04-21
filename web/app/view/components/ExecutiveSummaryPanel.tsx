"use client";

import { TrendingUp } from "lucide-react";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/app/view/components/ui/card";
import type { TrustScoreBreakdown } from "@/lib/trust-score";
import type { BundleMetaResponse, VerificationResponse } from "@/lib/types";

type Props = {
  verification: VerificationResponse | null;
  meta: BundleMetaResponse | null;
  trustScore: TrustScoreBreakdown;
};

export function ExecutiveSummaryPanel({ verification, meta, trustScore }: Props) {
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
              ? topEventTypes.map(([type, count]) => `${type} (${count})`).join(", ")
              : "No event aggregates available."}
          </p>
        </div>
      </CardContent>
    </Card>
  );
}
