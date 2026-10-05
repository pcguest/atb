"use client";

import { Lock, ShieldOff } from "lucide-react";
import { useEffect, useId, useRef, useState } from "react";

import { useBundleMetaQuery, useVerificationQuery } from "@/lib/api-client";
import { displayBundlePath } from "@/lib/display-path";

// ─── Pure / presentational ───────────────────────────────────────────────────

type VerificationBannerProps = {
  status: "loading" | "valid" | "invalid" | null;
  chainLength?: number;
  headHash?: string | null;
  bundlePath?: string | null;
  message?: string | null;
};

const BASE =
  "fixed inset-x-0 top-0 z-50 flex h-[var(--banner-h)] w-full items-center gap-3 border-b px-4";

/**
 * The verification banner owns the *verification state* of the loaded object
 * (L1: is the presented recorded sequence intact?), not object identity — the
 * investigation shell's ObjectIdentityHeader owns identity and evidence-state
 * facts. Exact chain/hash detail is Level 3 and is reachable behind the
 * "Integrity details" disclosure; it is never replaced by a score.
 */
export function VerificationBanner({
  status,
  chainLength,
  headHash,
  bundlePath,
  message,
}: VerificationBannerProps) {
  const [detailsOpen, setDetailsOpen] = useState(false);
  const detailsId = useId();
  const detailsRef = useRef<HTMLDivElement>(null);

  // Dismiss the disclosure on Escape or an outside click.
  useEffect(() => {
    if (!detailsOpen) return;
    const onKey = (event: KeyboardEvent) => { if (event.key === "Escape") setDetailsOpen(false); };
    const onPointer = (event: MouseEvent) => {
      if (detailsRef.current && !detailsRef.current.contains(event.target as Node)) setDetailsOpen(false);
    };
    document.addEventListener("keydown", onKey);
    document.addEventListener("mousedown", onPointer);
    return () => { document.removeEventListener("keydown", onKey); document.removeEventListener("mousedown", onPointer); };
  }, [detailsOpen]);

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
        aria-label="Bundle integrity verified: the presented records match their hash chain and recorded order"
        className={`${BASE} border-green-800/50 bg-green-950/80`}
      >
        <Lock className="h-3.5 w-3.5 shrink-0 text-green-300" aria-hidden="true" />
        <span className="text-xs font-medium text-green-300">Hash chain verified</span>
        <span className="hidden text-xs text-green-200/80 sm:inline">
          — {chainLength ?? 0} recorded events, in recorded order
        </span>
        <div className="relative ml-auto" ref={detailsRef}>
          <button
            type="button"
            onClick={() => setDetailsOpen((open) => !open)}
            aria-expanded={detailsOpen}
            aria-controls={detailsId}
            className="rounded-sm text-xs text-green-200 underline underline-offset-2 hover:text-green-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            Integrity details
          </button>
          {detailsOpen && (
            <div id={detailsId} className="absolute right-0 top-full z-50 mt-1 w-72 max-w-[calc(100vw-2rem)] rounded-md border border-border bg-popover p-3 text-xs text-foreground shadow-lg">
              <dl className="space-y-2">
                <div>
                  <dt className="text-muted-foreground">Chain length</dt>
                  <dd className="mt-0.5">{chainLength ?? 0} recorded events</dd>
                </div>
                {headHash && (
                  <div>
                    <dt className="text-muted-foreground">Head hash</dt>
                    <dd className="mt-0.5 break-all font-mono">{headHash}</dd>
                  </div>
                )}
              </dl>
              <p className="mt-2 leading-5 text-muted-foreground">
                Verification establishes the integrity and recorded order of the records presented. It
                does not establish truth, completeness, or independent custody.
              </p>
            </div>
          )}
        </div>
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
        <span className="font-mono text-xs text-red-300">{chainLength ?? 0} recorded events</span>
        {headHash && (
          <span className="ml-auto min-w-0 flex-1 truncate text-right font-mono text-xs text-red-300" title={headHash}>
            {headHash}
          </span>
        )}
      </div>
      <p className="truncate font-mono text-xs text-red-200" title={diagnosis}>
        {diagnosis}
      </p>
      {recheckPath && (
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
