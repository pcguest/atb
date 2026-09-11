import Image from "next/image";

const STEPS = [
  ["Finding", "See what the captured evidence supports, and where the conclusion stops."],
  ["Timeline", "Follow recorded event order, with timestamps only where available."],
  [
    "Exact evidence",
    "Inspect source records, hashes and masked fields. Reveal is audited separately.",
  ],
  ["Trust", "Review integrity, profile coverage and corroboration as separate questions."],
];

export default function Features() {
  return (
    <section id="features" className="border-b border-border bg-background py-12 sm:py-16">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <header className="mb-6 max-w-3xl">
          <p className="font-mono text-xs font-semibold uppercase tracking-widest text-primary">
            Inside ATB View
          </p>
          <h2 className="mt-3 text-3xl font-semibold tracking-tight">
            From a finding to the record behind it.
          </h2>
          <p className="mt-4 text-base leading-7 text-muted-foreground">
            When an agent action needs explaining, a log line is only a starting point. Investigate
            the recorded sequence and inspect the evidence that supports each conclusion.
          </p>
        </header>
        <figure className="overflow-hidden rounded-lg border border-border bg-surface-sidebar">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 py-3 text-sm">
            <span className="font-medium">ATB View · Local investigation</span>
            <span className="text-xs text-muted-foreground">
              Recorded evidence · independently verifiable
            </span>
          </div>
          <a
            href="/product/investigation.png"
            className="block focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
            aria-label="Open the full-size ATB View investigation screenshot"
          >
            <Image
              src="/product/investigation.png"
              alt="ATB View investigating the recorded incident, with a selected finding and its supporting evidence."
              width={1280}
              height={720}
              unoptimized
              className="h-auto w-full"
            />
          </a>
          <figcaption className="border-t border-border px-4 py-3 text-xs leading-5 text-muted-foreground">
            Actual ATB View, using the repository’s deterministic incident example. Open the image
            to inspect at full size.
          </figcaption>
        </figure>
        <ol className="mt-6 grid gap-0 divide-y divide-border border-y border-border sm:grid-cols-2 sm:divide-y-0 lg:grid-cols-4">
          {STEPS.map(([title, detail], index) => (
            <li key={title} className="px-4 py-5 first:pl-0">
              <h3 className="text-sm font-semibold">
                <span className="mr-2 font-mono text-xs text-primary">0{index + 1}</span>
                {title}
              </h3>
              <p className="mt-2 text-sm leading-6 text-muted-foreground">{detail}</p>
            </li>
          ))}
        </ol>
      </div>
    </section>
  );
}
