"use client";

import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
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

interface ActorsResponse {
  actors: {
    [actorDisplayName: string]: SessionEntry[];
  };
}

const fetchActorSessions = async (): Promise<ActorsResponse['actors']> => {
  const response = await fetch('/api/v1/sessions/by-actor');
  if (!response.ok) {
    throw new Error('Failed to fetch actor sessions');
  }
  const data: ActorsResponse = await response.json();
  return data.actors;
};

const ActorSessions: React.FC = () => {
  const router = useRouter();
  const { data: actors, isLoading, error } = useQuery<ActorsResponse['actors']>({
    queryKey: ['actorSessions'],
    queryFn: fetchActorSessions,
  });
  const [expandedActors, setExpandedActors] = useState<Set<string>>(new Set());

  if (isLoading) return <div>Loading actor sessions...</div>;
  if (error) return <div>Error: {error.message}</div>;
  if (!actors || Object.keys(actors).length === 0) return <div>No actor sessions found.</div>;

  const toggleExpand = (actorDisplayName: string) => {
    setExpandedActors((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(actorDisplayName)) {
        newSet.delete(actorDisplayName);
      } else {
        newSet.add(actorDisplayName);
      }
      return newSet;
    });
  };

  const handleSessionRowClick = (session: SessionEntry) => {
    const currentFragment = window.location.hash;
    router.push(`/view?bundlePath=${encodeURIComponent(session.bundle_path)}${currentFragment}`);
  };

  return (
    <div className="bg-gray-900 text-gray-100 p-4 rounded-lg shadow-lg">
      <h2 className="text-xl font-semibold mb-4">Sessions by Actor</h2>
      {Object.entries(actors).map(([actorDisplayName, sessions]) => {
        const isExpanded = expandedActors.has(actorDisplayName);
        const hasUnresolvedIdentity = sessions.some((s) =>
          s.anomaly_flags.includes('unresolved_identity')
        );

        return (
          <div key={actorDisplayName} className="mb-4 border border-gray-700 rounded-md">
            <div
              className="flex items-center justify-between p-3 bg-gray-800 cursor-pointer hover:bg-gray-700"
              role="button"
              tabIndex={0}
              aria-expanded={isExpanded}
              onClick={() => toggleExpand(actorDisplayName)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  toggleExpand(actorDisplayName);
                }
              }}
            >
              <div className="flex items-center">
                <span className="font-bold text-lg">{actorDisplayName}</span>
                {hasUnresolvedIdentity && (
                  <AnomalyBadge flag="unresolved_identity" />
                )}
              </div>
              <span>{isExpanded ? 'Collapse ▲' : 'Expand ▼'}</span>
            </div>
            {isExpanded && (
              <div className="p-3">
                <div className="flex font-bold border-b border-gray-600 pb-2 mb-2 text-sm">
                  <div className="w-1/5 p-1">Session ID</div>
                  <div className="w-1/5 p-1">Started At</div>
                  <div className="w-1/12 p-1 text-center">Exchanges</div>
                  <div className="w-1/5 p-1">Profile</div>
                  <div className="w-1/12 p-1 text-center">CAS</div>
                  <div className="w-1/5 p-1">Anomalies</div>
                </div>
                {sessions.map((session) => {
                  const startedAt = new Date(session.started_at).toLocaleString();
                  return (
                    <div
                      key={session.session_id}
                      className="flex items-center border-b border-gray-700 hover:bg-gray-800 cursor-pointer text-sm"
                      role="button"
                      tabIndex={0}
                      onClick={() => handleSessionRowClick(session)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault();
                          handleSessionRowClick(session);
                        }
                      }}
                    >
                      <div className="w-1/5 p-1 truncate">{session.session_id}</div>
                      <div className="w-1/5 p-1 truncate">{startedAt}</div>
                      <div className="w-1/12 p-1 text-center">{session.exchange_count}</div>
                      <div className="w-1/5 p-1 truncate">
                        {session.inferred_profile.replace('atb.profile.', '')}
                      </div>
                      <div className="w-1/12 p-1 text-center">{session.cas_grade}</div>
                      <div className="w-1/5 p-1 flex items-center">
                        {session.anomaly_flags.map((flag) => (
                          <AnomalyBadge key={flag} flag={flag} />
                        ))}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};

export default ActorSessions;
