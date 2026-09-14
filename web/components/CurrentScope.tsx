export default function CurrentScope() {
  return (
    <section id="scope" className="border-b border-border bg-surface-sidebar py-12 sm:py-16">
      <div className="mx-auto grid max-w-7xl gap-8 px-4 sm:px-6 lg:grid-cols-[1fr_1.4fr] lg:px-8">
        <header>
          <p className="font-mono text-xs font-semibold uppercase tracking-widest text-primary">
            A precise boundary
          </p>
          <h2 className="mt-3 text-3xl font-semibold tracking-tight">
            Know what the evidence establishes.
          </h2>
          <p className="mt-4 text-sm leading-7 text-muted-foreground">
            ATB is open source and works locally, without Mortise or Tenon cloud infrastructure.
          </p>
        </header>
        <div>
          <dl className="divide-y divide-border border-y border-border text-sm">
            <div className="grid gap-2 py-4 sm:grid-cols-[9rem_1fr]">
              <dt className="font-semibold">Integrity</dt>
              <dd className="leading-6 text-muted-foreground">
                Verify the hash chain of the presented records. This does not establish complete
                capture or the truth of a recorded claim.
              </dd>
            </div>
            <div className="grid gap-2 py-4 sm:grid-cols-[9rem_1fr]">
              <dt className="font-semibold">Coverage</dt>
              <dd className="leading-6 text-muted-foreground">
                Inspect evidence against a selected profile. Profile results do not certify
                compliance or model correctness.
              </dd>
            </div>
            <div className="grid gap-2 py-4 sm:grid-cols-[9rem_1fr]">
              <dt className="font-semibold">Custody</dt>
              <dd className="leading-6 text-muted-foreground">
                Local custody is explicit. Independent corroboration requires recorded external
                evidence. Mortise may add organisational custody.
              </dd>
            </div>
          </dl>
          <p className="mt-4 text-xs leading-6 text-muted-foreground">
            Tenon supplies the family’s assurance language. Shared workspaces, hosted storage and
            enterprise administration are outside the local ATB product.
          </p>
        </div>
      </div>
    </section>
  );
}
