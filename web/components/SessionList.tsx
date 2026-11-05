"use client";

import React from "react";
import { FixedSizeList } from "react-window";
import { useRouter } from "next/navigation";

import { useSessionsQuery } from "@/lib/api-client";
import { localViewerHref, parseLocalViewerContext } from "@/lib/local-viewer-context";
import type { SessionEntry } from "@/lib/types";
import AnomalyBadge from "./AnomalyBadge";

const SessionRows = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  function SessionRows(props, ref) {
    return <div {...props} ref={ref} role="rowgroup" />;
  },
);

// SessionList renders the workspace session index from GET /api/v1/sessions.
// It reads through the shared api-client so the viewer session token (delivered
// in the URL fragment) is attached to the request; a raw fetch() would be
// rejected by the authenticated viewer.
const SessionList: React.FC = () => {
  const router = useRouter();
  const { data: sessions, isLoading, error } = useSessionsQuery(true);

  if (isLoading) return <div className="text-muted-foreground">Loading sessions…</div>;
  if (error) return <div className="text-danger">Error: {error.message}</div>;
  if (!sessions || sessions.length === 0) {
    return <div className="text-muted-foreground">No sessions found.</div>;
  }

  const Row = ({ index, style }: { index: number; style: React.CSSProperties }) => {
    const session: SessionEntry = sessions[index];
    const startedAt = new Date(session.started_at).toLocaleString();
    const actorLabel = session.actor.display_name.trim() || session.session_id;

    const handleRowClick = () => {
      // Preserve the current local-viewer context and session-token fragment
      // while selecting this session's bundle.
      const context = parseLocalViewerContext(window.location.href);
      router.push(localViewerHref("/view", context, {
        bundlePath: session.bundle_path,
        sessionId: session.session_id,
        surface: "incident",
        seq: null,
        from: null,
        focus: null,
      }));
    };

    return (
      <div
        style={style}
        className="group flex items-center border-b border-border hover:bg-surface-2"
        role="row"
        aria-rowindex={index + 2}
      >
        <div className="w-1/5 truncate p-2" role="cell">
          <button
            type="button"
            className="w-full truncate rounded text-left hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring"
            aria-label={`Open session ${session.session_id} for ${actorLabel}, started ${startedAt}`}
            onClick={handleRowClick}
          >
            {actorLabel}
          </button>
        </div>
        <div className="w-1/5 truncate p-2" role="cell">
          {startedAt}
        </div>
        <div className="w-1/12 p-2 text-center" role="cell">
          {session.exchange_count}
        </div>
        <div className="w-1/5 truncate p-2" role="cell">
          {session.inferred_profile.replace("atb.profile.", "")}
        </div>
        <div className="w-1/12 p-2 text-center" role="cell">
          {session.cas_grade}
        </div>
        <div className="flex w-1/5 items-center p-2" role="cell">
          {session.anomaly_flags.map((flag) => (
            <AnomalyBadge key={flag} flag={flag} />
          ))}
        </div>
      </div>
    );
  };

  return (
    <div className="rounded-lg bg-surface-1 p-4 text-foreground shadow-lg">
      <h2 className="mb-4 text-xl font-semibold">All sessions</h2>
      <div role="table" aria-label="Sessions" aria-colcount={6} aria-rowcount={sessions.length + 1}>
        <div role="rowgroup">
          <div className="mb-2 flex border-b-2 border-border pb-2 font-bold" role="row">
            <div className="w-1/5 p-2" role="columnheader">
              Actor
            </div>
            <div className="w-1/5 p-2" role="columnheader">
              Started at
            </div>
            <div className="w-1/12 p-2 text-center" role="columnheader">
              Exchanges
            </div>
            <div className="w-1/5 p-2" role="columnheader">
              Profile
            </div>
            <div className="w-1/12 p-2 text-center" role="columnheader">
              CAS grade
            </div>
            <div className="w-1/5 p-2" role="columnheader">
              Anomalies
            </div>
          </div>
        </div>
        <FixedSizeList
          height={500}
          itemCount={sessions.length}
          itemSize={40}
          width="100%"
          outerElementType={SessionRows}
        >
          {Row}
        </FixedSizeList>
      </div>
    </div>
  );
};

export default SessionList;
