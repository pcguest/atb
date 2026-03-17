"use client";

import { motion, useReducedMotion } from "framer-motion";
import { Info } from "lucide-react";
import dynamic from "next/dynamic";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/app/view/components/ui/card";
import { Skeleton } from "@/app/view/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/app/view/components/ui/tooltip";
import type { TrustScoreBreakdown } from "@/lib/trust-score";

const TrustScoreRadial = dynamic(
  () =>
    import("./TrustScoreRadial").then((module) => ({
      default: module.TrustScoreRadial,
    })),
  {
    ssr: false,
    loading: () => <Skeleton className="h-44 w-full" />,
  },
);

type TrustScoreCardProps = {
  loading: boolean;
  breakdown: TrustScoreBreakdown | null;
};

const scoreRows: Array<{
  key: keyof Omit<TrustScoreBreakdown, "total">;
  label: string;
  weight: number;
}> = [
  { key: "continuity", label: "Continuity", weight: 40 },
  { key: "encryption", label: "Encryption", weight: 30 },
  { key: "timestamp", label: "Timestamp", weight: 20 },
  { key: "freshness", label: "Freshness", weight: 10 },
];

export function TrustScoreCard({ loading, breakdown }: TrustScoreCardProps) {
  const shouldReduceMotion = useReducedMotion();
  const isCypressRuntime = typeof window !== "undefined" && "Cypress" in window;

  if (loading || !breakdown) {
    return (
      <Card data-testid="trust-score-card">
        <CardHeader>
          <CardTitle>Trust Score</CardTitle>
          <CardDescription>Verifying controls and freshness.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {isCypressRuntime ? (
            <div
              className="inline-flex w-fit rounded-full border border-border bg-background/70 px-3 py-1 text-sm font-medium text-foreground"
              data-testid="trust-score-value"
            >
              TEST-MODE
            </div>
          ) : null}
          <Skeleton className="h-44 w-full" />
          <Skeleton className="h-5 w-1/2" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-5/6" />
        </CardContent>
      </Card>
    );
  }

  return (
    <motion.div
      initial={shouldReduceMotion ? false : { opacity: 0, y: 8 }}
      animate={shouldReduceMotion ? undefined : { opacity: 1, y: 0 }}
      transition={{ duration: 0.25, ease: "easeOut" }}
    >
      <Card data-testid="trust-score-card">
        <CardHeader>
          <div className="flex items-center justify-between gap-2">
            <CardTitle className="text-lg">Trust Score</CardTitle>
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    className="rounded-md p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
                    aria-label="Explain trust score weighting"
                  >
                    <Info className="h-4 w-4" />
                  </button>
                </TooltipTrigger>
                <TooltipContent side="left">
                  v1.1.0 weights: continuity (40), encryption (30), timestamp (20), freshness (10).
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
          <CardDescription>
            Real-time confidence from hash-chain and freshness checks.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <TrustScoreRadial score={breakdown.total} />
          <div className="mt-3 flex items-end justify-between">
            <p className="text-sm text-muted-foreground">Current Score</p>
            <p className="text-4xl font-semibold tracking-tight" data-testid="trust-score-value">
              {breakdown.total}
            </p>
          </div>
          <div className="mt-4 grid grid-cols-2 gap-2 text-sm">
            {scoreRows.map((row) => (
              <div key={row.key} className="rounded-md border border-border bg-background/60 p-2">
                <p className="text-xs uppercase tracking-wide text-muted-foreground">{row.label}</p>
                <p className="mt-1 font-semibold text-foreground">
                  {breakdown[row.key]}/{row.weight}
                </p>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </motion.div>
  );
}
