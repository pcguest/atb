"use client";

import { useState } from "react";

const TABS = [
  { id: "cli", label: "CLI (Go)" },
  { id: "python", label: "Python SDK" },
  { id: "typescript", label: "TypeScript SDK" },
];

const CODE = {
  cli: [
    {
      prompt: "$",
      cmd: "atb init",
      out: "✓ Initialised ATB bundle at run.atb/bundle.atb",
      delay: 0,
    },
    {
      prompt: "$",
      cmd: `atb append agent.run '{"agent":"support-triage","workspace":"internal"}'`,
      out: "✓ Appended event #1 [agent.run] hash=cdc87dac2d8d61bf...",
      delay: 300,
    },
    {
      prompt: "$",
      cmd: `atb append decision '{"action":"mask_email","reason":"privacy_policy"}'`,
      out: "✓ Appended event #2 [decision] hash=e0b539b812dec40b...",
      delay: 600,
    },
    {
      prompt: "$",
      cmd: "atb verify",
      out: "✓ Bundle verified: 2 events, chain intact.",
      delay: 900,
    },
  ],
  python: `from atb import Bundle

# Create a new bundle
bundle = Bundle()

# Record an agent run without changing your storage model
bundle.append("agent.run", {
    "agent": "support-triage",
    "workspace": "internal",
    "request_id": "req-42",
})

bundle.append("decision", {
    "action": "mask_email",
    "reason": "privacy_policy",
    "reviewer": "atb",
})

# Save to disk (NDJSON format)
bundle.save("run.atb/bundle.atb")

# Later — reload and verify integrity
b = Bundle.load("run.atb/bundle.atb")
b.verify()  # Raises ATBVerificationError if tampered
print(f"✓ Verified {len(b)} events — chain intact.")

# LangChain integration
from atb.langchain_callback import ATBCallbackHandler
handler = ATBCallbackHandler(bundle, auto_save=True)
llm = ChatOpenAI(callbacks=[handler])`,
  typescript: `import { Bundle } from "@pcguest/atb-sdk";

// Create a new bundle
const bundle = new Bundle();

// Record an agent run without changing your storage model
bundle.append("agent.run", {
  agent: "support-triage",
  workspace: "internal",
  requestId: "req-42",
});

bundle.append("decision", {
  action: "mask_email",
  reason: "privacy_policy",
  reviewer: "atb",
});

// Save to disk (NDJSON format)
bundle.save("run.atb/bundle.atb");

// Later — reload and verify integrity
const loaded = Bundle.load("run.atb/bundle.atb");
loaded.verify(); // Throws ATBVerificationError if tampered
console.log(\`✓ Verified \${loaded.length} events — chain intact.\`);`,
};

function TerminalLine({ line }: { line: { prompt: string; cmd: string; out: string } }) {
  return (
    <div className="mb-3">
      <div className="flex items-start gap-2">
        <span className="mt-0.5 select-none font-mono text-sm text-primary">{line.prompt}</span>
        <span className="font-mono text-sm text-foreground">{line.cmd}</span>
      </div>
      <div className="mt-1 pl-4 font-mono text-sm text-verified">{line.out}</div>
    </div>
  );
}

