// Event family categorisation shared by the timeline (colour) and the stats
// overview (counts). Keeping a single source of truth ensures capture and
// accountability event types — atb.tool.call, atb.llm.*, ai.action.* — are
// categorised consistently rather than falling through to "other"/grey.

export type EventFamily =
  | "llm"
  | "tool"
  | "chain"
  | "policy"
  | "action"
  | "human"
  | "job"
  | "corroboration"
  | "export"
  | "other";

/**
 * eventFamily maps a canonical ATB event type to its display family. Both the
 * `ai.*` (SDK/runtime) and `atb.*` (capture/proxy) namespaces are recognised,
 * so e.g. `atb.tool.call` groups with tools and `atb.llm.response` with LLMs.
 */
export function eventFamily(eventType: string): EventFamily {
  const t = eventType ?? "";
  if (t.startsWith("ai.llm") || t.startsWith("atb.llm")) return "llm";
  if (t.startsWith("ai.tool") || t === "atb.tool.call") return "tool";
  if (t.startsWith("ai.chain")) return "chain";
  if (t.startsWith("ai.policy")) return "policy";
  if (t.startsWith("ai.action")) return "action";
  if (t.startsWith("ai.human") || t.startsWith("atb.human")) return "human";
  if (t.startsWith("ai.job")) return "job";
  if (t.startsWith("atb.corroboration")) return "corroboration";
  if (
    t.startsWith("ai.export") ||
    t.startsWith("data.export") ||
    t.startsWith("atb.data.export")
  )
    return "export";
  return "other";
}

const familyClass: Record<EventFamily, string> = {
  llm: "ev-llm",
  tool: "ev-tool",
  chain: "ev-chain",
  policy: "ev-policy",
  action: "ev-action",
  human: "ev-human",
  job: "ev-default",
  corroboration: "ev-export",
  export: "ev-export",
  other: "ev-default",
};

/** eventFamilyClass returns the CSS colour class for an event type's family. */
export function eventFamilyClass(eventType: string): string {
  return familyClass[eventFamily(eventType)];
}
