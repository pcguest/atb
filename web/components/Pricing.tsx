const SEGMENTS = [
  {
    name: "Security-minded AI teams",
    description:
      "Keep agent traces local and still have something engineering, security, and leadership can verify after an incident or model failure.",
    cta: "Read Quickstart",
    ctaHref: "https://github.com/pcguest/atb/blob/main/docs/quickstart.md",
    features: [
      "Local bundles for internal copilots and agent workflows",
      "CLI verification for post-incident review",
      "Privacy reveal audit trail for sensitive fields",
      "No required hosted trace backend",
    ],
  },
  {
    name: "Consultancies",
    description:
      "Deliver a portable audit artefact with the work. Clients get a bundle they can inspect locally instead of screenshots, vendor seats, or reconstructed notes.",
    cta: "See Export Docs",
    ctaHref: "https://github.com/pcguest/atb/blob/main/docs/compliance/export.md",
    features: [
      "Portable bundles for project handoff",
      "Deterministic export paths for review packs",
      "Works across Python, TypeScript, and CLI workflows",
      "Useful when clients cannot adopt another platform",
    ],
  },
  {
    name: "Privacy-sensitive enterprise builders",
    description:
      "Use ATB when trace data, customer context, or internal policy make default external storage a hard sell from day one.",
    cta: "Review Security Model",
    ctaHref: "https://github.com/pcguest/atb/blob/main/docs/security.md",
    features: [
      "Local-first storage for sensitive traces",
      "Authenticated privacy reveal controls",
      "Evidence exports for audit and governance work",
      "Clear boundary between product data and review artefacts",
    ],
  },
];

export default function Pricing() {
  return (
    <section id="fit" className="py-24 relative">
      <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-[#1e1e2e] to-transparent" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section header */}
        <div className="text-center mb-16">
          <span className="inline-block font-mono text-indigo-400 text-sm mb-3">
            {"// who it fits"}
          </span>
          <h2 className="text-3xl sm:text-4xl font-bold text-white mb-4">Good fit, clear fit</h2>
          <p className="text-[#9ca3af] text-lg max-w-2xl mx-auto">
            ATB is for teams that need a trace they can keep, verify, and hand over without
            introducing another hosted system.
          </p>
        </div>

        {/* Audience cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 items-start">
          {SEGMENTS.map((segment, index) => (
            <div
              key={segment.name}
              className={`rounded-xl border p-6 transition-all duration-300 ${
                index === 0
                  ? "border-indigo-500/50 bg-indigo-600/5 glow"
                  : "border-[#1e1e2e] bg-[#111118]/50 hover:border-[#2e2e3e]"
              }`}
            >
              <span className="inline-block font-mono text-xs text-indigo-300 bg-indigo-500/10 border border-indigo-500/20 px-2 py-1 rounded mb-4">
                buyer segment
              </span>
              <div className="mb-4">
                <h3 className="text-white font-semibold text-lg">{segment.name}</h3>
                <p className="text-[#6b7280] text-sm mt-2 leading-relaxed">{segment.description}</p>
              </div>

              {/* CTA */}
              <a
                href={segment.ctaHref}
                target={segment.ctaHref.startsWith("http") ? "_blank" : undefined}
                rel={segment.ctaHref.startsWith("http") ? "noopener noreferrer" : undefined}
                className={`block w-full text-center py-2.5 rounded-lg font-medium text-sm transition-all mb-6 ${
                  index === 0
                    ? "bg-indigo-600 hover:bg-indigo-500 text-white"
                    : "border border-[#1e1e2e] hover:border-indigo-500/50 text-[#e2e8f0] hover:bg-indigo-500/5"
                }`}
              >
                {segment.cta}
              </a>

              {/* Features */}
              <ul className="space-y-2.5">
                {segment.features.map((feature) => (
                  <li key={feature} className="flex items-start gap-2.5">
                    <svg
                      className="w-4 h-4 text-indigo-400 mt-0.5 shrink-0"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M5 13l4 4L19 7"
                      />
                    </svg>
                    <span className="text-[#9ca3af] text-sm">{feature}</span>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* Note */}
        <p className="text-center text-[#6b7280] text-sm mt-8 font-mono">
          Start with the open source CLI and SDKs. Keep raw traces local by default.
        </p>
      </div>
    </section>
  );
}