export default function CodeDemo() {
  const [activeTab, setActiveTab] = useState("cli");

  return (
    <section id="demo" className="border-b border-border bg-background py-20">
      <div className="mx-auto max-w-5xl px-4 sm:px-6 lg:px-8">
        <div className="mb-10 max-w-3xl">
          <p className="font-mono text-xs font-semibold uppercase tracking-[0.16em] text-primary">
            One evidence format
          </p>
          <h2 className="mt-3 text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
            Use the interface that fits the workflow.
          </h2>
          <p className="mt-4 max-w-2xl text-lg leading-8 text-muted-foreground">
            The Go CLI, Python SDK, and TypeScript SDK write the same local bundle format and verify
            the same hash chain.
          </p>
        </div>

        <div className="overflow-hidden rounded-xl border border-border bg-card">
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border bg-surface-1 px-4 py-3">
            <span className="font-mono text-xs text-muted-foreground">capture.example</span>
            <div
              className="flex items-center gap-1"
              role="tablist"
              aria-label="Code example language"
              onKeyDown={(event) => {
                const current = TABS.findIndex((tab) => tab.id === activeTab);
                const next =
                  event.key === "ArrowRight"
                    ? (current + 1) % TABS.length
                    : event.key === "ArrowLeft"
                      ? (current + TABS.length - 1) % TABS.length
                      : event.key === "Home"
                        ? 0
                        : event.key === "End"
                          ? TABS.length - 1
                          : -1;
                if (next < 0) return;
                event.preventDefault();
                setActiveTab(TABS[next].id);
                document.getElementById(`code-tab-${TABS[next].id}`)?.focus();
              }}
            >
              {TABS.map((tab) => (
                <button
                  key={tab.id}
                  type="button"
                  onClick={() => setActiveTab(tab.id)}
                  role="tab"
                  id={`code-tab-${tab.id}`}
                  aria-controls="code-example-panel"
                  tabIndex={activeTab === tab.id ? 0 : -1}
                  aria-selected={activeTab === tab.id}
                  className={`min-h-8 rounded px-3 py-1 font-mono text-xs transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
                    activeTab === tab.id
                      ? "border border-primary/30 bg-primary/10 text-primary"
                      : "border border-transparent text-muted-foreground hover:bg-muted hover:text-foreground"
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </div>

          {/* Code content */}
          <div
            id="code-example-panel"
            aria-labelledby={`code-tab-${activeTab}`}
            tabIndex={0}
            className="min-h-[320px] overflow-x-auto bg-surface-code p-6 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
            role="tabpanel"
          >
            {activeTab === "cli" ? (
              <div>
                {(CODE.cli as Array<{ prompt: string; cmd: string; out: string }>).map(
                  (line, i) => (
                    <TerminalLine key={i} line={line} />
                  ),
                )}
                <div className="flex items-center gap-2 mt-2">
                  <span className="font-mono text-sm text-primary">$</span>
                  <span className="terminal-cursor font-mono text-sm text-foreground" />
                </div>
              </div>
            ) : (
              <pre className="whitespace-pre-wrap font-mono text-sm leading-relaxed text-foreground">
                <code>
                  {activeTab === "python"
                    ? (CODE.python as string).split("\n").map((line, i) => (
                        <span key={i}>
                          {line
                            .replace(/(from|import|def|class|return|if|for|in|as)/g, "<kw>$1</kw>")
                            .split(/(<kw>.*?<\/kw>)/)
                            .map((part, j) =>
                              part.startsWith("<kw>") ? (
                                <span key={j} className="text-primary">
                                  {part.replace(/<\/?kw>/g, "")}
                                </span>
                              ) : part.startsWith("#") ? (
                                <span key={j} className="text-muted-foreground">
                                  {part}
                                </span>
                              ) : (
                                <span key={j}>{part}</span>
                              ),
                            )}
                          {"\n"}
                        </span>
                      ))
                    : (CODE.typescript as string).split("\n").map((line, i) => (
                        <span key={i}>
                          {line.startsWith("//") ? (
                            <span className="text-muted-foreground">{line}</span>
                          ) : (
                            line
                              .replace(
                                /(import|from|const|new|await|export|type|interface)/g,
                                "<kw>$1</kw>",
                              )
                              .split(/(<kw>.*?<\/kw>)/)
                              .map((part, j) =>
                                part.startsWith("<kw>") ? (
                                  <span key={j} className="text-primary">
                                    {part.replace(/<\/?kw>/g, "")}
                                  </span>
                                ) : (
                                  <span key={j}>{part}</span>
                                ),
                              )
                          )}
                          {"\n"}
                        </span>
                      ))}
                </code>
              </pre>
            )}
          </div>
        </div>

        {/* Install commands */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mt-6">
          {[
            { label: "Go CLI", cmd: "go install github.com/pcguest/atb/cmd/atb@latest" },
            { label: "Python SDK", cmd: "pip install atb-sdk" },
            { label: "TypeScript SDK", cmd: "npm install @pcguest/atb-sdk" },
          ].map((item) => (
            <div
              key={item.label}
              className="flex min-w-0 flex-col items-start gap-2 rounded-md border border-border bg-surface-code p-3"
            >
              <span className="text-text-tertiary text-xs font-mono">{item.label}</span>
              <code className="break-all text-primary text-xs font-mono">{item.cmd}</code>
            </div>
          ))}
        </div>
        <p className="mt-3 text-xs leading-6 text-muted-foreground">
          These install commands use published releases. For the full embedded View, build from a
          checkout with <code className="text-foreground">make build</code>; <code>go install</code>{" "}
          provides the CLI and viewer-installation guidance.
        </p>
      </div>
    </section>
  );
}
