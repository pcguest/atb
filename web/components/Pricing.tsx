const PLANS = [
  {
    name: "Open Source",
    price: "Free",
    period: "forever",
    description: "For individual developers and open source projects.",
    highlight: false,
    cta: "Get Started",
    ctaHref: "https://github.com/pcguest/atb",
    features: [
      "Go CLI (all commands)",
      "Python SDK",
      "TypeScript SDK",
      "Local storage (run.atb/)",
      "Hash-chained bundles",
      "RFC 8785 canonicalization",
      "LangChain & LlamaIndex integrations",
      "MIT License",
    ],
  },
  {
    name: "Pro",
    price: "$20",
    period: "per month",
    description: "For professional developers and small teams who need cloud sync and sharing.",
    highlight: true,
    cta: "Join Waitlist",
    ctaHref: "#waitlist",
    features: [
      "Everything in Open Source",
      "Cloud sync (encrypted Cloudflare R2)",
      "Password-protected sharing links",
      "Time-limited share expiry",
      "Web viewer (timeline, JSON explorer)",
      "PDF & Markdown export",
      "API access (key + usage stats)",
      "Priority support",
    ],
  },
  {
    name: "Enterprise",
    price: "$500",
    period: "per month",
    description: "For teams with compliance requirements and enterprise security needs.",
    highlight: false,
    cta: "Contact Us",
    ctaHref: "mailto:enterprise@atb.dev",
    features: [
      "Everything in Pro",
      "SSO (SAML/OIDC)",
      "Custom RBAC roles",
      "SOC2 / GDPR / HIPAA export templates",
      "Admin audit logs",
      "White-labeling & custom branding",
      "SLA & dedicated support",
      "On-premise deployment option",
    ],
  },
];

export default function Pricing() {
  return (
    <section id="pricing" className="py-24 relative">
      <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-[#1e1e2e] to-transparent" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section header */}
        <div className="text-center mb-16">
          <span className="inline-block font-mono text-indigo-400 text-sm mb-3">
            // pricing
          </span>
          <h2 className="text-3xl sm:text-4xl font-bold text-white mb-4">
            Start Free, Scale When Ready
          </h2>
          <p className="text-[#9ca3af] text-lg max-w-2xl mx-auto">
            The core engine is open source and always will be. Pay only when
            you need cloud features and team collaboration.
          </p>
        </div>

        {/* Pricing cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 items-start">
          {PLANS.map((plan) => (
            <div
              key={plan.name}
              className={`relative rounded-xl border p-6 transition-all duration-300 ${
                plan.highlight
                  ? "border-indigo-500/50 bg-indigo-600/5 glow"
                  : "border-[#1e1e2e] bg-[#111118]/50 hover:border-[#2e2e3e]"
              }`}
            >
              {plan.highlight && (
                <div className="absolute -top-3 left-1/2 -translate-x-1/2">
                  <span className="inline-block px-3 py-1 rounded-full bg-indigo-600 text-white text-xs font-medium">
                    Most Popular
                  </span>
                </div>
              )}

              {/* Plan name */}
              <div className="mb-4">
                <h3 className="text-white font-semibold text-lg">{plan.name}</h3>
                <p className="text-[#6b7280] text-sm mt-1">{plan.description}</p>
              </div>

              {/* Price */}
              <div className="mb-6">
                <div className="flex items-baseline gap-1">
                  <span className="text-4xl font-bold text-white">{plan.price}</span>
                  {plan.period !== "forever" && (
                    <span className="text-[#6b7280] text-sm">/{plan.period}</span>
                  )}
                </div>
                {plan.period === "forever" && (
                  <span className="text-[#22c55e] text-sm font-mono">forever free</span>
                )}
              </div>

              {/* CTA */}
              <a
                href={plan.ctaHref}
                target={plan.ctaHref.startsWith("http") ? "_blank" : undefined}
                rel={plan.ctaHref.startsWith("http") ? "noopener noreferrer" : undefined}
                className={`block w-full text-center py-2.5 rounded-lg font-medium text-sm transition-all mb-6 ${
                  plan.highlight
                    ? "bg-indigo-600 hover:bg-indigo-500 text-white"
                    : "border border-[#1e1e2e] hover:border-indigo-500/50 text-[#e2e8f0] hover:bg-indigo-500/5"
                }`}
              >
                {plan.cta}
              </a>

              {/* Features */}
              <ul className="space-y-2.5">
                {plan.features.map((feature) => (
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
          All plans include the open source CLI and SDKs. No credit card required to start.
        </p>
      </div>
    </section>
  );
}
