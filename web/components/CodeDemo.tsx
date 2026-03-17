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
      cmd: `atb append dev.session '{"date":"2025-01-15","features":["hash chaining"]}'`,
      out: "✓ Appended event #1 [dev.session] hash=cdc87dac2d8d61bf...",
      delay: 300,
    },
    {
      prompt: "$",
      cmd: `atb append decision '{"choice":"Go over Rust","reason":"velocity"}'`,
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

# Append events with arbitrary JSON payloads
bundle.append("dev.session", {
    "date": "2025-01-15",
    "features_built": ["hash chaining", "CLI init"],
    "blockers": ["RFC 8785 library compatibility"],
})

bundle.append("decision", {
    "choice": "Go over Rust for CLI",
    "reason": "Solo founder velocity",
    "alternatives": ["Rust", "Python-only"],
})

# Save to disk (NDJSON format)
bundle.save("run.atb/bundle.atb")

# Later — reload and verify integrity
b = Bundle.load("run.atb/bundle.atb")
b.verify()  # Raises ATBVerificationError if tampered
print(f"✓ Verified {len(b)} events — chain intact.")

# LangChain integration
from atb.integrations.langchain import ATBCallbackHandler
handler = ATBCallbackHandler(bundle, auto_save=True)
llm = ChatOpenAI(callbacks=[handler])`,
  typescript: `import { Bundle } from "@pcguest/atb-sdk";

// Create a new bundle
const bundle = new Bundle();

// Append events with arbitrary JSON payloads
bundle.append("dev.session", {
  date: "2025-01-15",
  featuresBuilt: ["hash chaining", "CLI init"],
  blockers: ["RFC 8785 library compatibility"],
});

bundle.append("decision", {
  choice: "Go over Rust for CLI",
  reason: "Solo founder velocity",
  alternatives: ["Rust", "Python-only"],
});

// Save to disk (NDJSON format)
bundle.save("run.atb/bundle.atb");

// Later — reload and verify integrity
const loaded = Bundle.load("run.atb/bundle.atb");
loaded.verify(); // Throws ATBVerificationError if tampered
console.log(\`✓ Verified \${loaded.length} events — chain intact.\`);`,
};

function TerminalLine({
  line,
  index,
}: {
  line: { prompt: string; cmd: string; out: string };
  index: number;
}) {
  return (
    <div className="mb-3">
      <div className="flex items-start gap-2">
        <span className="text-indigo-400 font-mono text-sm select-none mt-0.5">{line.prompt}</span>
        <span className="font-mono text-sm text-[#e2e8f0]">{line.cmd}</span>
      </div>
      <div className="font-mono text-sm text-[#22c55e] mt-1 pl-4">{line.out}</div>
    </div>
  );
}

export default function CodeDemo() {
  const [activeTab, setActiveTab] = useState("cli");

  return (
    <section id="demo" className="py-24 relative">
      <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section header */}
        <div className="text-center mb-12">
          <h2 className="text-3xl sm:text-4xl font-bold text-white mb-4">Simple by Design</h2>
          <p className="text-[#9ca3af] text-lg max-w-2xl mx-auto">
            Three commands to get started. Works with your existing AI stack. No infrastructure
            required.
          </p>
        </div>

        {/* Terminal window */}
        <div className="rounded-xl border border-[#1e1e2e] bg-[#111118] overflow-hidden glow">
          {/* Window chrome */}
          <div className="flex items-center justify-between px-4 py-3 border-b border-[#1e1e2e] bg-[#0d0d14]">
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-[#ef4444]" />
              <div className="w-3 h-3 rounded-full bg-[#eab308]" />
              <div className="w-3 h-3 rounded-full bg-[#22c55e]" />
            </div>
            {/* Tabs */}
            <div className="flex items-center gap-1">
              {TABS.map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`px-3 py-1 rounded text-xs font-mono transition-all ${
                    activeTab === tab.id
                      ? "bg-indigo-600/20 text-indigo-300 border border-indigo-500/30"
                      : "text-[#6b7280] hover:text-[#9ca3af]"
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>
            <div className="w-16" />
          </div>

          {/* Code content */}
          <div className="p-6 min-h-[320px] overflow-x-auto">
            {activeTab === "cli" ? (
              <div>
                {(CODE.cli as Array<{ prompt: string; cmd: string; out: string }>).map(
                  (line, i) => (
                    <TerminalLine key={i} line={line} index={i} />
                  ),
                )}
                <div className="flex items-center gap-2 mt-2">
                  <span className="text-indigo-400 font-mono text-sm">$</span>
                  <span className="terminal-cursor font-mono text-sm text-[#e2e8f0]" />
                </div>
              </div>
            ) : (
              <pre className="font-mono text-sm text-[#e2e8f0] leading-relaxed whitespace-pre-wrap">
                <code>
                  {activeTab === "python"
                    ? (CODE.python as string).split("\n").map((line, i) => (
                        <span key={i}>
                          {line
                            .replace(/(from|import|def|class|return|if|for|in|as)/g, "<kw>$1</kw>")
                            .split(/(<kw>.*?<\/kw>)/)
                            .map((part, j) =>
                              part.startsWith("<kw>") ? (
                                <span key={j} className="text-indigo-400">
                                  {part.replace(/<\/?kw>/g, "")}
                                </span>
                              ) : part.startsWith("#") ? (
                                <span key={j} className="text-[#6b7280]">
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
                            <span className="text-[#6b7280]">{line}</span>
                          ) : (
                            line
                              .replace(
                                /(import|from|const|new|await|export|type|interface)/g,
                                "<kw>$1</kw>",
                              )
                              .split(/(<kw>.*?<\/kw>)/)
                              .map((part, j) =>
                                part.startsWith("<kw>") ? (
                                  <span key={j} className="text-indigo-400">
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
              className="flex items-center gap-3 p-3 rounded-lg border border-[#1e1e2e] bg-[#111118]/50"
            >
              <span className="text-[#6b7280] text-xs font-mono shrink-0">{item.label}</span>
              <code className="text-indigo-300 text-xs font-mono truncate">{item.cmd}</code>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
