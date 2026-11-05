"use client";

import { useEffect, useMemo, useState, type ReactNode } from "react";

import { ThemeToggle } from "@/components/ThemeToggle";
import { CompactInvestigationNavigation, InvestigationNavigation, surfaceLabel } from "@/app/view/components/InvestigationNavigation";
import { displayBundlePath } from "@/lib/display-path";
import {
  localViewerHref,
  parseLocalViewerContext,
  type LocalViewerContext,
  type LocalViewerRoute,
} from "@/lib/local-viewer-context";
import { cn } from "@/lib/utils";
import type { SurfaceId } from "@/lib/viewer-location";

type LocalViewerShellProps = {
  route: LocalViewerRoute;
  children: ReactNode;
  title?: string;
  description?: string;
  bundlePath?: string;
  sessionId?: string;
  surface?: SurfaceId | null;
  findingCount?: number;
  controls?: ReactNode;
  mainId?: string;
  contentClassName?: string;
  onSelectSurface?: (surface: SurfaceId) => void;
};

function fallbackContext(route: LocalViewerRoute, bundlePath?: string, sessionId?: string, surface?: SurfaceId | null): LocalViewerContext {
  return {
    route,
    bundlePath: bundlePath ?? null,
    sessionId: sessionId ?? null,
    surface: surface ?? null,
    seq: null,
    from: null,
    focus: null,
    fragment: "",
    discarded: [],
  };
}

function LibraryLink({ href, active, children }: { href: string; active: boolean; children: ReactNode }) {
  return (
    <a
      href={href}
      aria-current={active ? "page" : undefined}
      className={cn(
        "flex items-center gap-3 rounded-md border px-3 py-2.5 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        active
          ? "border-primary/30 bg-primary/10 text-foreground"
          : "border-transparent text-muted-foreground hover:bg-muted hover:text-foreground",
      )}
    >
      {children}
    </a>
  );
}

export function LocalViewerShell({
  route,
  children,
  title,
  description,
  bundlePath,
  sessionId,
  surface,
  findingCount,
  controls,
  mainId = "local-viewer-content",
  contentClassName,
  onSelectSurface,
}: LocalViewerShellProps) {
  const [locationContext, setLocationContext] = useState<LocalViewerContext>(() =>
    fallbackContext(route, bundlePath, sessionId, surface),
  );

  useEffect(() => {
    const sync = () => setLocationContext(parseLocalViewerContext(window.location.href));
    sync();
    window.addEventListener("popstate", sync);
    window.addEventListener("hashchange", sync);
    return () => {
      window.removeEventListener("popstate", sync);
      window.removeEventListener("hashchange", sync);
    };
  }, []);

  const context = useMemo(
    () => ({
      ...locationContext,
      route,
      bundlePath: bundlePath ?? locationContext.bundlePath,
      sessionId: sessionId ?? locationContext.sessionId,
      surface: surface ?? locationContext.surface,
    }),
    [locationContext, route, bundlePath, sessionId, surface],
  );

  const clearViewState = { surface: null, seq: null, from: null, focus: null } as const;
  const bundlesHref = localViewerHref("/workspace", context, clearViewState);
  const sessionsHref = localViewerHref("/sessions", context, clearViewState);
  const activeSurface = surface ?? context.surface;
  const bundleLabel = bundlePath ? displayBundlePath(bundlePath) : "No bundle selected";

  return (
    <div className={cn("flex w-full bg-background", route === "/view" ? "h-full min-h-0 overflow-hidden" : "min-h-screen")}>
      <button
        type="button"
        onClick={() => document.getElementById(mainId)?.focus()}
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-md focus:border focus:border-border focus:bg-card focus:px-3 focus:py-2 focus:text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        Skip to content
      </button>
      <aside className="hidden w-44 shrink-0 flex-col border-r border-border bg-surface-1 md:flex xl:w-56" aria-label="Local viewer navigation">
        <div className="border-b border-border px-4 py-5">
          <a href={bundlesHref} className="font-semibold tracking-tight text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
            ATB
          </a>
          <p className="mt-1 text-xs text-muted-foreground">Local evidence viewer</p>
        </div>
        <nav aria-label="Library" className="space-y-1 p-2">
          <p className="px-3 pt-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Library</p>
          <LibraryLink href={bundlesHref} active={route === "/workspace"}>Bundles</LibraryLink>
          <LibraryLink href={sessionsHref} active={route === "/sessions"}>Sessions</LibraryLink>
        </nav>
        {route === "/view" && onSelectSurface ? (
          <InvestigationNavigation active={activeSurface ?? "incident"} onSelect={onSelectSurface} findingCount={findingCount} />
        ) : null}
        <div className="mt-auto border-t border-border p-4 text-xs leading-5 text-muted-foreground">
          <p className="font-medium text-foreground">Local viewer</p>
          <p className="mt-1 break-words font-mono text-[11px]" title={bundlePath ?? undefined}>{bundleLabel}</p>
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="shrink-0 border-b border-border bg-surface-1 px-4 py-2.5 xl:px-6">
          <div className="flex flex-wrap items-center gap-3">
            <nav aria-label="Breadcrumb" className="min-w-0 flex-1">
              <ol className="flex flex-wrap items-center gap-1 text-xs text-muted-foreground">
                <li><a href={bundlesHref} className="hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">Library</a></li>
                <li aria-hidden="true">/</li>
                {route === "/workspace" ? (
                  <li aria-current="page" className="font-medium text-foreground">Bundles</li>
                ) : route === "/sessions" ? (
                  <li aria-current="page" className="font-medium text-foreground">Sessions</li>
                ) : (
                  <>
                    <li><a href={bundlesHref} className="hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">Bundles</a></li>
                    {bundlePath ? (
                      <>
                        <li aria-hidden="true">/</li>
                        <li className="max-w-[16rem] truncate font-mono text-[11px]" title={bundlePath}>{bundleLabel}</li>
                      </>
                    ) : null}
                    <li aria-hidden="true">/</li>
                    <li aria-current="page" className="font-medium text-foreground">{surfaceLabel(activeSurface ?? "incident")}</li>
                  </>
                )}
              </ol>
            </nav>
            <div className="flex items-center gap-2">
              {controls}
              <ThemeToggle />
            </div>
          </div>
        </header>

        <div className="md:hidden">
          <nav aria-label="Library" className="flex flex-wrap gap-1 border-b border-border p-2">
            <LibraryLink href={bundlesHref} active={route === "/workspace"}>Bundles</LibraryLink>
            <LibraryLink href={sessionsHref} active={route === "/sessions"}>Sessions</LibraryLink>
          </nav>
          {route === "/view" && onSelectSurface ? (
            <CompactInvestigationNavigation active={activeSurface ?? "incident"} onSelect={onSelectSurface} findingCount={findingCount} />
          ) : null}
        </div>

        <main
          id={mainId}
          tabIndex={-1}
          className={cn("flex min-w-0 flex-1 flex-col focus:outline-none", route === "/view" ? "min-h-0 overflow-y-auto" : "overflow-y-auto", contentClassName)}
        >
          {title || description ? (
            <div className="border-b border-border px-4 py-5 xl:px-6">
              {title ? <h1 className="text-xl font-semibold text-foreground">{title}</h1> : null}
              {description ? <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">{description}</p> : null}
            </div>
          ) : null}
          {children}
        </main>
      </div>
    </div>
  );
}
