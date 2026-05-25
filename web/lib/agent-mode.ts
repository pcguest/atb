const agentModeBuildFlag = process.env.NEXT_PUBLIC_ATB_AGENT_MODE === "1";
const agentProbeTimeoutMs = 2000;

let cachedAgentMode: boolean | null = null;

/** True when the viewer build targets the ATB Agent (build-time flag). */
export function isAgentModeBuildTime(): boolean {
  return agentModeBuildFlag;
}

/**
 * Detect whether the viewer is backed by the ATB Agent workspace API.
 * Build-time flag wins; otherwise probe GET /v1/workspace/bundles once with a short timeout.
 */
export async function detectAgentMode(): Promise<boolean> {
  if (agentModeBuildFlag) {
    return true;
  }
  if (cachedAgentMode !== null) {
    return cachedAgentMode;
  }
  if (typeof window === "undefined") {
    return false;
  }

  const controller = new AbortController();
  const timer = window.setTimeout(() => controller.abort(), agentProbeTimeoutMs);
  try {
    const response = await fetch("/v1/workspace/bundles", {
      method: "GET",
      cache: "no-store",
      signal: controller.signal,
    });
    cachedAgentMode = response.ok;
  } catch {
    cachedAgentMode = false;
  } finally {
    window.clearTimeout(timer);
  }
  return cachedAgentMode;
}

/** Reset cached probe result (tests only). */
export function resetAgentModeCacheForTests(): void {
  cachedAgentMode = null;
}
