"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { detectAgentMode } from "@/lib/agent-mode";
import { useWorkspaceBundlesQuery } from "@/lib/api-client";
import {
  bundleDisplayId,
  bundleViewHref,
  formatWorkspaceTimestamp,
} from "@/lib/workspace-nav";

import { WorkspaceWelcome } from "@/app/workspace/WorkspaceWelcome";

export default function WorkspacePage() {
  const [agentMode, setAgentMode] = useState<boolean | null>(null);

  useEffect(() => {
    let mounted = true;
    void detectAgentMode().then((enabled) => {
      if (mounted) {
        setAgentMode(enabled);
      }
    });
    return () => {
      mounted = false;
    };
  }, []);

  const bundlesQuery = useWorkspaceBundlesQuery(agentMode === true);

  return (
    <div className="mx-auto flex min-h-screen max-w-5xl flex-col px-4 py-8">
      <header className="mb-6 border-b border-border pb-4">
        <p className="font-mono text-xs uppercase tracking-widest text-muted-foreground">ATB</p>
        <h1 className="mt-1 text-2xl font-semibold text-foreground">Workspace bundles</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Read-only index of closed session bundles. Evidence remains immutable on disk; this page
          only lists metadata for navigation.
        </p>
      </header>

      {agentMode === null && (
        <p className="text-sm text-muted-foreground" role="status">
          Detecting workspace backend…
        </p>
      )}

      {agentMode === false && (
        <div
          className="rounded-md border border-border bg-muted/30 p-4 text-sm text-muted-foreground"
          role="status"
        >
          Workspace bundle listing is available when the viewer is served by the ATB Agent. Single-bundle{" "}
          <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">atb view</code> mode opens
          one bundle directly and does not expose this index.
        </div>
      )}

      {agentMode === true && bundlesQuery.isLoading && (
        <p className="text-sm text-muted-foreground" role="status">
          Loading bundles…
        </p>
      )}

      {agentMode === true && bundlesQuery.isError && (
        <div className="rounded-md border border-red-900/50 bg-red-950/40 p-4 text-sm text-red-200" role="alert">
          Failed to load workspace bundles: {bundlesQuery.error.message}
        </div>
      )}

      {agentMode === true && bundlesQuery.isSuccess && bundlesQuery.data.bundles.length === 0 && (
        <WorkspaceWelcome />
      )}

      {agentMode === true && bundlesQuery.isSuccess && bundlesQuery.data.bundles.length > 0 && (
        <div className="overflow-x-auto rounded-md border border-border">
          <table className="w-full min-w-[640px] border-collapse text-left text-sm">
            <thead className="border-b border-border bg-muted/40">
              <tr>
                <th scope="col" className="px-3 py-2 font-medium text-foreground">
                  ID
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-foreground">
                  Profile
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-foreground">
                  Events
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-foreground">
                  Opened
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-foreground">
                  Closed
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-foreground">
                  View
                </th>
              </tr>
            </thead>
            <tbody>
              {bundlesQuery.data.bundles.map((bundle) => (
                <tr key={bundle.id} className="border-b border-border/70 last:border-b-0">
                  <td className="px-3 py-2 font-mono text-xs">{bundleDisplayId(bundle)}</td>
                  <td className="px-3 py-2 font-mono text-xs text-muted-foreground">
                    {bundle.profile_id ?? "—"}
                  </td>
                  <td className="px-3 py-2 tabular-nums">{bundle.event_count}</td>
                  <td className="px-3 py-2 whitespace-nowrap text-muted-foreground">
                    {formatWorkspaceTimestamp(bundle.opened_at)}
                  </td>
                  <td className="px-3 py-2 whitespace-nowrap text-muted-foreground">
                    {formatWorkspaceTimestamp(bundle.closed_at)}
                  </td>
                  <td className="px-3 py-2">
                    <Link
                      href={bundleViewHref(bundle, true)}
                      className="font-mono text-xs text-primary underline-offset-2 hover:underline"
                    >
                      Open
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
