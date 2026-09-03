import { Boxes, FileCheck2, FileSearch, Fingerprint, PackageOpen, ShieldCheck } from "lucide-react";

const CAPABILITIES = [
  {
    icon: Boxes,
    title: "Portable capture",
    detail: "Go, Python, and TypeScript producers write the same local NDJSON bundle format.",
    command: "atb capture run",
  },
  {
    icon: Fingerprint,
    title: "Independent verification",
    detail:
      "Hash-chain verification detects changed, reordered, or removed records in the presented bundle.",
    command: "atb verify",
  },
  {
    icon: FileSearch,
    title: "Forensic investigation",
    detail: "Start with bounded findings, follow the timeline, then inspect exact source records.",
    command: "atb view",
  },
  {
    icon: ShieldCheck,
    title: "Explicit trust limits",
    detail:
      "Integrity, profile coverage, and external corroboration stay separate—never one health score.",
    command: "Trust",
  },
  {
    icon: FileCheck2,
    title: "Audited reveal",
    detail:
      "Masked fields can be revealed individually; the action is recorded in a separate sidecar.",
    command: "privacy.reveal",
  },
  {
    icon: PackageOpen,
    title: "Deterministic export",
    detail:
      "Create incident and evidence packs for repeatable offline review without claiming certification.",
    command: "atb export",
  },
];

export default function Features() {
  return (
    <section id="features" className="border-b border-border bg-background py-20">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="grid gap-8 lg:grid-cols-[0.7fr_1.3fr]">
          <header className="max-w-md">
            <p className="font-mono text-xs font-semibold uppercase tracking-[0.16em] text-primary">
              Shipped capability
            </p>
            <h2 className="mt-3 text-3xl font-semibold tracking-tight text-foreground">
              Evidence work, end to end.
            </h2>
            <p className="mt-4 text-base leading-7 text-muted-foreground">
              ATB stays focused on the artefact: capture it, verify it, investigate it, and hand it
              off with honest boundaries.
            </p>
          </header>
          <ul className="overflow-hidden rounded-xl border border-border bg-card sm:grid sm:grid-cols-2">
            {CAPABILITIES.map(({ icon: Icon, title, detail, command }, index) => (
              <li
                key={title}
                className={`p-5 ${index < 4 ? "border-b border-border" : ""} ${index % 2 === 0 ? "sm:border-r sm:border-border" : ""}`}
              >
                <div className="flex items-start gap-3">
                  <span className="grid h-9 w-9 shrink-0 place-items-center rounded-md bg-muted text-primary">
                    <Icon className="h-4 w-4" aria-hidden="true" />
                  </span>
                  <div>
                    <h3 className="text-sm font-semibold">{title}</h3>
                    <p className="mt-1.5 text-sm leading-6 text-muted-foreground">{detail}</p>
                    <code className="mt-3 inline-block rounded bg-muted px-2 py-1 text-[11px] text-foreground">
                      {command}
                    </code>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  );
}
