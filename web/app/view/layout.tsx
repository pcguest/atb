"use client";

import type { ReactNode } from "react";

import { VerificationBannerConnected } from "@/components/dashboard/VerificationBanner";

export default function ViewLayout({ children }: { children: ReactNode }) {
  return (
    <>
      <VerificationBannerConnected />
      <div className="view-shell flex h-screen overflow-hidden bg-background">
        {children}
      </div>
    </>
  );
}
