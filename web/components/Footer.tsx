export default function Footer() {
  return (
    <footer className="bg-background py-12">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mb-8 grid grid-cols-1 gap-8 md:grid-cols-4">
          {/* Brand */}
          <div className="md:col-span-2">
            <div className="flex items-center gap-2 mb-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-md border border-primary/35 bg-primary/10">
                <span className="font-mono text-xs font-bold text-primary">A</span>
              </div>
              <span className="font-semibold text-foreground">ATB</span>
            </div>
            <p className="max-w-xs text-sm leading-relaxed text-muted-foreground">
              Portable evidence for AI-agent incidents. Capture, verify, investigate, and export
              locally.
            </p>
            <div className="flex items-center gap-3 mt-4">
              <a
                href="https://github.com/pcguest/atb"
                target="_blank"
                rel="noopener noreferrer"
                className="text-muted-foreground transition-colors hover:text-foreground"
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
            <h4 className="mb-3 text-sm font-medium text-foreground">Docs</h4>
            <ul className="space-y-2">
              {[
                {
                  label: "Quickstart",
                  href: "https://github.com/pcguest/atb/blob/main/docs/getting-started/quickstart.md",
                },
                {
                  label: "Capture guide",
                  href: "https://github.com/pcguest/atb/blob/main/docs/capture/overview.md",
                },
                {
                  label: "Incident forensics",
                  href: "https://github.com/pcguest/atb/blob/main/docs/investigate/incidents.md",
                },
                {
                  label: "Documentation hub",
                  href: "https://github.com/pcguest/atb/blob/main/docs/README.md",
                },
                {
                  label: "Security Model",
                  href: "https://github.com/pcguest/atb/blob/main/docs/concepts/trust-model.md",
                },
                { label: "Current Scope", href: "#scope" },
              ].map((item) => (
                <li key={item.label}>
                  <a
                    href={item.href}
                    target={item.href.startsWith("http") ? "_blank" : undefined}
                    rel={item.href.startsWith("http") ? "noopener noreferrer" : undefined}
                    className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                  >
                    {item.label}
                  </a>
                </li>
              ))}
            </ul>
          </div>

          {/* Developers */}
          <div>
            <h4 className="mb-3 text-sm font-medium text-foreground">Developers</h4>
            <ul className="space-y-2">
              {[
                { label: "GitHub", href: "https://github.com/pcguest/atb" },
                {
                  label: "CLI Reference",
                  href: "https://github.com/pcguest/atb/tree/main/cmd/atb",
                },
                {
                  label: "Viewer API",
                  href: "https://github.com/pcguest/atb/blob/main/docs/api/README.md",
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
                    className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                  >
                    {item.label}
                  </a>
                </li>
              ))}
            </ul>
          </div>
        </div>

        {/* Bottom bar */}
        <div className="flex flex-col items-center justify-between gap-4 border-t border-border pt-8 sm:flex-row">
          <p className="font-mono text-sm text-muted-foreground">© 2026 ATB. MIT License.</p>
          <div className="flex items-center gap-4 text-sm text-muted-foreground">
            <span className="font-mono text-xs">
              Built and maintained in the open by Patrick Guest.
            </span>
          </div>
        </div>
      </div>
    </footer>
  );
}
