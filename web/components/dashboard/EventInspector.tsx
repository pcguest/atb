"use client";

import { useEffect, useMemo, useRef, useState } from "react";

import { eventFamilyClass, eventSummary } from "@/lib/event-family";
import { HashValue } from "@/components/dashboard/HashValue";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/app/view/components/ui/tooltip";
import { collectMaskedPaths, setByPath } from "@/lib/pii";
import { copyTextToClipboard } from "@/lib/hash-display";
import type { EventRecord } from "@/lib/types";

type EventInspectorProps = {
  event: EventRecord | null;
  disabled?: boolean;
  onReveal: (seq: number, fieldPath: string) => Promise<unknown>;
};

// Reported attribution carried in an event's payload. ATB preserves these
// producer assertions; it does not verify that the principal acted or was
// authorised. Only fields the producer actually recorded are surfaced.
type ReportedAttribution = {
  principalType?: string;
  principalIdHash?: string;
  onBehalfOf?: string;
  identityProvider?: string;
  identitySubject?: string;
  assertionType?: string;
};

function parseReportedAttribution(data: Record<string, unknown> | undefined): ReportedAttribution | null {
  if (!data || typeof data !== "object") return null;
  const out: ReportedAttribution = {};
  const principal = data["principal"];
  if (principal && typeof principal === "object") {
    const p = principal as Record<string, unknown>;
    if (typeof p.type === "string") out.principalType = p.type;
    if (typeof p.id_hash === "string") out.principalIdHash = p.id_hash;
    if (typeof p.on_behalf_of === "string") out.onBehalfOf = p.on_behalf_of;
  }
  const identity = data["identity_evidence"];
  if (identity && typeof identity === "object") {
    const e = identity as Record<string, unknown>;
    if (typeof e.identity_provider === "string") out.identityProvider = e.identity_provider;
    if (typeof e.subject === "string") out.identitySubject = e.subject;
    if (typeof e.assertion_type === "string") out.assertionType = e.assertion_type;
  }
  if (!out.principalType && !out.principalIdHash && !out.identityProvider) return null;
  return out;
}

