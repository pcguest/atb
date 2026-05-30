"use client";

import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { FixedSizeList } from 'react-window';
import { useRouter } from 'next/navigation';
import AnomalyBadge from './AnomalyBadge'; // Assuming AnomalyBadge is in the same directory

// Actor summary as serialized by internal/sessionindex (nested object).
interface ActorSummary {
  display_name: string;
  email: string;
}

// Define the SessionEntry type based on the Go struct
interface SessionEntry {
  session_id: string;
  actor: ActorSummary;
  started_at: string; // ISO 8601 string
  closed_at: string | null; // ISO 8601 string or null
  exchange_count: number;
  inferred_profile: string;
  cas_grade: string;
  anomaly_flags: string[];
  bundle_path: string;
}

interface SessionsResponse {
  sessions: SessionEntry[];
}

const fetchSessions = async (): Promise<SessionEntry[]> => {
  const response = await fetch('/api/v1/sessions');
  if (!response.ok) {
    throw new Error('Failed to fetch sessions');
  }
  const data: SessionsResponse = await response.json();
  return data.sessions;
};

const SessionList: React.FC = () => {
  const router = useRouter();
  const { data: sessions, isLoading, error } = useQuery<SessionEntry[]>({
    queryKey: ['sessions'],
    queryFn: fetchSessions,
  });

  if (isLoading) return <div>Loading sessions...</div>;
  if (error) return <div>Error: {error.message}</div>;
  if (!sessions || sessions.length === 0) return <div>No sessions found.</div>;

  const Row = ({ index, style }: { index: number; style: React.CSSProperties }) => {
    const session = sessions[index];
    const startedAt = new Date(session.started_at).toLocaleString();
    const closedAt = session.closed_at ? new Date(session.closed_at).toLocaleString() : 'N/A';

    const handleRowClick = () => {
      // Navigate to the existing bundle viewer
      // The session token is expected to be in the URL fragment, so we need to preserve it.
      // Assuming the current URL already has the session token in the fragment.
      const currentFragment = window.location.hash;
      router.push(`/view?bundlePath=${encodeURIComponent(session.bundle_path)}${currentFragment}`);
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
          {session.inferred_profile.replace('atb.profile.', '')}
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
      <h2 className="text-xl font-semibold mb-4">All Sessions</h2>
      <div className="flex font-bold border-b-2 border-gray-600 pb-2 mb-2">
        <div className="w-1/5 p-2">Actor</div>
        <div className="w-1/5 p-2">Started At</div>
        <div className="w-1/12 p-2 text-center">Exchanges</div>
        <div className="w-1/5 p-2">Profile</div>
        <div className="w-1/12 p-2 text-center">CAS Grade</div>
        <div className="w-1/5 p-2">Anomalies</div>
      </div>
      <FixedSizeList
        height={500} // Fixed height for the virtualized list
        itemCount={sessions.length}
        itemSize={40} // Height of each row
        width="100%"
      >
        {Row}
      </FixedSizeList>
    </div>
  );
};

export default SessionList;
