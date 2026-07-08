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
  message?: string | null;
};

const BASE =
  "fixed inset-x-0 top-0 z-50 flex h-[var(--banner-h)] w-full items-center gap-2 border-b px-4";

export function VerificationBanner({
  status,
  chainLength,
  headHash,
  bundlePath,
  message,
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

  const diagnosis = message?.trim() || "hash chain verification failed";
  const recheckPath = bundlePath ? displayBundlePath(bundlePath) : null;

  return (
    <div
      role="alert"
      aria-live="assertive"
      aria-label={`Bundle tamper detected. Interaction restricted. ${diagnosis}`}
      className="fixed inset-x-0 top-0 z-50 flex min-h-[var(--banner-h)] w-full flex-col justify-center gap-0.5 border-b border-red-500 bg-red-950 px-4 py-1.5"
    >
      <div className="flex items-center gap-2">
        <ShieldOff
          className="h-3.5 w-3.5 shrink-0 animate-pulse text-red-400"
          aria-hidden="true"
        />
        <span className="font-mono text-xs font-bold uppercase tracking-widest text-red-300">
          ⚠ TAMPER DETECTED
        </span>
        <span className="font-mono text-xs text-red-300">chain: {chainLength ?? 0} events</span>
        {recheckPath && (
          <span
            className="ml-auto min-w-0 flex-1 truncate text-right font-mono text-xs text-red-200"
            title={recheckPath}
          >
            {recheckPath}
          </span>
        )}
      </div>
      <p className="truncate font-mono text-xs text-red-200" title={diagnosis}>
        {diagnosis}
      </p>
      {bundlePath && (
        <p className="truncate font-mono text-xs text-red-300/90">
          {/* The copyable command needs the real path; the shortened display
              form may not resolve when pasted into a shell. */}
          re-check: <span className="select-all text-red-200">atb verify {bundlePath}</span>
        </p>
      )}
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

  // The meta query only runs once verification is valid; on an invalid result
  // its cached data can be stale, so the failing verification response's own
  // bundle path takes precedence there.
  const bundlePath =
    status === "invalid"
      ? (verificationQuery.data?.bundle_path ?? metaQuery.data?.bundle_path ?? null)
      : (metaQuery.data?.bundle_path ?? verificationQuery.data?.bundle_path ?? null);

  return (
    <VerificationBanner
      status={status}
      chainLength={verificationQuery.data?.chain_length}
      headHash={verificationQuery.data?.head_hash ?? null}
      bundlePath={bundlePath}
      message={verificationQuery.data?.message ?? null}
    />
  );
}
