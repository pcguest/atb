const FEATURES = [
  {
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
      </svg>
    ),
    title: "Hash-Chained Logs",
    description:
      "Every event is cryptographically linked to the previous one via SHA-256. Tamper with any event and the entire chain breaks — making alterations immediately detectable.",
    tag: "SHA-256",
  },
  {
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
    ),
    title: "RFC 8785 Canonicalization",
    description:
      "JSON is canonicalized using the IETF RFC 8785 standard before hashing. This ensures identical hashes across Go, Python, and TypeScript — regardless of key ordering or whitespace.",
    tag: "RFC 8785",
  },
  {
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
      </svg>
    ),
    title: "Local-First Storage",
    description:
      "Bundles are stored as NDJSON files in a local run.atb/ directory. No database, no server, no configuration. Works offline and in air-gapped environments.",
    tag: "Zero Infrastructure",
  },
  {
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
      </svg>
    ),
    title: "Multi-Language SDKs",
    description:
      "Native SDKs for Go (CLI), Python, and TypeScript. Cross-language hash compatibility is verified by golden tests — the same event produces the same hash in every language.",
    tag: "Go · Python · TypeScript",
  },
  {
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
      </svg>
    ),
    title: "Framework Integrations",
    description:
      "Drop-in integrations for LangChain, LlamaIndex, CrewAI, and more. Automatically record LLM calls, tool invocations, agent decisions, and chain outputs with zero boilerplate.",
    tag: "LangChain · LlamaIndex",
  },
  {
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
      </svg>
    ),
    title: "Compliance-Ready",
    description:
      "Export bundles as PDF compliance reports, Markdown summaries, or PowerPoint executive presentations. Built-in role-based views for developers, managers, and executives.",
    tag: "SOC2 · GDPR · HIPAA",
  },
];

export default function Features() {
  return (
    <section id="features" className="py-24 relative">
      {/* Subtle separator */}
      <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-[#1e1e2e] to-transparent" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section header */}
        <div className="text-center mb-16">
          <span className="inline-block font-mono text-indigo-400 text-sm mb-3">
            // why atb
          </span>
          <h2 className="text-3xl sm:text-4xl font-bold text-white mb-4">
            Built for Production AI
          </h2>
          <p className="text-[#9ca3af] text-lg max-w-2xl mx-auto">
            AI agents make thousands of decisions. ATB makes every one of them
            auditable, replayable, and tamper-evident.
          </p>
        </div>

        {/* Features grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {FEATURES.map((feature, i) => (
            <div
              key={i}
              className="group p-6 rounded-xl border border-[#1e1e2e] bg-[#111118]/50 hover:border-indigo-500/30 hover:bg-[#111118] transition-all duration-300"
            >
              {/* Icon */}
              <div className="w-10 h-10 rounded-lg bg-indigo-600/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400 mb-4 group-hover:bg-indigo-600/20 transition-colors">
                {feature.icon}
              </div>

              {/* Title */}
              <h3 className="text-white font-semibold text-lg mb-2">
                {feature.title}
              </h3>

              {/* Description */}
              <p className="text-[#9ca3af] text-sm leading-relaxed mb-4">
                {feature.description}
              </p>

              {/* Tag */}
              <span className="inline-block font-mono text-xs text-indigo-400 bg-indigo-500/10 border border-indigo-500/20 px-2 py-1 rounded">
                {feature.tag}
              </span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
