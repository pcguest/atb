"use client";

import React from "react";
import { FixedSizeList } from "react-window";
import { useRouter } from "next/navigation";

import { useSessionsQuery } from "@/lib/api-client";
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

  if (isLoading) return <div className="text-gray-300">Loading sessions…</div>;
  if (error) return <div className="text-red-400">Error: {error.message}</div>;
  if (!sessions || sessions.length === 0) {
    return <div className="text-gray-400">No sessions found.</div>;
  }

  const Row = ({ index, style }: { index: number; style: React.CSSProperties }) => {
    const session: SessionEntry = sessions[index];
    const startedAt = new Date(session.started_at).toLocaleString();

    const handleRowClick = () => {
      // Navigate to the single-bundle viewer for this session, preserving the
      // session-token fragment so the destination view stays authenticated.
      const currentFragment = window.location.hash;
      router.push(`/view?bundlePath=${encodeURIComponent(session.bundle_path)}${currentFragment}`);
    };

    return (
      <div
        style={style}
        className="group flex items-center border-b border-gray-700 hover:bg-gray-800"
        role="row"
        aria-rowindex={index + 2}
      >
        <div className="w-1/5 truncate p-2" role="cell">
          <button
            type="button"
            className="w-full truncate rounded text-left hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring"
            aria-label={`Open session ${session.session_id} for ${session.actor.display_name}, started ${startedAt}`}
            onClick={handleRowClick}
          >
            {session.actor.display_name}
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
    <div className="rounded-lg bg-gray-900 p-4 text-gray-100 shadow-lg">
      <h2 className="mb-4 text-xl font-semibold">All sessions</h2>
      <div role="table" aria-label="Sessions" aria-colcount={6} aria-rowcount={sessions.length + 1}>
        <div role="rowgroup">
          <div className="mb-2 flex border-b-2 border-gray-600 pb-2 font-bold" role="row">
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
