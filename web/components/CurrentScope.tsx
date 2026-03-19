const SHIPPED = [
  "`atb init`, `append`, `snapshot`, `verify`, `archive`, `export`, and `trust-report` in the Go CLI",
  "`atb view` running a localhost viewer with verification, timeline, graph, and inspector",
  "Optional bundle encryption and decryption for local handoff workflows",
  "Deterministic `soc2` and `gdpr` evidence exports",
  "Python SDK plus LangChain callback middleware",
  "TypeScript SDK plus Vercel AI SDK middleware",
];

const NOT_SHIPPED = [
  "Hosted workspaces, shared comments, or collaborative review queues",
  "Plan tiers, seat pricing, billing flows, or SaaS entitlement checks",
  "SSO, tenant management, RBAC, or an enterprise admin console",
  "Managed cloud storage or server-side key custody",
  "Integrations beyond LangChain (Python) and Vercel AI SDK (TypeScript)",
];

function ScopeCard({ title, eyebrow, items }: { title: string; eyebrow: string; items: string[] }) {
  return (
    <div className="rounded-xl border border-[#1e1e2e] bg-[#111118]/60 p-6">
      <span className="inline-block font-mono text-xs text-indigo-300 bg-indigo-500/10 border border-indigo-500/20 px-2 py-1 rounded mb-4">
        {eyebrow}
      </span>
      <h3 className="text-white font-semibold text-lg mb-4">{title}</h3>
      <ul className="space-y-3">
        {items.map((item) => (
          <li key={item} className="flex items-start gap-3">
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
            <span className="text-[#9ca3af] text-sm leading-relaxed">{item}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

export default function CurrentScope() {
  return (
    <section id="scope" className="py-24 relative">
      <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-[#1e1e2e] to-transparent" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-12">
          <span className="inline-block font-mono text-indigo-400 text-sm mb-3">
            {"// current scope"}
          </span>
          <h2 className="text-3xl sm:text-4xl font-bold text-white mb-4">
            What ATB is, and what it is not
          </h2>
          <p className="text-[#9ca3af] text-lg max-w-3xl mx-auto">
            ATB is an open-source local toolchain for tamper-evident AI traces. The repo ships the
            items on the left today. The items on the right are intentionally out of scope for the
            current release.
          </p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <ScopeCard eyebrow="shipped today" title="Implemented in this repo" items={SHIPPED} />
          <ScopeCard
            eyebrow="not shipped"
            title="Do not infer these capabilities"
            items={NOT_SHIPPED}
          />
        </div>

        <p className="text-center text-[#6b7280] text-sm mt-8 font-mono">
          If you need the right-hand column, ATB is not claiming to solve that yet.
        </p>
      </div>
    </section>
  );
}
