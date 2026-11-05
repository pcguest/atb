"use client";

import React from "react";

import ActorSessions from "@/components/ActorSessions";
import { LocalViewerShell } from "@/components/local-viewer/LocalViewerShell";
import SchemaStatus from "@/components/SchemaStatus";
import SessionList from "@/components/SessionList";

const SessionsPage: React.FC = () => {
  return (
    <LocalViewerShell
      route="/sessions"
      title="Session overview"
      description={<>Review actor sessions, schema coverage, and closed trace bundles from a local ATB viewer. The workspace index is populated when the viewer is started with <code className="font-mono text-primary">atb view --sessions &lt;dir-or-glob&gt;</code>. A single-bundle viewer opens one bundle directly and reports only that bundle&apos;s sessions here.</>}
      contentClassName="mx-auto w-full max-w-5xl px-4 py-6"
    >
      <section className="space-y-6" aria-label="Session and schema overview">
        <SchemaStatus />
        <SessionList />
        <ActorSessions />
      </section>
    </LocalViewerShell>
  );
};

export default SessionsPage;
