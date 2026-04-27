import { ActionGate } from "./action-gate.js";

/** Minimal Vercel AI tool shape accepted by {@link gateVercelTool}. */
export interface VercelAITool<T extends Record<string, unknown>> {
  name?: string;
  description?: string;
  parameters?: unknown;
  execute: (parameters: T) => Promise<unknown> | unknown;
}

/**
 * @param toolDef Tool definition to wrap.
 * @param gate Action gate used to record and optionally enforce policy.
 * @returns Wrapped tool definition with gated execution.
 * @throws ActionGateDeniedError when the gate denies in enforce mode.
 */
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
