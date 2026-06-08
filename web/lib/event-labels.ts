import type { DashboardRole } from "@/lib/roles";

/** Friendly timeline labels for auditor and executive roles. */
export const FRIENDLY_EVENT_LABELS: Record<string, string> = {
  "atb.bundle.manifest": "Bundle manifest",
  "atb.bundle.anchor": "Timestamp anchor",
  "atb.bundle.signature": "Signature",
  "atb.bundle.pushed": "Bundle pushed",
  "atb.snapshot": "Snapshot",
  "atb.corroboration.external": "External corroboration",
  "atb.event.rag_index": "Knowledge index",
  "atb.event.rag_retrieval": "Knowledge retrieval",
  "ai.request.received": "Request received",
  "ai.response.sent": "Response sent",
  "ai.llm.call": "LLM call",
  "ai.tool.exec": "Tool execution",
  "ai.chain.run": "Reasoning chain",
  "ai.policy.decision": "Policy decision",
  "ai.retrieval.executed": "Retrieval executed",
  "ai.model.invoked": "Model invoked",
  "ai.model.output": "Model output",
  "ai.action.precommit": "Action precommit",
  "ai.action.executed": "Action executed",
  "ai.action.committed": "Action committed",
  "ai.human.approval": "Human approval",
  "ai.override.requested": "Override requested",
  "ai.job.scheduled": "Job scheduled",
  "ai.job.started": "Job started",
  "ai.job.step": "Job step",
  "ai.job.completed": "Job completed",
  "data.export.precommit": "Export precommit",
  "data.export.executed": "Export executed",
  "dev.session": "Developer session",
  "snapshot.build": "Build snapshot",
  "privacy.reveal": "Privacy reveal",
};

export function usesFriendlyEventLabels(role: DashboardRole): boolean {
  return role === "auditor" || role === "executive";
}

export function eventDisplayLabel(eventType: string, role: DashboardRole): string {
  if (!usesFriendlyEventLabels(role)) {
    return eventType;
  }
  return FRIENDLY_EVENT_LABELS[eventType] ?? eventType;
}
