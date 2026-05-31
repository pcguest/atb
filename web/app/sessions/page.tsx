import React from "react";

const SessionsPage: React.FC = () => {
  return (
    <main className="mx-auto min-h-screen max-w-5xl px-4 py-8">
      <header className="mb-6 border-b border-border pb-4">
        <p className="font-mono text-xs uppercase tracking-widest text-muted-foreground">
          ATB
        </p>
        <h1 className="mt-1 text-2xl font-semibold text-foreground">
          Session overview
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Review actor sessions, schema coverage, and closed trace bundles from a local ATB viewer.
        </p>
      </header>

      <section
        className="rounded-md border border-border bg-muted/30 p-5 text-sm text-muted-foreground"
        role="status"
      >
        <h2 className="text-base font-medium text-foreground">
          Local session API not connected
        </h2>

        <p className="mt-2 max-w-3xl">
          This page reads session data from the local ATB viewer API. Start a verified viewer
          session from a bundle, then reopen the viewer URL printed by the CLI.
        </p>

        <div className="mt-4 rounded-md border border-border bg-background/60 p-3">
          <code className="font-mono text-xs text-primary">
            atb view
          </code>
        </div>

        <p className="mt-4 max-w-3xl">
          Workspace indexes are only available when the viewer is served by the ATB Agent. Single-bundle
          viewer mode opens one bundle directly and does not expose the full workspace index.
        </p>
      </section>
    </main>
  );
};

export default SessionsPage;
