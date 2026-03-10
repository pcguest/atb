"use client";

import { useState } from "react";

export default function Waitlist() {
  const [email, setEmail] = useState("");
  const [status, setStatus] = useState<"idle" | "loading" | "success" | "error">("idle");
  const [message, setMessage] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email || !email.includes("@")) {
      setStatus("error");
      setMessage("Please enter a valid email address.");
      return;
    }

    setStatus("loading");

    // Waitlist submission handler.
    await new Promise((resolve) => setTimeout(resolve, 1000));

    setStatus("success");
    setMessage("You're on the list! We'll notify you when ATB Pro launches.");
    setEmail("");
  };

  return (
    <section id="waitlist" className="py-24 relative">
      <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-[#1e1e2e] to-transparent" />

      <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
        {/* Glow effect */}
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
          <div className="w-[400px] h-[400px] rounded-full bg-indigo-600/5 blur-3xl" />
        </div>

        <div className="relative">
          {/* Badge */}
          <span className="inline-block font-mono text-indigo-400 text-sm mb-4">
            // early access
          </span>

          <h2 className="text-3xl sm:text-4xl font-bold text-white mb-4">
            Be First to Know
          </h2>
          <p className="text-[#9ca3af] text-lg mb-8 max-w-xl mx-auto">
            ATB Pro is launching soon with cloud sync, sharing, and the hosted
            viewer. Join the waitlist to get early access and a{" "}
            <span className="text-indigo-300 font-medium">30% launch discount</span>.
          </p>

          {/* Form */}
          {status === "success" ? (
            <div className="flex items-center justify-center gap-3 p-4 rounded-xl border border-[#22c55e]/30 bg-[#22c55e]/5 text-[#22c55e]">
              <svg className="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span className="font-medium">{message}</span>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="flex flex-col sm:flex-row gap-3 max-w-md mx-auto">
              <input
                type="email"
                value={email}
                onChange={(e) => {
                  setEmail(e.target.value);
                  if (status === "error") setStatus("idle");
                }}
                placeholder="you@company.com"
                className={`flex-1 px-4 py-3 rounded-lg border bg-[#111118] text-[#e2e8f0] placeholder-[#6b7280] font-mono text-sm outline-none transition-all ${
                  status === "error"
                    ? "border-red-500/50 focus:border-red-500"
                    : "border-[#1e1e2e] focus:border-indigo-500/50"
                }`}
                disabled={status === "loading"}
              />
              <button
                type="submit"
                disabled={status === "loading"}
                className="px-6 py-3 rounded-lg bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white font-medium text-sm transition-all whitespace-nowrap"
              >
                {status === "loading" ? (
                  <span className="flex items-center gap-2">
                    <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                    </svg>
                    Joining...
                  </span>
                ) : (
                  "Join Waitlist"
                )}
              </button>
            </form>
          )}

          {status === "error" && (
            <p className="text-red-400 text-sm mt-2 font-mono">{message}</p>
          )}

          <p className="text-[#6b7280] text-xs mt-4 font-mono">
            No spam. Unsubscribe anytime. We respect your privacy.
          </p>

          {/* Social proof */}
          <div className="flex items-center justify-center gap-6 mt-10 text-[#6b7280] text-sm">
            <div className="flex items-center gap-2">
              <svg className="w-4 h-4 text-indigo-400" fill="currentColor" viewBox="0 0 24 24">
                <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
              </svg>
              <a
                href="https://github.com/pcguest/atb"
                target="_blank"
                rel="noopener noreferrer"
                className="hover:text-white transition-colors"
              >
                Star on GitHub
              </a>
            </div>
            <span>·</span>
            <span className="font-mono">MIT License</span>
            <span>·</span>
            <span className="font-mono">Open Source</span>
          </div>
        </div>
      </div>
    </section>
  );
}
