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
    title: "Hash-Chained Bundles",
    description:
      "ATB stores events as NDJSON records linked with SHA-256 over RFC 8785 canonical JSON. Mutation, reordering, or deletion causes `atb verify` to fail.",
    tag: "atb verify",
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
    title: "Local Viewer",
    description:
      "The shipped viewer runs from `atb view` on localhost. It includes verification status, a timeline, a trace graph, and an event inspector gated by bundle validity.",
    tag: "atb view",
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
    title: "Evidence Export",
    description:
      "ATB can package verified bundles and reports into deterministic `soc2` and `gdpr` archives for repeatable review workflows.",
    tag: "atb export",
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
    title: "Python SDK",
    description:
      "The Python package ships the core `Bundle` API plus `ATBCallbackHandler` for LangChain. It writes the same `.atb` format as the CLI.",
    tag: "LangChain",
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
    title: "TypeScript SDK",
    description:
      "The TypeScript package ships the core `Bundle` API plus `atbMiddleware` for Vercel AI SDK callbacks. Verification stays consistent with the CLI.",
    tag: "Vercel AI SDK",
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
    title: "Masked Reveal Flow",
    description:
      "Configured PII fields are masked by default in the local viewer. Individual reveals use a per-session viewer token, are rate-limited, and append an audit event to the bundle.",
    tag: "/api/v1/privacy/reveal",
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
            {"// shipped today"}
          </span>
          <h2 className="text-3xl sm:text-4xl font-bold text-white mb-4">
            What the repo implements today
          </h2>
          <p className="text-[#9ca3af] text-lg max-w-2xl mx-auto">
            This page describes only the capabilities implemented in the repo today: local bundle
            creation and verification, a localhost viewer, deterministic exports, and two SDK
            integration paths.
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
