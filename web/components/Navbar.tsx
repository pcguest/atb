export default function Navbar() {
  return (
    <nav
      className="fixed inset-x-0 top-0 z-50 border-b border-border bg-background/95 backdrop-blur"
      aria-label="Primary navigation"
    >
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <a
          href="#top"
          className="flex items-center gap-2.5 rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <span className="grid h-8 w-8 place-items-center rounded-md border border-primary/35 bg-primary/10 font-mono text-xs font-semibold text-primary">
            A
          </span>
          <span className="font-semibold tracking-tight text-foreground">ATB</span>
          <span className="hidden text-sm text-muted-foreground sm:inline">Agent Trace Bundle</span>
        </a>
        <div className="hidden items-center gap-6 md:flex">
          <a href="#features" className="text-sm text-muted-foreground hover:text-foreground">
            Product
          </a>
          <a href="#demo" className="text-sm text-muted-foreground hover:text-foreground">
            Integrate
          </a>
          <a href="#scope" className="text-sm text-muted-foreground hover:text-foreground">
            Trust boundary
          </a>
        </div>
        <a
          href="https://github.com/pcguest/atb/blob/main/docs/getting-started/quickstart.md"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex min-h-9 items-center rounded-md border border-border bg-card px-3 text-sm font-medium text-foreground hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          Quickstart
        </a>
      </div>
    </nav>
  );
}
