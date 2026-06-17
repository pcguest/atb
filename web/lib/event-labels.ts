import type { DashboardRole } from "@/lib/roles";

/** Friendly timeline labels for auditor and executive roles. */
export const FRIENDLY_EVENT_LABELS: Record<string, string> = {
  "atb.bundle.manifest": "Bundle manifest",
  "atb.bundle.anchor": "Timestamp anchor",
  "atb.bundle.signature": "Signature",
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
  "ai.action.error": "Action error",
  "ai.human.approval": "Human approval",
  "ai.job.scheduled": "Job scheduled",
  "ai.job.started": "Job started",
  "ai.job.step": "Job step",
  "ai.job.completed": "Job completed",
  "data.export.precommit": "Export precommit",
  "data.export.executed": "Export executed",
  "data.export.error": "Export error",
  "data.retention.policy_set": "Retention policy set",
  "data.retention.policy_changed": "Retention policy changed",
  "data.retention.enforced": "Retention operation",
  "dev.session": "Developer session",
  "privacy.reveal": "Privacy reveal",
  "atb.capture.scope": "Capture scope",
  "atb.llm.request": "Captured LLM request",
  "atb.llm.response": "Captured LLM response",
  "atb.tool.call": "Captured tool call",
  "atb.human.approval": "Operator approval",
  "atb.human.override": "Operator override",
  "atb.exchange.complete": "Exchange complete",
  "atb.session.close": "Session closed",
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
