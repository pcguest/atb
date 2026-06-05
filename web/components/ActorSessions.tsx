"use client";

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';

import { useActorSessionsQuery } from '@/lib/api-client';
import type { SessionEntry } from '@/lib/types';
import AnomalyBadge from './AnomalyBadge';

// ActorSessions renders the actor-grouped session index from
// GET /api/v1/sessions/by-actor. It reads through the shared api-client so the
// viewer session token (delivered in the URL fragment) is attached to the
// request; a raw fetch() would be rejected by the authenticated viewer.
const ActorSessions: React.FC = () => {
  const router = useRouter();
  const { data: actors, isLoading, error } = useActorSessionsQuery(true);
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
