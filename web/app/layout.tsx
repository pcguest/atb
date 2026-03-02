import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "ATB — The Audit Trail for AI Workflows",
  description:
    "Tamper-evident, replayable traces for AI agents. Hash-chained audit logs with RFC 8785 canonicalization. Open source, local-first, developer-first.",
  keywords: [
    "AI audit trail",
    "agent tracing",
    "AI observability",
    "hash chaining",
    "RFC 8785",
    "LangChain audit",
    "AI compliance",
  ],
  authors: [{ name: "ATB" }],
  openGraph: {
    title: "ATB — The Audit Trail for AI Workflows",
    description:
      "Tamper-evident, replayable traces for AI agents. Hash-chained audit logs with RFC 8785 canonicalization.",
    type: "website",
    url: "https://atb.dev",
  },
  twitter: {
    card: "summary_large_image",
    title: "ATB — The Audit Trail for AI Workflows",
    description: "Tamper-evident, replayable traces for AI agents.",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark">
      <body className="bg-[#0a0a0f] text-[#e2e8f0] antialiased">
        {children}
      </body>
    </html>
  );
}
