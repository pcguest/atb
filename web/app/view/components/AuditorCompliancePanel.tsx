"use client";

import { Download } from "lucide-react";

import { Button } from "@/app/view/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/app/view/components/ui/card";
import { canExportEvidence } from "@/lib/roles";
import type { ViewerHealthBreakdown } from "@/lib/trust-score";
import type { BundleMetaResponse, VerificationResponse } from "@/lib/types";

type Props = {
  verification: VerificationResponse | null;
  meta: BundleMetaResponse | null;
  viewerHealth: ViewerHealthBreakdown;
};

function downloadReviewSummary(payload: Record<string, unknown>): void {
  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `atb-review-summary-${Date.now()}.json`;
  link.click();
  URL.revokeObjectURL(url);
}

export function AuditorCompliancePanel({ verification, meta, viewerHealth }: Props) {
  const reviewSummaryEnabled = canExportEvidence("auditor");

  function handleDownload(): void {
    if (!verification || !meta) return;
    downloadReviewSummary({
      generated_at: new Date().toISOString(),
      viewer_health: viewerHealth.total,
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
          <p className="text-xs uppercase tracking-wide text-muted-foreground">Viewer Health</p>
          <p className="mt-1 text-sm font-medium text-foreground">{viewerHealth.total} / 100</p>
          <p className="mt-1 text-[11px] leading-snug text-muted-foreground">
            Composite of verification and recency. Not the integrity result.
          </p>
        </div>
        <div className="md:col-span-3">
          <Button
            type="button"
            variant="outline"
            data-testid="export-evidence-btn"
            className="inline-flex items-center gap-2"
            onClick={handleDownload}
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
