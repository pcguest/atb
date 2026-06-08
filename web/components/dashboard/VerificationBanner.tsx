"use client";

import { Lock, ShieldOff } from "lucide-react";
import { useEffect } from "react";

import { useBundleMetaQuery, useVerificationQuery } from "@/lib/api-client";
import { displayBundlePath } from "@/lib/display-path";
import { truncateSha256 } from "@/lib/hash-display";

// ─── Pure / presentational ───────────────────────────────────────────────────

type VerificationBannerProps = {
  status: "loading" | "valid" | "invalid" | null;
  chainLength?: number;
  headHash?: string | null;
  bundlePath?: string | null;
};

const BASE =
  "fixed inset-x-0 top-0 z-50 flex h-[var(--banner-h)] w-full items-center gap-2 border-b px-4";

export function VerificationBanner({
  status,
  chainLength,
  headHash,
  bundlePath,
}: VerificationBannerProps) {
  useEffect(() => {
    if (status === "invalid") {
      document.documentElement.setAttribute("data-tamper", "true");
    } else {
      document.documentElement.removeAttribute("data-tamper");
    }
    return () => {
      document.documentElement.removeAttribute("data-tamper");
    };
  }, [status]);

  if (status === "loading" || status === null) {
    return (
      <div
        role="status"
        aria-label="Verifying bundle integrity"
        className="fixed inset-x-0 top-0 z-50 h-[var(--banner-h)] w-full animate-pulse border-b border-muted bg-muted/60"
      />
    );
  }

  if (status === "valid") {
    return (
      <div
        role="status"
        aria-label={`Bundle integrity verified. Chain length: ${chainLength ?? 0}`}
        className={`${BASE} border-green-800/50 bg-green-950/80`}
      >
        <Lock className="h-3 w-3 shrink-0 text-green-300" aria-hidden="true" />
        <span className="font-mono text-xs font-medium tracking-wider text-green-400">
          chain: {chainLength ?? 0} events
        </span>
        {headHash && (
          <span className="font-mono text-xs text-green-300" title={headHash}>
            {truncateSha256(headHash)}
          </span>
        )}
        {bundlePath && (
          <span
            className="ml-auto min-w-0 flex-1 truncate text-right font-mono text-xs text-green-200"
            title={displayBundlePath(bundlePath)}
          >
            {displayBundlePath(bundlePath)}
          </span>
        )}
      </div>
    );
  }

  return (
    <div
      role="alert"
      aria-live="assertive"
      aria-label="Bundle tamper detected — interaction restricted"
      className={`${BASE} border-red-500 bg-red-950`}
    >
      <ShieldOff className="h-3.5 w-3.5 shrink-0 animate-pulse text-red-400" aria-hidden="true" />
      <span className="font-mono text-xs font-bold uppercase tracking-widest text-red-300">
        ⚠ TAMPER DETECTED
      </span>
    </div>
  );
}

// ─── Connected ───────────────────────────────────────────────────────────────

export function VerificationBannerConnected() {
  const verificationQuery = useVerificationQuery();
  const verificationValid = verificationQuery.data?.status === "valid";
  const metaQuery = useBundleMetaQuery(verificationValid);

  if (verificationQuery.isLoading) {
    return <VerificationBanner status="loading" />;
  }

  const status = verificationQuery.data?.status ?? null;

  return (
    <VerificationBanner
      status={status}
      chainLength={verificationQuery.data?.chain_length}
      headHash={verificationQuery.data?.head_hash ?? null}
      bundlePath={metaQuery.data?.bundle_path ?? null}
    />
  );
}
