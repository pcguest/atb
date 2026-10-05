"use client";

import { useId } from "react";
import { Boxes, Clock3, FileJson2, GitBranch, Info, ListChecks, ShieldCheck, type LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";
import { SURFACE_IDS, type SurfaceId } from "@/lib/viewer-location";

/**
 * Canonical investigation surfaces. Names and order are load-bearing product
 * concepts — do not rename, remove, merge, or reorder them. The id list is
 * derived from `SURFACE_IDS` so the taxonomy has a single source of truth.
 */
export type Surface = SurfaceId;

const SURFACE_META: Record<SurfaceId, { label: string; Icon: LucideIcon }> = {
  incident: { label: "Run", Icon: Info },
  findings: { label: "Findings", Icon: ListChecks },
  timeline: { label: "Timeline", Icon: Clock3 },
  context: { label: "Context", Icon: Boxes },
  relationships: { label: "Relationships", Icon: GitBranch },
  evidence: { label: "Evidence", Icon: FileJson2 },
  trust: { label: "Evidence status", Icon: ShieldCheck },
};

export const investigationSurfaces: ReadonlyArray<readonly [SurfaceId, string, LucideIcon]> = SURFACE_IDS.map(
  (id) => [id, SURFACE_META[id].label, SURFACE_META[id].Icon] as const,
);

/**
 * Navigational grouping only. Groups are orientation labels, not routes, and
 * every canonical surface remains a single direct action away.
 */
export const investigationGroups = [
  { label: "Overview", ids: ["incident", "findings"] },
  { label: "Investigate", ids: ["timeline", "context", "relationships"] },
  { label: "Inspect", ids: ["evidence", "trust"] },
] as const satisfies ReadonlyArray<{ label: string; ids: readonly SurfaceId[] }>;

export function surfaceLabel(id: Surface): string {
  return SURFACE_META[id].label;
}

type NavProps = {
  active: Surface;
  onSelect: (id: Surface) => void;
  findingCount?: number;
};

function NavButton({ id, label, Icon, active, onSelect, findingCount, compact }: {
  id: Surface;
  label: string;
  Icon: LucideIcon;
  active: Surface;
  onSelect: (id: Surface) => void;
  findingCount?: number;
  compact?: boolean;
}) {
  const showsCount = id === "findings" && findingCount !== undefined && findingCount > 0;
  const countId = useId();
  return (
    <button
      type="button"
      onClick={() => onSelect(id)}
      aria-label={label}
      aria-describedby={showsCount ? countId : undefined}
      aria-current={active === id ? "page" : undefined}
      className={cn(
        "flex items-center gap-3 rounded-md border px-3 py-2.5 text-left text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        compact ? "shrink-0 rounded px-3 py-2" : "w-full",
        active === id
          ? "border-primary/30 bg-primary/10 text-foreground"
          : "border-transparent text-muted-foreground hover:bg-muted hover:text-foreground",
      )}
    >
      <Icon className="h-4 w-4 shrink-0" aria-hidden="true" />
      {label}
      {showsCount && (
        <>
          <span className="ml-auto text-xs" aria-hidden="true">
            {findingCount}
          </span>
          <span id={countId} className="sr-only">
            {findingCount} {findingCount === 1 ? "finding" : "findings"}
          </span>
        </>
      )}
    </button>
  );
}

/** Desktop grouped navigation. */
export function InvestigationNavigation({ active, onSelect, findingCount }: NavProps) {
  return (
    <nav className="flex-1 space-y-4 p-2" aria-label="Investigation">
      {investigationGroups.map((group) => (
        <div key={group.label} className="space-y-1">
          <p className="px-3 pt-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
            {group.label}
          </p>
          {group.ids.map((id) => {
            const surface = investigationSurfaces.find(([surfaceId]) => surfaceId === id);
            if (!surface) return null;
            return (
              <NavButton
                key={id}
                id={surface[0]}
                label={surface[1]}
                Icon={surface[2]}
                active={active}
                onSelect={onSelect}
                findingCount={findingCount}
              />
            );
          })}
        </div>
      ))}
    </nav>
  );
}

/** Compact (narrow viewport) navigation retaining all seven destinations. */
export function CompactInvestigationNavigation({ active, onSelect, findingCount }: NavProps) {
  return (
    <nav
      aria-label="Investigation"
      className="flex shrink-0 gap-1 overflow-x-auto border-b border-border p-2 md:hidden"
    >
      {investigationSurfaces.map(([id, label, Icon]) => (
        <NavButton
          key={id}
          id={id}
          label={label}
          Icon={Icon}
          active={active}
          onSelect={onSelect}
          findingCount={findingCount}
          compact
        />
      ))}
    </nav>
  );
}
