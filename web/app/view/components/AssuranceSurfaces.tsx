"use client";

import { useState } from "react";
import { TraceGraph } from "@/components/dashboard/TraceGraph";
import { CopyAction, EmptyState, StatusBadge } from "./ui/investigation";
import type {
  InvestigationContext,
  InvestigationRelationships,
  InvestigationTrust,
  ProfileReportSummary,
  TimelineEvent,
} from "@/lib/types";

const control =
  "rounded-md border border-border px-3 py-2 text-sm hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";
const row =
  "w-full border-b border-border p-3 text-left text-sm hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring";
const human = (value: string) => value.replaceAll("_", " ");
const operationLabel: Record<string, string> = {
  select: "Selected context",
  transform: "Transformed context",
  compact: "Compacted context",
  cache_reuse: "Reused cached context",
  assemble: "Assembled context",
};
const evidenceState = (value: string) =>
  !value || value.toLowerCase() === "absent" ? "Absent from recorded evidence" : human(value);

function RecordLink({
  seq,
  onOpenEvidence,
}: {
  seq: number;
  onOpenEvidence: (seq: number) => void;
}) {
  return (
    <button type="button" className={control} onClick={() => onOpenEvidence(seq)}>
      Open evidence #{seq}
    </button>
  );
}

export function ContextSurface({
  data,
  onOpenEvidence,
}: {
  data: InvestigationContext;
  onOpenEvidence: (seq: number) => void;
}) {
  const [selected, setSelected] = useState<string | null>(null);
  const entries = [
    ...data.lineage.units.map((unit) => ({
      key: `unit:${unit.id}`,
      title: unit.source || unit.id,
      kind: human(unit.kind),
      seq: unit.event_sequence,
      record: unit,
    })),
    ...data.lineage.operations.map((operation) => ({
      key: `operation:${operation.id}`,
      title: operation.id,
      kind: operationLabel[operation.type] || human(operation.type),
      seq: operation.event_sequence,
      record: operation,
    })),
    ...data.capabilities.map((capability, index) => ({
      key: `capability:${index}`,
      title: capability.raw_event_type,
      kind: human(capability.name),
      seq: capability.event_sequence,
      record: capability,
    })),
  ].sort((a, b) => a.seq - b.seq);
  const active = entries.find((entry) => entry.key === selected) ?? entries[0];
  const noLineage = data.lineage.units.length === 0 && data.lineage.operations.length === 0;
  return (
    <section className="space-y-4" aria-label="Context evidence">
      <h2 className="text-xl font-semibold tracking-tight">Context evidence</h2>
      <p className="text-sm text-muted-foreground">
        Observable sources and transformations only. ATB does not store hidden model reasoning.
        “Supplied to model” applies only when an invocation bind is proven.
      </p>
      {noLineage && (
        <div className="rounded-lg border border-border bg-card p-4">
          <h2 className="font-semibold">No structured context lineage</h2>
          <p className="mt-2 text-sm text-muted-foreground">
            <span>
              {data.capabilities.length
                ? "Retrieval was recorded; structured lineage units were not."
                : "No context lineage units or retrieval capabilities were recorded."}
            </span>{" "}
            This does not prove retrieval did not occur.
          </p>
          <dl className="mt-4 grid gap-4 text-sm sm:grid-cols-2">
            <div>
              <dt className="font-medium">What would appear here</dt>
              <dd className="mt-1 text-muted-foreground">
                Recorded context sources, parent references, selection, transformations and
                invocation references.
              </dd>
            </div>
            <div>
              <dt className="font-medium">What was checked</dt>
              <dd className="mt-1 text-muted-foreground">
                Structured lineage units, operations and mapped retrieval capabilities in this
                bundle. This lineage response does not establish invocation binding.
              </dd>
            </div>
          </dl>
          <p className="mt-3 text-sm text-muted-foreground">
            Inspect related records below, or open Evidence from the investigation navigation.
          </p>
        </div>
      )}
      {data.lineage.warnings.length > 0 && (
        <div className="rounded-lg border border-warning/40 bg-card p-4">
          <h3 className="text-sm font-medium">Lineage qualifications</h3>
          <ul className="mt-2 space-y-2 text-sm text-muted-foreground">
            {data.lineage.warnings.map((warning, index) => (
              <li key={index}>{warning}</li>
            ))}
          </ul>
        </div>
      )}
      {active && (
        <div className="grid overflow-hidden rounded-lg border border-border bg-card xl:grid-cols-[minmax(16rem,2fr)_minmax(0,3fr)]">
          <div
            className="max-h-[36rem] overflow-auto border-b border-border xl:border-b-0 xl:border-r"
            aria-label="Recorded context"
          >
            <h2 className="border-b border-border p-3 text-sm font-semibold">
              <span>
                {noLineage
                  ? "Recorded retrieval capabilities"
                  : "Sources and operations"}
              </span>{" "}
              · {entries.length}
            </h2>
            {entries.map((entry) => (
              <button
                type="button"
                key={entry.key}
                onClick={() => setSelected(entry.key)}
                aria-current={active.key === entry.key ? "true" : undefined}
                className={`${row} ${active.key === entry.key ? "bg-primary/10" : ""}`}
              >
                <span className="block font-medium">{entry.kind}</span>
                <span className="mt-1 block break-all font-mono text-xs text-muted-foreground">
                  #{entry.seq} · {entry.title}
                </span>
              </button>
            ))}
          </div>
          <article className="min-w-0 space-y-4 p-4">
            <div>
              <h3 className="font-semibold">{active.kind}</h3>
              <p className="mt-1 break-all font-mono text-xs text-muted-foreground">
                {active.title}
              </p>
            </div>
            <RecordLink seq={active.seq} onOpenEvidence={onOpenEvidence} />
            {"parent_ids" in active.record && (
              <dl className="space-y-3 text-sm">
                <div>
                  <dt className="text-muted-foreground">Parent references</dt>
                  <dd className="break-all font-mono text-xs">
                    {active.record.parent_ids.join(", ") || "None recorded"}
                  </dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Digest</dt>
                  <dd className="mt-1 break-all font-mono text-xs">{active.record.digest}</dd>
                </div>
              </dl>
            )}
            {"input_unit_ids" in active.record && (
              <>
                <dl className="space-y-3 text-sm">
                  {[
                    ["Input units", active.record.input_unit_ids.join(", ")],
                    ["Output units", active.record.output_unit_ids.join(", ")],
                    ["Invocation reference", active.record.invocation_id],
                  ].map(([label, value]) => (
                    <div key={label}>
                      <dt className="text-muted-foreground">{label}</dt>
                      <dd className="break-all font-mono text-xs">{value || "Not recorded"}</dd>
                    </div>
                  ))}
                </dl>
                <p className="rounded-md border border-border bg-muted/40 p-3 text-sm text-muted-foreground">
                  {active.record.invocation_id
                    ? `Bound to recorded invocation ${active.record.invocation_id}.`
                    : "No invocation binding was recorded for this operation; it does not establish that context was supplied to a model."}
                </p>
              </>
            )}
            <details>
              <summary className="cursor-pointer text-sm">Recorded metadata</summary>
              <pre tabIndex={0} className="mt-3 max-h-80 overflow-auto rounded-md bg-muted p-3 text-xs">
                {JSON.stringify(active.record, null, 2)}
              </pre>
            </details>
          </article>
        </div>
      )}
    </section>
  );
}

