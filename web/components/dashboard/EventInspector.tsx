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
import type { EventRecord } from "@/lib/types";

type EventInspectorProps = {
  event: EventRecord | null;
  disabled?: boolean;
  onReveal: (seq: number, fieldPath: string) => Promise<unknown>;
};

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

  const canonicalRecord = useMemo(() => event ? { ...event, data: renderedData } : null, [event, renderedData]);

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
            <div><dt className="inline text-muted-foreground">Canonical event type: </dt><dd className={`inline font-medium ${eventFamilyClass(event.type)}`}>{event.type}</dd></div>
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

        <details className="rounded border border-border p-3 text-xs" open>
          <summary className="cursor-pointer font-medium">Identifiers and hashes</summary>
          <dl className="mt-3 grid gap-2 text-foreground">
            <div><dt className="inline text-muted-foreground">Sequence: </dt><dd className="inline"><button type="button" aria-label={`Copy sequence ${event.seq}`} className="rounded-sm font-mono text-primary underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" onClick={() => void navigator.clipboard?.writeText(String(event.seq))}>#{event.seq}</button></dd></div>
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
          <summary className="cursor-pointer px-3 py-2 text-sm font-medium">Canonical record</summary>
          <pre className="max-h-[380px] overflow-auto border-t border-border bg-muted p-3 text-xs text-foreground">
            {JSON.stringify(canonicalRecord, null, 2)}
          </pre>
        </details>
      </div>
    </div>
  );
}
