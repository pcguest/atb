import type { Metadata } from "next";

import { Providers } from "@/app/providers";
import "./globals.css";

export const metadata: Metadata = {
  title: "ATB | Local-first trace bundles for AI systems",
  description:
    "Open-source local-first trace bundles for AI systems. Go CLI, localhost viewer, deterministic exports, and Python and TypeScript SDKs.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body className="min-h-screen bg-background text-foreground antialiased">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
