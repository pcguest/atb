"use client";

import { Command, Search, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";

export type PaletteGroup = "navigate" | "operate" | "copy";

export type PaletteAction = {
  id: string;
  label: string;
  hint?: string;
  group?: PaletteGroup;
  run: () => void | Promise<void>;
};

const GROUP_ORDER: PaletteGroup[] = ["operate", "navigate", "copy"];
const GROUP_LABEL: Record<PaletteGroup, string> = {
  operate: "Operate",
  navigate: "Navigate",
  copy: "Copy",
};

export function CommandPalette({ actions }: { actions: PaletteAction[] }) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const closeRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setOpen((current) => !current);
        setActiveIndex(0);
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

  const resolvedIndex =
    filtered.length === 0 ? -1 : Math.min(Math.max(activeIndex, 0), filtered.length - 1);

  const grouped = useMemo(() => {
    return GROUP_ORDER.map((group) => ({
      group,
      actions: filtered.filter((action) => (action.group ?? "navigate") === group),
    })).filter((entry) => entry.actions.length > 0);
  }, [filtered]);

  async function run(action: PaletteAction) {
    setOpen(false);
    setQuery("");
    await action.run();
  }

  function moveActive(delta: number) {
    if (filtered.length === 0) return;
    setActiveIndex((current) => {
      const start = current < 0 ? 0 : current;
      return (start + delta + filtered.length) % filtered.length;
    });
  }

  function handleDialogKeyDown(event: React.KeyboardEvent<HTMLDivElement>) {
    if (event.key === "Tab") {
      event.preventDefault();
      const next = document.activeElement === inputRef.current ? closeRef.current : inputRef.current;
      next?.focus();
      return;
    }
    if (event.key === "ArrowDown") {
      event.preventDefault();
      moveActive(1);
      return;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      moveActive(-1);
      return;
    }
    if (event.key === "Enter" && resolvedIndex >= 0 && filtered[resolvedIndex]) {
      event.preventDefault();
      void run(filtered[resolvedIndex]);
    }
  }

  const activeId =
    resolvedIndex >= 0 && filtered[resolvedIndex]
      ? `palette-option-${filtered[resolvedIndex].id}`
      : undefined;

  return (
    <>
      <button
        type="button"
        onClick={() => {
          setOpen(true);
          setActiveIndex(0);
        }}
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
          role="presentation"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) {
              setOpen(false);
              setQuery("");
            }
          }}
        >
          <div
            className="w-full max-w-xl overflow-hidden rounded-lg border border-border bg-popover shadow-2xl"
            role="dialog"
            aria-modal="true"
            aria-label="Command palette"
            onKeyDown={handleDialogKeyDown}
          >
            <div className="flex items-center border-b border-border px-3">
              <Search className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
              <input
                ref={inputRef}
                value={query}
                onChange={(event) => {
                  setQuery(event.target.value);
                  setActiveIndex(0);
                }}
                placeholder="Find an ATB operation or investigation view…"
                className="h-12 flex-1 bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground"
                role="combobox"
                aria-autocomplete="list"
                aria-expanded="true"
                aria-controls="command-palette-listbox"
                aria-activedescendant={activeId}
                aria-label="Filter commands"
              />
              <button
                ref={closeRef}
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
            <ul
              id="command-palette-listbox"
              role="listbox"
              aria-label="Commands"
              className="max-h-80 overflow-y-auto p-2"
            >
              {grouped.map((entry) => (
                <li key={entry.group} role="presentation">
                  <p className="px-3 py-1 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                    {GROUP_LABEL[entry.group]}
                  </p>
                  <ul role="presentation">
                    {entry.actions.map((action) => {
                      const selected = filtered[resolvedIndex]?.id === action.id;
                      return (
                        <li key={action.id} role="presentation">
                          <button
                            type="button"
                            id={`palette-option-${action.id}`}
                            role="option"
                            aria-selected={selected}
                            tabIndex={-1}
                            onMouseEnter={() =>
                              setActiveIndex(filtered.findIndex((candidate) => candidate.id === action.id))
                            }
                            onClick={() => void run(action)}
                            className={`flex w-full items-center justify-between rounded-md px-3 py-2.5 text-left text-sm ${selected ? "bg-muted" : "hover:bg-muted"}`}
                          >
                            <span>{action.label}</span>
                            {action.hint && (
                              <span className="font-mono text-xs text-muted-foreground">{action.hint}</span>
                            )}
                          </button>
                        </li>
                      );
                    })}
                  </ul>
                </li>
              ))}
              {filtered.length === 0 && (
                <li className="px-3 py-8 text-center text-sm text-muted-foreground" role="presentation">
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
