"use client";

import { Command, Search, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";

export type PaletteAction = {
  id: string;
  label: string;
  hint?: string;
  run: () => void | Promise<void>;
};

export function CommandPalette({ actions }: { actions: PaletteAction[] }) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setOpen((current) => !current);
      }
      if (event.key === "Escape") {
        setOpen(false);
        setQuery("");
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  useEffect(() => {
    if (open) inputRef.current?.focus();
  }, [open]);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return needle
      ? actions.filter((action) => action.label.toLowerCase().includes(needle))
      : actions;
  }, [actions, query]);

  async function run(action: PaletteAction) {
    setOpen(false);
    setQuery("");
    await action.run();
  }

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="inline-flex h-9 items-center gap-2 rounded-md border border-border bg-card px-3 text-sm text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        aria-label="Open command palette"
      >
        <Command className="h-4 w-4" aria-hidden="true" />
        <span>Commands</span>
        <kbd className="rounded border border-border px-1.5 font-mono text-[10px]">⌘K</kbd>
      </button>
      {open && (
        <div
          className="fixed inset-0 z-50 flex items-start justify-center bg-background/80 px-4 pt-[12vh] backdrop-blur-sm"
          role="dialog"
          aria-modal="true"
          aria-label="Command palette"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) {
              setOpen(false);
              setQuery("");
            }
          }}
        >
          <div className="w-full max-w-xl overflow-hidden rounded-lg border border-border bg-popover shadow-2xl">
            <div className="flex items-center border-b border-border px-3">
              <Search className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
              <input
                ref={inputRef}
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Find an ATB operation or investigation view…"
                className="h-12 flex-1 bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground"
                aria-label="Filter commands"
              />
              <button
                type="button"
                onClick={() => {
                  setOpen(false);
                  setQuery("");
                }}
                className="rounded p-2 text-muted-foreground hover:bg-muted hover:text-foreground"
                aria-label="Close command palette"
              >
                <X className="h-4 w-4" aria-hidden="true" />
              </button>
            </div>
            <ul className="max-h-80 overflow-y-auto p-2">
              {filtered.map((action) => (
                <li key={action.id}>
                  <button
                    type="button"
                    onClick={() => void run(action)}
                    className="flex w-full items-center justify-between rounded-md px-3 py-2.5 text-left text-sm hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    <span>{action.label}</span>
                    {action.hint && (
                      <span className="font-mono text-xs text-muted-foreground">{action.hint}</span>
                    )}
                  </button>
                </li>
              ))}
              {filtered.length === 0 && (
                <li className="px-3 py-8 text-center text-sm text-muted-foreground">
                  No matching command
                </li>
              )}
            </ul>
          </div>
        </div>
      )}
    </>
  );
}
