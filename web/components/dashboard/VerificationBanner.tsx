"use client";

import type { VerificationResponse } from "@/lib/types";

type VerificationBannerProps = {
  verification: VerificationResponse | null;
  loading: boolean;
  error: string | null;
};

export function VerificationBanner({ verification, loading, error }: VerificationBannerProps) {
  if (loading) {
    return (
      <div className="rounded-lg border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-slate-300">
        Verifying hash chain...
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-500/50 bg-red-950 px-4 py-3 text-sm text-red-200">
        Viewer error: {error}
      </div>
    );
  }

  if (!verification) {
    return null;
  }

  if (verification.status === "valid") {
    return (
      <div className="rounded-lg border border-emerald-500/40 bg-emerald-950 px-4 py-3 text-sm text-emerald-100">
        <span className="font-semibold">Integrity Verified</span>
        <span className="ml-2">{verification.message}</span>
      </div>
    );
  }

  return (
    <div className="rounded-lg border-2 border-red-500 bg-red-950 px-4 py-3 text-sm text-red-100">
      <span className="font-bold">TAMPER DETECTED</span>
      <span className="ml-2">{verification.message}</span>
    </div>
  );
}
