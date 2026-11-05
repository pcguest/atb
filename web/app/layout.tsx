import type { Metadata } from "next";
import Script from "next/script";

import { Providers } from "@/app/providers";
import { themeBootstrapScript } from "@/lib/theme";
import "./globals.css";

export const metadata: Metadata = {
  title: "ATB | Portable evidence for AI-agent incidents",
  description:
    "Capture, verify, investigate, and export portable evidence for AI-agent incidents without default hosted storage.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className="min-h-screen bg-background text-foreground antialiased">
        <Script id="atb-theme-bootstrap" strategy="beforeInteractive">
          {themeBootstrapScript}
        </Script>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
