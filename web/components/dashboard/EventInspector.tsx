"use client";

import { useEffect, useMemo, useState } from "react";

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
      <div className="rounded-lg border border-slate-800 bg-slate-900 p-4 text-sm text-slate-200">
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
    <div className="rounded-lg border border-slate-800 bg-slate-950">
      <div className="border-b border-slate-800 px-3 py-2 text-xs uppercase tracking-wide text-slate-200">
        Inspector
      </div>
      <div className="space-y-3 p-3">
        <div className="grid gap-2 text-xs text-slate-200">
          <div>
            <span className="text-slate-300">type:</span>{" "}
            <span className="text-slate-200">{event.type}</span>
          </div>
          <div>
            <span className="text-slate-300">seq:</span>{" "}
            <span className="text-slate-200">{event.seq}</span>
          </div>
          <div className="break-all">
            <span className="text-slate-300">hash:</span>{" "}
            <span className="text-slate-200">{event.hash}</span>
          </div>
        </div>

        {maskedPaths.length > 0 && (
          <div className="rounded border border-slate-800 bg-slate-900 p-2">
            <div className="mb-2 text-xs uppercase tracking-wide text-slate-200">Masked Fields</div>
            <div className="space-y-1">
              {maskedPaths.map((path) => (
                <button
                  key={path}
                  type="button"
                  disabled={disabled || pending === path}
                  onClick={() => handleReveal(path)}
                  className="w-full rounded border border-slate-700 bg-slate-800 px-2 py-1 text-left text-xs text-slate-100 hover:border-slate-600 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {pending === path ? "Revealing..." : `Click to Reveal: ${path}`}
                </button>
              ))}
            </div>
          </div>
        )}

        {error && <div className="text-xs text-red-300">{error}</div>}

        <pre className="max-h-[380px] overflow-auto rounded bg-slate-900 p-3 text-xs text-slate-200">
          {JSON.stringify(renderedData, null, 2)}
        </pre>
      </div>
    </div>
  );
}
