"use client";

import type { ReactNode } from "react";

import { CopyAction } from "./ui/investigation";
import { HashValue } from "@/components/dashboard/HashValue";
import { displayBundlePath } from "@/lib/display-path";

/**
 * Evidence-object identity for the investigation shell.
 *
 * It communicates the evidence object, its human-readable identity, the
 * current investigation surface and bounded summary facts. It deliberately
 * does NOT invent a Run/Session identity that the backend has not established.
 *
 * Evidential identity is the bundle head hash (content-addressed). The bundle
 * path is shown as a locator only: a path can be reused, moved, or renamed and
 * is not the stable identity of the evidence. When the head hash is available
 * it is shown with the path; both stay available via copy.
 */
export function ObjectIdentityHeader({
  bundlePath,
  headHash,
  eventCount,
  surfaceLabel,
  controls,
  status,
}: {
  bundlePath: string;
  headHash?: string;
  eventCount: number;
  surfaceLabel: string;
  controls?: ReactNode;
  status?: ReactNode;
}) {
  return (
    <header className="shrink-0 border-b border-border bg-surface-1 px-4 py-3 xl:px-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
            Evidence bundle
          </p>
          <div className="mt-1 flex items-center gap-2">
            <h1 className="min-w-0 truncate text-sm font-semibold" title={bundlePath}>
              {displayBundlePath(bundlePath)}
            </h1>
            <CopyAction value={bundlePath} label="Copy bundle path" />
          </div>
          <p className="mt-1 break-words text-xs text-muted-foreground">
            {eventCount} {eventCount === 1 ? "record" : "records"}
            <span aria-hidden="true"> · </span>
            {surfaceLabel}
          </p>
          {headHash ? (
            <p className="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
              <span>Bundle head</span>
              <HashValue hash={headHash} className="text-foreground" />
              <span aria-hidden="true">·</span>
              <span>path is a locator, not identity</span>
            </p>
          ) : null}
        </div>
        {controls ? <div className="flex items-center gap-2">{controls}</div> : null}
      </div>
      {status ? <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 text-xs">{status}</div> : null}
    </header>
  );
}
