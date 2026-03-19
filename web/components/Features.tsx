const FEATURES = [
  {
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={1.5}
          d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"
        />
      </svg>
    ),
    title: "Verifiable Event Chain",
    description:
      "Every event is cryptographically linked to the previous one. If a trace is edited, reordered, or truncated, verification fails immediately.",
    tag: "SHA-256",
  },
  {
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={1.5}
          d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
        />
      </svg>
    ),
    title: "Portable Evidence",
    description:
      "Bundles and exports travel cleanly between teams. Consultancies can hand clients a file they can inspect instead of reconstructed notes or screenshots.",
    tag: "Deterministic Export",
  },
  {
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={1.5}
          d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"
        />
      </svg>
    ),
    title: "Keep Raw Traces Local",
    description:
      "ATB writes NDJSON bundles to disk by default. No hosted control plane, no required backend, and no need to move sensitive traces off the machine or network.",
    tag: "Local-First",
  },
  {
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={1.5}
          d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
        />
      </svg>
    ),
    title: "Fits Existing AI Stacks",
    description:
      "Use the CLI for local runs or add Python and TypeScript SDKs to agent workflows already in production. The verification model stays the same across languages.",
    tag: "Go · Python · TypeScript",
  },
  {
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={1.5}
          d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
        />
      </svg>
    ),
    title: "Privacy Reveal Controls",
    description:
      "Protected fields stay masked until an authorized reveal is requested. Reveal actions are authenticated, rate-limited, and written back into the audit trail.",
    tag: "Auth + Audit",
  },
  {
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={1.5}
          d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
        />
      </svg>
    ),
    title: "Useful During Review",
    description:
      "ATB helps during incident review, customer handoff, and compliance evidence collection when teams need proof of what ran and how data was handled.",
    tag: "Review-Ready",
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
            {"// why atb"}
          </span>
          <h2 className="text-3xl sm:text-4xl font-bold text-white mb-4">
            Built for teams that need proof
          </h2>
          <p className="text-[#9ca3af] text-lg max-w-2xl mx-auto">
            Security-minded AI teams, consultancies, and enterprise builders use ATB when raw
            traces need to stay private but still stand up to review.
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
              <h3 className="text-white font-semibold text-lg mb-2">{feature.title}</h3>

              {/* Description */}
              <p className="text-[#9ca3af] text-sm leading-relaxed mb-4">{feature.description}</p>

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
