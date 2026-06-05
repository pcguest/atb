"use client";

import React from "react";
import { FixedSizeList } from "react-window";
import { useRouter } from "next/navigation";

import { useSessionsQuery } from "@/lib/api-client";
import type { SessionEntry } from "@/lib/types";
import AnomalyBadge from "./AnomalyBadge";

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
      router.push(
        `/view?bundlePath=${encodeURIComponent(session.bundle_path)}${currentFragment}`,
      );
    };

    return (
      <div
        style={style}
        className="flex items-center border-b border-gray-700 hover:bg-gray-800 cursor-pointer"
        onClick={handleRowClick}
      >
        <div className="w-1/5 p-2 truncate">{session.actor.display_name}</div>
        <div className="w-1/5 p-2 truncate">{startedAt}</div>
        <div className="w-1/12 p-2 text-center">{session.exchange_count}</div>
        <div className="w-1/5 p-2 truncate">
          {session.inferred_profile.replace("atb.profile.", "")}
        </div>
        <div className="w-1/12 p-2 text-center">{session.cas_grade}</div>
        <div className="w-1/5 p-2 flex items-center">
          {session.anomaly_flags.map((flag) => (
            <AnomalyBadge key={flag} flag={flag} />
          ))}
        </div>
      </div>
    );
  };

  return (
    <div className="bg-gray-900 text-gray-100 p-4 rounded-lg shadow-lg">
      <h2 className="text-xl font-semibold mb-4">All sessions</h2>
      <div className="flex font-bold border-b-2 border-gray-600 pb-2 mb-2">
        <div className="w-1/5 p-2">Actor</div>
        <div className="w-1/5 p-2">Started at</div>
        <div className="w-1/12 p-2 text-center">Exchanges</div>
        <div className="w-1/5 p-2">Profile</div>
        <div className="w-1/12 p-2 text-center">CAS grade</div>
        <div className="w-1/5 p-2">Anomalies</div>
      </div>
      <FixedSizeList height={500} itemCount={sessions.length} itemSize={40} width="100%">
        {Row}
      </FixedSizeList>
    </div>
  );
};

export default SessionList;
