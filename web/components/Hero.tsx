import { ArrowRight, CheckCircle2, FileSearch, PackageCheck, ShieldCheck } from "lucide-react";

const FLOW = [
  ["01", "Capture", "Record agent activity into a portable bundle."],
  ["02", "Verify", "Check record order and hash-chain integrity."],
  ["03", "Investigate", "Move from findings to exact supporting evidence."],
  ["04", "Export", "Package bounded evidence for offline review."],
];

export default function Hero() {
  return (
    <section id="top" className="border-b border-border bg-background pt-24">
      <div className="mx-auto grid max-w-7xl gap-14 px-4 pb-16 sm:px-6 lg:grid-cols-[1.15fr_0.85fr] lg:px-8 lg:pb-20 lg:pt-12">
        <div className="max-w-3xl">
          <div className="inline-flex items-center gap-2 rounded-full border border-verified/30 bg-verified/10 px-3 py-1.5 text-xs font-medium text-verified">
            <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
            Local-first evidence for AI-agent incidents
          </div>
          <h1 className="mt-7 text-4xl font-semibold leading-[1.08] tracking-[-0.035em] text-foreground sm:text-6xl">
            Know what happened. Verify what was recorded.
          </h1>
          <p className="mt-6 max-w-2xl text-lg leading-8 text-muted-foreground">
            ATB is a local-first evidence system for AI agents. It records agent activity into
            portable, tamper-evident bundles that can be independently verified and investigated
            offline.
          </p>
          <p className="mt-4 max-w-2xl text-sm leading-6 text-muted-foreground">
            Verification establishes the integrity and order of records presented in a bundle. It
            does not prove complete capture, model correctness, or external custody unless that
            evidence exists.
          </p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <a
              href="https://github.com/pcguest/atb/blob/main/docs/getting-started/quickstart.md"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex min-h-11 items-center justify-center gap-2 rounded-md bg-primary px-5 text-sm font-semibold text-primary-foreground transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
            >
              Start with the quickstart <ArrowRight className="h-4 w-4" aria-hidden="true" />
            </a>
            <a
              href="https://github.com/pcguest/atb"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex min-h-11 items-center justify-center rounded-md border border-border bg-card px-5 text-sm font-semibold text-foreground transition-colors hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              Inspect the source
            </a>
          </div>
        </div>

        <aside
          className="self-end rounded-xl border border-border bg-card p-5"
          aria-label="ATB trust boundary"
        >
          <div className="flex items-center gap-3 border-b border-border pb-4">
            <span className="grid h-9 w-9 place-items-center rounded-md border border-primary/30 bg-primary/10 text-primary">
              <ShieldCheck className="h-5 w-5" aria-hidden="true" />
            </span>
            <div>
              <p className="text-sm font-semibold">Evidence boundary</p>
              <p className="text-xs text-muted-foreground">Clear claims, inspectable records</p>
            </div>
          </div>
          <dl className="divide-y divide-border">
            <div className="grid grid-cols-[7.5rem_1fr] gap-3 py-4 text-sm">
              <dt className="text-muted-foreground">Integrity</dt>
              <dd className="font-medium">SHA-256 + RFC 8785 chain</dd>
            </div>
            <div className="grid grid-cols-[7.5rem_1fr] gap-3 py-4 text-sm">
              <dt className="text-muted-foreground">Review</dt>
              <dd className="font-medium">Incident → exact evidence</dd>
            </div>
            <div className="grid grid-cols-[7.5rem_1fr] gap-3 py-4 text-sm">
              <dt className="text-muted-foreground">Custody</dt>
              <dd className="font-medium">Local unless independently recorded</dd>
            </div>
          </dl>
          <div className="mt-1 flex items-center gap-2 rounded-md bg-muted/50 px-3 py-2 font-mono text-xs text-muted-foreground">
            <PackageCheck className="h-4 w-4 text-verified" aria-hidden="true" />
            atb verify run.atb/bundle.atb
          </div>
        </aside>
      </div>

      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <ol className="grid border-x border-t border-border bg-surface-1 sm:grid-cols-2 lg:grid-cols-4">
          {FLOW.map(([step, title, detail], index) => (
            <li key={title} className="border-b border-border p-5 lg:border-r lg:last:border-r-0">
              <div className="flex items-center gap-2">
                <span className="font-mono text-[11px] text-primary">{step}</span>
                {index === 2 ? (
                  <FileSearch className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
                ) : null}
                <h2 className="text-sm font-semibold">{title}</h2>
              </div>
              <p className="mt-2 text-sm leading-6 text-muted-foreground">{detail}</p>
            </li>
          ))}
        </ol>
      </div>
    </section>
  );
}
