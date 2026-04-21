"use client";

import type { ReactNode } from "react";

import { VerificationBannerConnected } from "@/components/dashboard/VerificationBanner";

export default function ViewLayout({ children }: { children: ReactNode }) {
  return (
    <>
      <VerificationBannerConnected />
      <div
        style={{ paddingTop: "var(--banner-h)" }}
        className="flex h-screen overflow-hidden bg-background"
      >
        {children}
      </div>
    </>
  );
}
