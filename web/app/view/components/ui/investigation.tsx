"use client";

import { AlertCircle, Check, CheckCircle2, Copy, LoaderCircle, MinusCircle } from "lucide-react";
import { useState } from "react";

import { cn } from "@/lib/utils";

export type StatusTone = "verified" | "warning" | "danger" | "unknown";

const statusStyles: Record<StatusTone, string> = {
  verified: "border-verified/35 bg-verified/10 text-verified",
  warning: "border-warning/35 bg-warning/10 text-warning",
  danger: "border-danger/35 bg-danger/10 text-danger",
  unknown: "border-border bg-muted/45 text-muted-foreground",
};

export function StatusBadge({
  tone,
  children,
  className,
}: {
  tone: StatusTone;
  children: React.ReactNode;
  className?: string;
}) {
  const Icon = tone === "verified" ? CheckCircle2 : tone === "danger" ? AlertCircle : MinusCircle;
  return (
    <span
      className={cn(
        "inline-flex min-h-7 items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium",
        statusStyles[tone],
        className,
      )}
    >
      <Icon className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
      {children}
    </span>
  );
}

export function SurfaceHeader({
  eyebrow,
  title,
  description,
  action,
}: {
  eyebrow?: string;
  title: string;
  description?: string;
  action?: React.ReactNode;
}) {
  return (
    <header className="flex flex-wrap items-end justify-between gap-4 border-b border-border pb-4">
      <div className="max-w-3xl">
        {eyebrow && (
          <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
            {eyebrow}
          </p>
        )}
        <h2 className="mt-1 text-xl font-semibold tracking-tight">{title}</h2>
        {description && (
          <p className="mt-1.5 text-sm leading-6 text-muted-foreground">{description}</p>
        )}
      </div>
      {action}
    </header>
  );
}

export function EmptyState({
  title,
  children,
  action,
}: {
  title?: string;
  children: React.ReactNode;
  action?: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border border-dashed border-border bg-card/45 px-5 py-8 text-center">
      {title && <p className="text-sm font-medium text-foreground">{title}</p>}
      <div
        className={cn(
          "mx-auto max-w-xl text-sm leading-6 text-muted-foreground",
          title && "mt-1.5",
        )}
      >
        {children}
      </div>
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}

export function LoadingState({ label = "Loading evidence…" }: { label?: string }) {
  return (
    <div
      className="flex min-h-40 items-center justify-center gap-2 rounded-lg border border-border bg-card text-sm text-muted-foreground"
      role="status"
    >
      <LoaderCircle className="h-4 w-4 animate-spin" aria-hidden="true" />
      {label}
    </div>
  );
}

export function ErrorState({
  title = "This evidence could not be loaded",
  children,
  onRetry,
}: {
  title?: string;
  children: React.ReactNode;
  onRetry?: () => void;
}) {
  return (
    <div className="rounded-lg border border-danger/35 bg-danger/5 p-5" role="alert">
      <div className="flex gap-3">
        <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-danger" aria-hidden="true" />
        <div>
          <p className="text-sm font-medium">{title}</p>
          <div className="mt-1 text-sm leading-6 text-muted-foreground">{children}</div>
          {onRetry && <button type="button" className="mt-3 rounded-md border border-border bg-card px-3 py-2 text-sm hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" onClick={onRetry}>Try again</button>}
        </div>
      </div>
    </div>
  );
}

export function CopyAction({ value, label }: { value: string; label: string }) {
  const [copied, setCopied] = useState(false);
  const [failed, setFailed] = useState(false);
  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setFailed(false);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      setFailed(true);
    }
  }
  return (
    <button
      type="button"
      onClick={() => void copy()}
      className="inline-flex h-8 items-center gap-1.5 rounded-md border border-border bg-card px-2.5 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
      disabled={!value}
      aria-label={label}
    >
      {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
      <span role="status">{failed ? "Copy failed" : copied ? "Copied" : "Copy"}</span>
    </button>
  );
}
