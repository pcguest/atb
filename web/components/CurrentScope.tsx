const SHIPPED = [
  "`atb init`, `append`, `snapshot`, `verify`, `archive`, `export`, `trust-report`, and `evidence` in the Go CLI",
  "`atb capture run` and `atb import chatlog` for Capture v1 workflows",
  "`atb corroborate`, `atb push`, `atb anchor`, and `atb mcp serve`",
  "`atb view` forensic investigation with findings, timeline, evidence, and explicit trust boundaries",
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
    <div className="rounded-xl border border-border bg-card p-6">
      <span className="mb-4 inline-block rounded border border-primary/25 bg-primary/10 px-2 py-1 font-mono text-xs text-primary">
        {eyebrow}
      </span>
      <h3 className="mb-4 text-lg font-semibold text-foreground">{title}</h3>
      <ul className="space-y-3">
        {items.map((item) => (
          <li key={item} className="flex items-start gap-3">
            <svg
              className="mt-0.5 h-4 w-4 shrink-0 text-primary"
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
            <span className="text-sm leading-relaxed text-muted-foreground">{item}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

export default function CurrentScope() {
  return (
    <section id="scope" className="border-b border-border bg-surface-1 py-20">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mb-12 max-w-3xl">
          <span className="mb-3 inline-block font-mono text-xs font-semibold uppercase tracking-[0.16em] text-primary">
            Current scope
          </span>
          <h2 className="mb-4 text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
            A precise boundary builds trust.
          </h2>
          <p className="max-w-3xl text-lg leading-8 text-muted-foreground">
            ATB is a local-first evidence toolchain. These capabilities are explicit so a reviewer
            can distinguish what the bundle establishes from what it cannot.
          </p>
        </div>

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <ScopeCard eyebrow="shipped today" title="Implemented in this repo" items={SHIPPED} />
          <ScopeCard
            eyebrow="not shipped"
            title="Do not infer these capabilities"
            items={NOT_SHIPPED}
          />
        </div>

        <p className="mt-8 text-sm text-muted-foreground">
          Mortise may add independent organisational custody. Tenon supplies the shared product
          language. Neither changes what a local ATB bundle proves.
        </p>
      </div>
    </section>
  );
}