export function RelationshipsSurface({
  data,
  timeline,
  onOpenEvidence,
}: {
  data: InvestigationRelationships;
  timeline: TimelineEvent[];
  onOpenEvidence: (seq: number) => void;
}) {
  const [selected, setSelected] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [graphOpen, setGraphOpen] = useState(false);
  const filtered = data.relationships.filter((relationship) =>
    `${relationship.source_seq} ${relationship.target_seq} ${relationship.kind} ${relationship.evidence_value}`
      .toLowerCase()
      .includes(query.toLowerCase()),
  );
  const active = filtered.find((relationship) => relationship.id === selected) ?? filtered[0];
  const graph = (() => {
    const seqs = [
      ...new Set(
        filtered.flatMap((relationship) => [relationship.source_seq, relationship.target_seq]),
      ),
    ].sort((a, b) => a - b);
    return {
      nodes: seqs.map((seq) => {
        const event = timeline.find((item) => item.seq === seq);
        return {
          id: `evt-${seq}`,
          label: `#${seq} · ${event?.label || event?.type || "Recorded event"}`,
          type: "event",
          event_type: event?.type || "event",
        };
      }),
      edges: filtered.map((relationship) => ({
        id: relationship.id,
        source: `evt-${relationship.source_seq}`,
        target: `evt-${relationship.target_seq}`,
        label: human(relationship.kind),
      })),
    };
  })();
  return (
    <section className="space-y-4" aria-label="Relationships">
      <p className="text-sm text-muted-foreground">
        Shared identifiers derived from recorded fields. These rows do not assert causation.
      </p>
      <input
        aria-label="Filter relationships"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder="Filter identifiers, relation or sequence…"
        className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      />
      {!active ? (
        <EmptyState title={query ? "No matching relationships" : "No derived relationships"}>
          {query
            ? "Change the filter to inspect other recorded relationships."
            : "No shared-identifier relationships were derived."}
        </EmptyState>
      ) : (
        <div className="grid overflow-hidden rounded-lg border border-border bg-card xl:grid-cols-[minmax(0,3fr)_minmax(16rem,2fr)]">
          <div className="min-w-0 overflow-auto">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-border text-xs text-muted-foreground">
                <tr>
                  {["Source", "Relation", "Target", "Shared identifier / basis"].map((heading) => (
                    <th key={heading} className="p-3 font-medium">
                      {heading}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {filtered.map((relationship) => (
                  <tr
                    key={relationship.id}
                    className={`border-b border-border ${active.id === relationship.id ? "bg-primary/10" : ""}`}
                  >
                    <td className="p-3">
                      <button
                        type="button"
                        className={control}
                        aria-current={active.id === relationship.id ? "true" : undefined}
                        onClick={() => setSelected(relationship.id)}
                      >
                        Inspect relationship
                      </button>
                      <button
                        type="button"
                        className="mt-2 block font-mono text-xs underline underline-offset-2"
                        onClick={() => onOpenEvidence(relationship.source_seq)}
                      >
                        Source #{relationship.source_seq}
                      </button>
                    </td>
                    <td className="p-3">{human(relationship.kind)}</td>
                    <td className="p-3">
                      <button
                        type="button"
                        className="font-mono text-xs underline underline-offset-2"
                        onClick={() => onOpenEvidence(relationship.target_seq)}
                      >
                        Target #{relationship.target_seq}
                      </button>
                    </td>
                    <td className="max-w-64 break-all p-3 font-mono text-xs">
                      {relationship.evidence_value}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <article className="min-w-0 space-y-4 border-t border-border p-4 xl:border-l xl:border-t-0">
            <h2 className="font-semibold">{human(active.kind)}</h2>
            <p className="text-sm text-muted-foreground">
              Derived shared identifier · {active.strength}
            </p>
            <p className="break-all font-mono text-xs">{active.evidence_value}</p>
            <CopyAction value={active.evidence_value} label="Copy shared identifier" />
            <div className="flex flex-wrap gap-2">
              <RecordLink seq={active.source_seq} onOpenEvidence={onOpenEvidence} />
              <RecordLink seq={active.target_seq} onOpenEvidence={onOpenEvidence} />
            </div>
            <p className="text-xs text-muted-foreground">
              Inspect both source records to establish what the shared identifier supports. This
              relationship does not establish cause or temporal order.
            </p>
          </article>
        </div>
      )}
      <div className="rounded-lg border border-border bg-card">
        <button type="button" className="w-full cursor-pointer p-3 text-left text-sm font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" aria-expanded={graphOpen} onClick={() => setGraphOpen((open) => !open)}>
          Open optional graph{" "}
          <span className="font-normal text-muted-foreground">
            · shared identifiers, not causation
          </span>
        </button>
        {graphOpen && <><p className="px-4 pb-3 text-xs text-muted-foreground">
          Shows the filtered relationship rows. Node placement is a visual arrangement; consult
          Timeline for recorded order.
        </p>
        <div className="h-[420px] border-t border-border">
          <TraceGraph graph={graph} onSelectSeq={onOpenEvidence} />
        </div></>}
      </div>
    </section>
  );
}

export function TrustSurface({
  data,
  profile,
  timeline,
  onOpenEvidence,
}: {
  data: InvestigationTrust;
  profile?: ProfileReportSummary | null;
  timeline: TimelineEvent[];
  onOpenEvidence: (seq: number) => void;
}) {
  const [selected, setSelected] = useState("Integrity");
  const questions = [
    {
      name: "Integrity",
      question: "Is recorded evidence intact?",
      status: data.integrity_valid ? "Hash chain verified" : "Hash chain failed",
      reason:
        "Canonical hashes and sequence are checked against the recorded bundle. Coverage cannot repair a broken chain.",
      source: "Bundle verification result",
      gap: data.integrity_valid
        ? "Integrity does not establish complete capture or truth of recorded claims."
        : "Event-derived evidence cannot be treated as verified.",
      tone: data.integrity_valid ? ("verified" as const) : ("danger" as const),
    },
    {
      name: "Coverage",
      question: "Does the selected evidence profile pass?",
      status: !data.integrity_valid ? "Untrusted" : !data.profile_id ? "Not assessed" : data.profile_pass ? "Pass" : "Does not pass",
      reason:
        "Profile-scoped completeness of recorded evidence, not proof that everything was captured.",
      source: data.profile_id || "No selected profile recorded",
      gap: "A passing profile does not establish universal capture completeness.",
      tone: !data.integrity_valid ? ("danger" as const) : ("unknown" as const),
    },
    {
      name: "Corroboration",
      question: "Is external or organisational custody recorded?",
      status: data.external_corroboration
        ? "External evidence present"
        : "Absent from recorded evidence",
      reason: "Independent records outside this bundle. Absence is not a fail score.",
      source: data.external_corroboration
        ? "Recorded corroboration event(s) and custody assessment"
        : "No recorded corroboration event",
      gap: data.external_corroboration
        ? data.custody_state
        : "Absent from recorded evidence. ATB cannot infer external custody from this bundle.",
      tone: "unknown" as const,
    },
  ];
  const active = questions.find((question) => question.name === selected) ?? questions[0];
  const related = timeline.filter((event) => event.family === "corroboration");
  return (
    <section className="space-y-4" aria-label="Trust">
      <div className="rounded-lg border border-border bg-card p-4">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          What ATB proves
        </h2>
        <p className="mt-2 text-lg font-semibold">{data.proof_statement}</p>
      </div>
      <p className="text-sm text-muted-foreground">
        Three independent questions. Integrity, profile coverage, and external custody are not
        collapsed into one score.
      </p>
      <div className="grid overflow-hidden rounded-lg border border-border bg-card lg:grid-cols-[minmax(15rem,2fr)_minmax(0,3fr)]">
        <div className="border-b border-border lg:border-b-0 lg:border-r">
          {questions.map((question) => (
            <button
              type="button"
              key={question.name}
              className={`${row} space-y-2 ${active.name === question.name ? "bg-primary/10" : ""}`}
              aria-current={active.name === question.name ? "true" : undefined}
              onClick={() => setSelected(question.name)}
            >
              <span className="block font-semibold">{question.name}</span>
              <span className="block text-xs text-muted-foreground">{question.question}</span>
              <StatusBadge tone={question.tone}>{question.status}</StatusBadge>
            </button>
          ))}
        </div>
        <article className="space-y-4 p-4">
          <h3 className="font-semibold">{active.question}</h3>
          <dl className="space-y-4 text-sm">
            {[
              ["Status", active.status],
              ["Why", active.reason],
              ["Source", active.source],
              ["Boundary / gap", active.gap],
            ].map(([label, value]) => (
              <div key={label}>
                <dt className="font-medium">{label}</dt>
                <dd className="mt-1 text-muted-foreground">{value}</dd>
              </div>
            ))}
          </dl>
          {active.name === "Coverage" && profile && (
            <div className="space-y-3 text-sm">
              <p>
                Profile {profile.profile_id}
                {profile.profile_version ? ` · version ${profile.profile_version}` : ""} ·{" "}
                {!data.integrity_valid ? "Untrusted" : profile.pass ? "Pass" : "Does not pass"}
              </p>
              {profile.critical_failures.map((failure, index) => (
                <p key={index} className="text-muted-foreground">
                  {human(failure.kind)}: {failure.detail}
                </p>
              ))}
              {profile.warnings.map((warning, index) => (
                <p key={index} className="text-muted-foreground">
                  {warning}
                </p>
              ))}
              {Object.entries(profile.dimension_assessments).map(([name, assessment]) => (
                <p key={name}>
                  <span className="font-medium">{human(name)}</span> ·{" "}
                  {assessment.assessable ? "Assessed" : "Not assessable"}
                  {assessment.reason ? ` — ${assessment.reason}` : ""}
                </p>
              ))}
              {profile.provability_gaps.map((gap, index) => (
                <div key={index} className="border-t border-border pt-3">
                  <p className="font-medium">{gap.gap}</p>
                  <p className="mt-1 text-muted-foreground">
                    {gap.layer} · {gap.mitigation}
                  </p>
                  <p className="mt-1 text-muted-foreground">Closure condition: {gap.closed_when}</p>
                </div>
              ))}
            </div>
          )}
          {active.name === "Corroboration" && data.integrity_valid && related.length > 0 && (
            <div className="space-y-2">
              <p className="text-sm font-medium">Open supporting evidence</p>
              <p className="text-xs text-muted-foreground">
                These records are inspection paths; their presence alone does not prove independent
                custody.
              </p>
              <div className="flex flex-wrap gap-2">
                {related.map((event) => (
                  <RecordLink key={event.seq} seq={event.seq} onOpenEvidence={onOpenEvidence} />
                ))}
              </div>
            </div>
          )}
          <p className="text-xs text-muted-foreground">
            Use Verify bundle in Commands to repeat verification. Inspect exact records through
            Evidence.
          </p>
        </article>
      </div>
      <dl className="grid gap-px overflow-hidden rounded-lg border border-border bg-border sm:grid-cols-2 xl:grid-cols-4">
        {[
          ["Signature", data.signature_status, "Verifier signature assessment"],
          ["Anchor", data.anchor_status, "Verifier anchoring assessment"],
          ["Custody", data.custody_state, "Recorded custody assessment"],
          ["Canonicalisation", data.canonicalisation, "Bundle canonicalisation contract"],
        ].map(([label, value, source]) => (
          <div key={label} className="min-w-0 bg-card p-3">
            <dt className="text-xs text-muted-foreground">{label}</dt>
            <dd className="mt-1 break-words text-sm font-medium">{evidenceState(value)}</dd>
            <dd className="mt-2 text-xs text-muted-foreground">Source: {source}</dd>
          </div>
        ))}
      </dl>
      <div className="rounded-lg border border-border bg-card p-4">
        <h2 className="text-sm font-semibold">What ATB does not prove</h2>
        <ul className="mt-3 grid gap-2 text-sm text-muted-foreground md:grid-cols-2">
          {data.limitations.map((limitation) => (
            <li key={limitation}>{limitation}</li>
          ))}
        </ul>
      </div>
    </section>
  );
}
