"use client";

import React from "react";

import SchemaStatus from "@/components/SchemaStatus";
import SessionList from "@/components/SessionList";

const SessionsPage: React.FC = () => {
  return (
    <main className="mx-auto min-h-screen max-w-5xl px-4 py-8">
      <header className="mb-6 border-b border-border pb-4">
        <p className="font-mono text-xs uppercase tracking-widest text-muted-foreground">ATB</p>
        <h1 className="mt-1 text-2xl font-semibold text-foreground">Session overview</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Review actor sessions, schema coverage, and closed trace bundles from a local ATB viewer.
        </p>
        <p className="mt-2 max-w-3xl text-xs text-muted-foreground">
          The workspace index is populated when the viewer is started with{" "}
          <code className="font-mono text-primary">atb view --sessions &lt;dir-or-glob&gt;</code>.
          A single-bundle viewer opens one bundle directly and reports no sessions here.
        </p>
      </header>

      <section className="space-y-6" aria-label="Session and schema overview">
        <SchemaStatus />
        <SessionList />
      </section>
    </main>
  );
};

export default SessionsPage;
