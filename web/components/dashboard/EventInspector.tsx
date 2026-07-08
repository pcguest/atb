"use client";

import { useEffect, useMemo, useState } from "react";

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

  useEffect(() => {
    setRevealed({});
    setPending(null);
    setError(null);
  }, [event?.seq]);

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
    setPending(path);
    setError(null);
    try {
      const value = await onReveal(eventSeq, fieldPath);
      setRevealed((prev) => ({ ...prev, [path]: value }));
    } catch (err) {
      const message = err instanceof Error ? err.message : "Reveal failed";
      setError(message);
    } finally {
      setPending(null);
    }
  }

  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="space-y-3 p-3">
        <div className="grid gap-2 text-xs text-foreground">
          <div>
            <span className="text-muted-foreground">type:</span>{" "}
            <span className={`font-medium ${eventFamilyClass(event.type)}`}>{event.type}</span>
          </div>
          {eventSummary(event.type, event.data) && (
            <div data-testid="event-summary">
              <span className="text-muted-foreground">summary:</span>{" "}
              <span className={`font-medium ${eventFamilyClass(event.type)}`}>
                {eventSummary(event.type, event.data)}
              </span>
            </div>
          )}
          <div>
            <span className="text-muted-foreground">seq:</span>{" "}
            <span className="text-foreground">{event.seq}</span>
          </div>
          <div className="break-all">
            <span className="text-muted-foreground">hash:</span>{" "}
            <HashValue hash={event.hash} className="text-foreground" />
          </div>
        </div>

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
                          className="w-full rounded border border-border bg-muted px-2 py-1 text-left text-xs text-foreground hover:border-ring disabled:cursor-not-allowed disabled:opacity-60"
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

        {error && <div className="text-xs text-red-300">{error}</div>}

        <pre className="max-h-[380px] overflow-auto rounded bg-muted p-3 text-xs text-foreground">
          {JSON.stringify(renderedData, null, 2)}
        </pre>
      </div>
    </div>
  );
}
