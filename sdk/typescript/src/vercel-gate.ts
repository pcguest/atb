import { ActionGate } from "./action-gate.js";

export interface VercelAITool<T extends Record<string, unknown>> {
  name?: string;
  description?: string;
  parameters?: unknown;
  execute: (parameters: T) => Promise<unknown> | unknown;
}

export function gateVercelTool<T extends Record<string, unknown>>(
  toolDef: VercelAITool<T>,
  gate: ActionGate,
): VercelAITool<T> {
  return {
    ...toolDef,
    execute: async (parameters: T) =>
      gate.run(
        {
          actionType: toolDef.name ?? "tool",
          targetResourceId: toolDef.name ?? "tool",
          intendedEffect: toolDef.description ?? "",
          actionParameters: parameters,
          subjectId: undefined,
        },
        () => toolDef.execute(parameters),
      ),
  };
}
