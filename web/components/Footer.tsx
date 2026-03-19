export default function Footer() {
  return (
    <footer className="border-t border-[#1e1e2e] py-12">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-8 mb-8">
          {/* Brand */}
          <div className="md:col-span-2">
            <div className="flex items-center gap-2 mb-3">
              <div className="w-7 h-7 rounded-lg bg-indigo-600 flex items-center justify-center">
                <span className="font-mono font-bold text-white text-xs">A</span>
              </div>
              <span className="font-mono font-semibold text-white">ATB</span>
            </div>
            <p className="text-[#6b7280] text-sm leading-relaxed max-w-xs">
              Agent Trace Bundle for private AI systems. Verifiable bundles, local-first storage,
              and review-ready exports.
            </p>
            <div className="flex items-center gap-3 mt-4">
              <a
                href="https://github.com/pcguest/atb"
                target="_blank"
                rel="noopener noreferrer"
                className="text-[#6b7280] hover:text-white transition-colors"
                aria-label="GitHub"
              >
                <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
                </svg>
              </a>
            </div>
          </div>

          {/* Product */}
          <div>
            <h4 className="text-white font-medium text-sm mb-3">Product</h4>
            <ul className="space-y-2">
              {[
                { label: "Features", href: "#features" },
                { label: "Who It Fits", href: "#fit" },
                {
                  label: "Release Notes",
                  href: "https://github.com/pcguest/atb/blob/main/docs/releases/v1.1.0.md",
                },
                { label: "Security Model", href: "https://github.com/pcguest/atb/blob/main/docs/security.md" },
              ].map((item) => (
                <li key={item.label}>
                  <a
                    href={item.href}
                    target={item.href.startsWith("http") ? "_blank" : undefined}
                    rel={item.href.startsWith("http") ? "noopener noreferrer" : undefined}
                    className="text-[#6b7280] hover:text-white text-sm transition-colors"
                  >
                    {item.label}
                  </a>
                </li>
              ))}
            </ul>
          </div>

          {/* Developers */}
          <div>
            <h4 className="text-white font-medium text-sm mb-3">Developers</h4>
            <ul className="space-y-2">
              {[
                { label: "GitHub", href: "https://github.com/pcguest/atb" },
                {
                  label: "CLI Reference",
                  href: "https://github.com/pcguest/atb/tree/main/cmd/atb",
                },
                {
                  label: "Python SDK",
                  href: "https://github.com/pcguest/atb/tree/main/sdk/python",
                },
                {
                  label: "TypeScript SDK",
                  href: "https://github.com/pcguest/atb/tree/main/sdk/typescript",
                },
              ].map((item) => (
                <li key={item.label}>
                  <a
                    href={item.href}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-[#6b7280] hover:text-white text-sm transition-colors"
                  >
                    {item.label}
                  </a>
                </li>
              ))}
            </ul>
          </div>
        </div>

        {/* Bottom bar */}
        <div className="flex flex-col sm:flex-row items-center justify-between pt-8 border-t border-[#1e1e2e] gap-4">
          <p className="text-[#6b7280] text-sm font-mono">© 2026 ATB. MIT License.</p>
          <div className="flex items-center gap-4 text-[#6b7280] text-sm">
            <span className="font-mono text-xs">Built and maintained by Patrick Guest.</span>
          </div>
        </div>
      </div>
    </footer>
  );
}
