const agentGuideURL =
  "https://github.com/pcguest/atb/blob/main/docs/guides/agent.md";

export function WorkspaceWelcome() {
  return (
    <div
      className="rounded-md border border-border bg-muted/30 p-5 text-sm text-muted-foreground"
      role="status"
      data-testid="workspace-welcome"
    >
      <h2 className="text-base font-medium text-foreground">No bundles yet</h2>
      <p className="mt-2">
        The Agent workspace is empty. Closed session bundles appear here after capture completes.
      </p>
      <ul className="mt-3 list-disc space-y-1 pl-5">
        <li>
          Run <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">atb agent run</code>{" "}
          in a terminal and keep it running on loopback.
        </li>
        <li>
          Instrument a workflow with TypeScript{" "}
          <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">AutomationSession</code>{" "}
          (<code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">ATB_AGENT_URL</code> /{" "}
          <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">ATB_AGENT_AUTO</code>) or
          run{" "}
          <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">atb capture run</code>.
        </li>
      </ul>
      <p className="mt-4">
        <a
          href={agentGuideURL}
          className="font-mono text-xs text-primary underline-offset-2 hover:underline"
          target="_blank"
          rel="noopener noreferrer"
        >
          Agent guide (first-run and configuration)
        </a>
      </p>
    </div>
  );
}
