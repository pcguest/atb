"use client";

import { Check, Copy } from "lucide-react";
import * as React from "react";

import { Badge } from "@/app/view/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/app/view/components/ui/card";
import { Skeleton } from "@/app/view/components/ui/skeleton";
import type { BundleMetaResponse, VerificationResponse } from "@/lib/types";

type BundleMetaPanelProps = {
  loading: boolean;
  verification: VerificationResponse | null;
  meta: BundleMetaResponse | null;
  chainLength: number;
};

function HashField({ label, value }: { label: string; value: string | null }) {
  const [copied, setCopied] = React.useState(false);

  async function onCopy(): Promise<void> {
    if (!value) {
      return;
    }
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch {
      setCopied(false);
    }
  }

  return (
    <div className="rounded-md border border-border bg-background/70 p-3">
      <p className="text-xs uppercase tracking-wide text-muted-foreground">{label}</p>
      {value ? (
        <div className="mt-1 flex items-center justify-between gap-2">
          <code
            className="truncate font-mono text-xs text-foreground"
            aria-label={`SHA-256 hash: ${value}`}
          >
            {value}
          </code>
          <button
            type="button"
            className="inline-flex h-7 w-7 items-center justify-center rounded-md border border-border text-muted-foreground hover:bg-muted hover:text-foreground"
            onClick={onCopy}
            aria-label="Copy SHA-256 hash to clipboard"
          >
            {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
          </button>
        </div>
      ) : (
        <p className="mt-1 text-sm text-muted-foreground">Unavailable</p>
      )}
    </div>
  );
}

export function BundleMetaPanel({
  loading,
  verification,
  meta,
  chainLength,
}: BundleMetaPanelProps) {
  if (loading) {
    return (
      <Card data-testid="bundle-meta-panel">
        <CardHeader>
          <CardTitle>Bundle Meta</CardTitle>
          <CardDescription>Core integrity metadata for audit review.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          <Skeleton className="h-20 w-full" />
          <Skeleton className="h-20 w-full" />
          <Skeleton className="h-20 w-full" />
        </CardContent>
      </Card>
    );
  }

  const verifiedAt = meta?.verified_at ?? null;

  return (
    <Card data-testid="bundle-meta-panel">
      <CardHeader>
        <div className="flex items-center justify-between gap-2">
          <CardTitle className="text-lg">Bundle Meta</CardTitle>
          <Badge variant={verification?.status === "valid" ? "success" : "danger"}>
            {verification?.status === "valid" ? "Verified" : "Not Verified"}
          </Badge>
        </div>
        <CardDescription>
          Hash-chain and provenance details from the local ATB viewer API.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-2">
        <div className="rounded-md border border-border bg-background/70 p-3">
          <p className="text-xs uppercase tracking-wide text-muted-foreground">Chain Length</p>
          <p className="mt-1 text-lg font-semibold">{chainLength}</p>
        </div>

        <HashField label="Genesis Hash" value={meta?.genesis_hash ?? null} />
        <HashField label="Head Hash" value={verification?.head_hash ?? null} />

        <div className="rounded-md border border-border bg-background/70 p-3">
          <p className="text-xs uppercase tracking-wide text-muted-foreground">Verified At</p>
          <p className="mt-1 text-sm text-muted-foreground">{verifiedAt ?? "Unavailable"}</p>
        </div>
      </CardContent>
    </Card>
  );
}