export function EventInspector({ event, disabled = false, onReveal }: EventInspectorProps) {
  const [revealed, setRevealed] = useState<Record<string, unknown>>({});
  const [pending, setPending] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const selectionKey = event ? `${event.seq}:${event.hash}` : "none";
  const activeSelection = useRef(selectionKey);
  activeSelection.current = selectionKey;

  useEffect(() => {
    setRevealed({});
    setPending(null);
    setError(null);
  }, [selectionKey]);

  const renderedData = useMemo(() => {
    if (!event) {
      return null;
    }
    let current: unknown = event.data;
    for (const [path, value] of Object.entries(revealed)) {
      current = setByPath(current, path, value);
    }
    return current;
  }, [event, revealed]);

  const maskedPaths = useMemo(() => {
    if (!renderedData) {
      return [];
    }
    return collectMaskedPaths(renderedData, "data");
  }, [renderedData]);

  const renderedRecord = useMemo(() => event ? { ...event, data: renderedData } : null, [event, renderedData]);
  const attribution = useMemo(
    () =>
      renderedData && typeof renderedData === "object"
        ? parseReportedAttribution(renderedData as Record<string, unknown>)
        : null,
    [renderedData],
  );

  if (!event) {
    return (
      <div className="rounded-lg border border-border bg-card p-4 text-sm text-muted-foreground">
        Select an event to inspect details.
      </div>
    );
  }

  async function handleReveal(path: string): Promise<void> {
    if (!event) {
      return;
    }
    const eventSeq = event.seq;
    const fieldPath = path.replace(/^data\./, "");
    const requestSelection = selectionKey;
    setPending(path);
    setError(null);
    try {
      const value = await onReveal(eventSeq, fieldPath);
      if (activeSelection.current !== requestSelection) return;
      setRevealed((prev) => ({ ...prev, [fieldPath]: value }));
    } catch (err) {
      if (activeSelection.current !== requestSelection) return;
      const message = err instanceof Error ? err.message : "Reveal failed";
      setError(message);
    } finally {
      if (activeSelection.current === requestSelection) setPending(null);
    }
  }

  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="space-y-4 p-3">
        <section aria-labelledby="record-summary-heading" className="space-y-2">
          <h4 id="record-summary-heading" className="text-sm font-semibold">Recorded event #{event.seq}</h4>
          {eventSummary(event.type, event.data) ? (
            <p data-testid="event-summary" className={`text-sm font-medium ${eventFamilyClass(event.type)}`}>
              {eventSummary(event.type, event.data)}
            </p>
          ) : <p className="text-sm text-muted-foreground">No concise human-readable summary was recorded for this event.</p>}
        </section>

        <details className="rounded border border-border p-3 text-xs" open>
          <summary className="cursor-pointer font-medium">Technical metadata</summary>
          <dl className="mt-3 grid gap-2 text-foreground">
            <div><dt className="inline text-muted-foreground">Event type: </dt><dd className={`inline font-medium ${eventFamilyClass(event.type)}`}>{event.type}</dd></div>
            <div><dt className="inline text-muted-foreground">Sequence: </dt><dd className="inline">{event.seq}</dd></div>
            {event.timestamp && <div><dt className="inline text-muted-foreground">Recorded timestamp: </dt><dd className="inline break-all">{event.timestamp}</dd></div>}
          </dl>
        </details>

        {(event.trace_id || event.span_id || event.parent_span_id) && <details className="rounded border border-border p-3 text-xs">
          <summary className="cursor-pointer font-medium">Source and provenance</summary>
          <dl className="mt-3 grid gap-2 break-all text-foreground">
            {event.trace_id && <div><dt className="inline text-muted-foreground">Trace: </dt><dd className="inline font-mono">{event.trace_id}</dd></div>}
            {event.span_id && <div><dt className="inline text-muted-foreground">Span: </dt><dd className="inline font-mono">{event.span_id}</dd></div>}
            {event.parent_span_id && <div><dt className="inline text-muted-foreground">Parent span: </dt><dd className="inline font-mono">{event.parent_span_id}</dd></div>}
          </dl>
        </details>}

        {attribution && (
          <details className="rounded border border-border p-3 text-xs" data-testid="attribution">
            <summary className="cursor-pointer font-medium">Attribution (reported by producer)</summary>
            <dl className="mt-3 grid gap-2 break-all text-foreground">
              {(attribution.principalType || attribution.principalIdHash) && (
                <div><dt className="inline text-muted-foreground">Acting principal: </dt><dd className="inline font-mono">{attribution.principalType ? `${attribution.principalType}` : "principal"}{attribution.principalIdHash ? `:${attribution.principalIdHash}` : ""}</dd></div>
              )}
              {(attribution.principalType || attribution.principalIdHash) && (
                <div><dt className="inline text-muted-foreground">On behalf of: </dt><dd className="inline font-mono">{attribution.onBehalfOf && attribution.onBehalfOf.length > 0 ? attribution.onBehalfOf : "Not reported"}</dd></div>
              )}
              {attribution.identityProvider && (
                <div><dt className="inline text-muted-foreground">Identity evidence: </dt><dd className="inline">{attribution.identityProvider}{attribution.identitySubject ? `/${attribution.identitySubject}` : ""}{attribution.assertionType ? ` (${attribution.assertionType})` : ""}</dd></div>
              )}
            </dl>
            <p className="mt-3 text-[10px] leading-4 text-muted-foreground">Reported by the producer and preserved in this bundle. This is an assertion recorded as evidence; it is not a verification that the principal acted, is who it claims, or was authorised. Authority and policy are interpreted elsewhere.</p>
          </details>
        )}

        {event.acquisition && (
          <details className="rounded border border-border p-3 text-xs" data-testid="acquisition-provenance">
            <summary className="cursor-pointer font-medium">Acquisition provenance</summary>
            <dl className="mt-3 grid gap-2 break-all text-foreground">
              {event.acquisition.mode && <div><dt className="inline text-muted-foreground">Acquisition mode: </dt><dd className="inline">{event.acquisition.mode}</dd></div>}
              {event.acquisition.source_system && <div><dt className="inline text-muted-foreground">Source system: </dt><dd className="inline">{event.acquisition.source_system}</dd></div>}
              {event.acquisition.source_record_id && <div><dt className="inline text-muted-foreground">Source record: </dt><dd className="inline font-mono">{event.acquisition.source_record_id}</dd></div>}
              {event.acquisition.source_timestamp && <div><dt className="inline text-muted-foreground">Original timestamp: </dt><dd className="inline">{event.acquisition.source_timestamp}</dd></div>}
              {event.acquisition.acquired_at && <div><dt className="inline text-muted-foreground">Acquired at: </dt><dd className="inline">{event.acquisition.acquired_at}</dd></div>}
              {event.acquisition.adapter && <div><dt className="inline text-muted-foreground">Adapter: </dt><dd className="inline font-mono">{event.acquisition.adapter}{event.acquisition.adapter_version ? ` v${event.acquisition.adapter_version}` : ""}</dd></div>}
              {event.acquisition.source_digest && <div><dt className="inline text-muted-foreground">Source digest: </dt><dd className="inline"><HashValue hash={event.acquisition.source_digest} className="text-foreground" /></dd></div>}
              <div><dt className="inline text-muted-foreground">Raw source: </dt><dd className="inline">{event.acquisition.raw_source_available ? "Retained in this bundle" : "Not retained; only the digest is recorded"}</dd></div>
              <div><dt className="inline text-muted-foreground">Checkpoint: </dt><dd className="inline">{event.acquisition.checkpoint_status === "recorded" ? `operational position ${event.acquisition.checkpoint_position ?? "recorded"}` : "not recorded"}</dd></div>
            </dl>
            <p className="mt-3 text-[10px] leading-4 text-muted-foreground">A source digest establishes source-representation identity and change, not truth. A checkpoint is operational state, not evidence truth. Retrospective means acquired after the original occurrence, not witnessed live.</p>
          </details>
        )}

        <details className="rounded border border-border p-3 text-xs" open>
          <summary className="cursor-pointer font-medium">Identifiers and hashes</summary>
          <dl className="mt-3 grid gap-2 text-foreground">
            <div><dt className="inline text-muted-foreground">Sequence: </dt><dd className="inline"><button type="button" aria-label={`Copy sequence ${event.seq}`} className="rounded-sm font-mono text-primary underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" onClick={() => void copyTextToClipboard(String(event.seq))}>#{event.seq}</button></dd></div>
            <div className="break-all"><dt className="inline text-muted-foreground">Record hash: </dt><dd className="inline"><HashValue hash={event.hash} className="text-foreground" /></dd></div>
            <div className="break-all"><dt className="inline text-muted-foreground">Previous hash: </dt><dd className="inline"><HashValue hash={event.prev_hash} className="text-foreground" /></dd></div>
          </dl>
        </details>

        {maskedPaths.length > 0 && (
          <div className="rounded border border-border bg-muted p-2">
            <div className="mb-2 text-xs uppercase tracking-wide text-muted-foreground">Masked Fields</div>
            <TooltipProvider>
              <div className="space-y-2">
                {maskedPaths.map((path) => (
                  <div key={path} className="space-y-0.5">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <button
                          type="button"
                          disabled={disabled || pending === path}
                          onClick={() => handleReveal(path)}
                          aria-label={`Reveal masked field ${path}`}
                          className="w-full rounded border border-border bg-muted px-2 py-1 text-left text-xs text-foreground hover:border-ring focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-60"
                        >
                          {pending === path ? "Revealing..." : `Click to Reveal: ${path}`}
                        </button>
                      </TooltipTrigger>
                      <TooltipContent side="left" className="max-w-xs">
                        <p>
                          Revealing writes a <span className="font-mono">privacy.reveal</span> event
                          to the independent <span className="font-mono">.reveals</span> sidecar,
                          which has its own hash chain. The authoritative bundle is never modified.
                        </p>
                      </TooltipContent>
                    </Tooltip>
                    <p className="px-0.5 font-mono text-[10px] text-amber-400/90">
                      writes privacy.reveal event to the .reveals sidecar
                    </p>
                  </div>
                ))}
              </div>
            </TooltipProvider>
          </div>
        )}

        {error && <div role="alert" className="text-xs text-red-300">{error}</div>}

        <details className="rounded border border-border" open>
          <summary className="cursor-pointer px-3 py-2 text-sm font-medium">Rendered record (projection)</summary>
          <p className="border-t border-border px-3 pt-2 text-[10px] leading-4 text-muted-foreground">
            This is the display/API projection of the record, formatted for reading. It is not the exact
            canonical event bytes used for hashing, and any revealed fields reflect a privacy-reveal
            overlay rather than the stored record. The record hash above is the stable identity.
          </p>
          <pre tabIndex={0} className="max-h-[380px] overflow-auto bg-muted p-3 text-xs text-foreground">
            {JSON.stringify(renderedRecord, null, 2)}
          </pre>
        </details>
      </div>
    </div>
  );
}
