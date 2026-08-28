/**
 * Internal HTTP client for the local ATB Agent capture API (v1).
 * Not part of the public SDK surface.
 */
import { spawnSync } from "node:child_process";
import { isIP } from "node:net";
import { URL } from "node:url";
import type { WorkflowEventSink } from "./workflow-common.js";

export const DEFAULT_AGENT_URL = "http://127.0.0.1:6180";
export const MAX_AGENT_RESPONSE_BYTES = 1 * 1024 * 1024;

export interface AgentOpenParams {
  actorId?: string;
  purposeTag?: string;
  profileId?: string;
  bundlePath?: string;
}

export interface AgentOpenResult {
  sessionId: string;
  bundlePath: string;
  actorId?: string;
  profileId?: string;
  purposeTag?: string;
}

export interface AgentAppendEvent {
  eventType: string;
  payload?: Record<string, unknown>;
}

export interface AgentCloseParams {
  snapshotName?: string;
}

export interface AgentCloseResult {
  sessionId: string;
  bundlePath: string;
  profileId?: string;
  headHash: string;
  eventCount: number;
  openedAt: string;
  closedAt: string;
}

export interface AgentHttpResponse {
  status: number;
  body: string;
}

export type AgentRequestFn = (
  url: string,
  init: {
    method: string;
    body?: string;
    headers?: Record<string, string>;
    timeoutMs?: number;
  }
) => AgentHttpResponse;

export class AgentClientError extends Error {
  readonly status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "AgentClientError";
    this.status = status;
  }
}

/** Resolve agent base URL from environment. */
export function resolveAgentBaseUrl(
  env: Record<string, string | undefined> = process.env
): string | null {
  if (env.ATB_AGENT_DISABLE === "1" || env.ATB_AGENT_DISABLE === "true") {
    return null;
  }
  const explicit = env.ATB_AGENT_URL?.trim();
  if (explicit === "0" || explicit === "false") {
    return null;
  }
  if (explicit) {
    return explicit.replace(/\/$/, "");
  }
  if (env.ATB_AGENT_AUTO === "1" || env.ATB_AGENT_AUTO === "true") {
    return DEFAULT_AGENT_URL;
  }
  return null;
}

/** True when ATB_AGENT_URL is set explicitly (skip health probe). */
export function isExplicitAgentUrl(
  env: Record<string, string | undefined> = process.env
): boolean {
  const explicit = env.ATB_AGENT_URL?.trim();
  return Boolean(explicit && explicit !== "0" && explicit !== "false");
}

/** Synchronous GET /healthz for local agent auto-discovery. */
export function probeAgentHealth(
  baseUrl: string,
  requestFn: AgentRequestFn = syncAgentRequest
): boolean {
  try {
    const response = requestFn(`${baseUrl}/healthz`, {
      method: "GET",
      headers: { Accept: "application/json" },
      timeoutMs: 250,
    });
    if (response.status !== 200) {
      return false;
    }
    const parsed = JSON.parse(response.body) as { status?: string };
    return parsed.status === "ok";
  } catch {
    return false;
  }
}

export interface AgentClientOptions {
  baseUrl: string;
  requestFn?: AgentRequestFn;
}

/** HTTP client for Agent session open / event / close. */
export class AgentClient implements WorkflowEventSink {
  readonly baseUrl: string;
  private readonly requestFn: AgentRequestFn;
  private sessionId?: string;
  bundlePath?: string;

  constructor(options: AgentClientOptions) {
    this.baseUrl = validatedAgentBaseUrl(options.baseUrl);
    this.requestFn = options.requestFn ?? syncAgentRequest;
  }

  get activeSessionId(): string | undefined {
    return this.sessionId;
  }

  openSession(params: AgentOpenParams = {}): AgentOpenResult {
    const body: Record<string, string> = {};
    if (params.actorId?.trim()) body.actor_id = params.actorId.trim();
    if (params.purposeTag?.trim()) body.purpose_tag = params.purposeTag.trim();
    if (params.profileId?.trim()) body.profile_id = params.profileId.trim();
    if (params.bundlePath?.trim()) body.bundle_path = params.bundlePath.trim();

    const response = this.post("/v1/session/open", body);
    const parsed = parseJson<AgentOpenResult & { session_id?: string; bundle_path?: string }>(
      response,
      "open session"
    );
    const sessionId = parsed.sessionId ?? parsed.session_id;
    const bundlePath = parsed.bundlePath ?? parsed.bundle_path;
    if (!sessionId || !bundlePath) {
      throw new AgentClientError("agent open session: missing session_id or bundle_path", response.status);
    }
    this.sessionId = sessionId;
    this.bundlePath = bundlePath;
    return {
      sessionId,
      bundlePath,
      actorId: parsed.actorId,
      profileId: parsed.profileId,
      purposeTag: parsed.purposeTag,
    };
  }

  append(eventType: string, payload: Record<string, unknown>): void {
    this.appendEvent({ eventType, payload });
  }

  appendEvent(event: AgentAppendEvent): void {
    const sessionId = this.requireSessionId();
    const response = this.post(`/v1/session/${encodeURIComponent(sessionId)}/event`, {
      event_type: event.eventType,
      payload: event.payload ?? {},
    });
    if (response.status !== 202 && response.status !== 204) {
      throw agentError(response, "append event");
    }
  }

  closeSession(params: AgentCloseParams = {}): AgentCloseResult {
    const sessionId = this.requireSessionId();
    const body: Record<string, string> = {};
    if (params.snapshotName?.trim()) {
      body.snapshot_name = params.snapshotName.trim();
    }
    const response = this.post(
      `/v1/session/${encodeURIComponent(sessionId)}/close`,
      body
    );
    const parsed = parseJson<
      AgentCloseResult & {
        session_id?: string;
        bundle_path?: string;
        head_hash?: string;
        event_count?: number;
        opened_at?: string;
        closed_at?: string;
      }
    >(response, "close session");
    const result: AgentCloseResult = {
      sessionId: parsed.sessionId ?? parsed.session_id ?? sessionId,
      bundlePath: parsed.bundlePath ?? parsed.bundle_path ?? this.bundlePath ?? "",
      profileId: parsed.profileId,
      headHash: parsed.headHash ?? parsed.head_hash ?? "",
      eventCount: parsed.eventCount ?? parsed.event_count ?? 0,
      openedAt: parsed.openedAt ?? parsed.opened_at ?? "",
      closedAt: parsed.closedAt ?? parsed.closed_at ?? "",
    };
    this.sessionId = undefined;
    return result;
  }

  private requireSessionId(): string {
    if (!this.sessionId) {
      throw new Error("agent session is not open");
    }
    return this.sessionId;
  }

  private post(path: string, body: Record<string, unknown>): AgentHttpResponse {
    return this.requestFn(`${this.baseUrl}${path}`, {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(body),
      timeoutMs: 30_000,
    });
  }
}

/** Create a client when the agent is configured and reachable. */
export function tryCreateAgentClient(
  env: Record<string, string | undefined> = process.env,
  requestFn?: AgentRequestFn
): AgentClient | null {
  const baseUrl = resolveAgentBaseUrl(env);
  if (!baseUrl) {
    return null;
  }
  const fn = requestFn ?? syncAgentRequest;
  if (!isExplicitAgentUrl(env) && !probeAgentHealth(baseUrl, fn)) {
    return null;
  }
  return new AgentClient({ baseUrl, requestFn: fn });
}

export function syncAgentRequest(
  url: string,
  init: {
    method: string;
    body?: string;
    headers?: Record<string, string>;
    timeoutMs?: number;
  }
): AgentHttpResponse {
  const target = validatedAgentRequestUrl(url);
  const timeoutMs = init.timeoutMs ?? 30_000;
  // Node cannot service an asynchronous socket callback while the calling API
  // remains synchronous. Run the small loopback request in a child Node process
  // so AutomationSession keeps its existing synchronous event-sink contract
  // without blocking the transport's event loop.
  const result = spawnSync(process.execPath, ["-e", AGENT_REQUEST_HELPER], {
    input: JSON.stringify({
      url: target.href,
      method: init.method,
      body: init.body,
      headers: init.headers,
      timeoutMs,
      maxBytes: MAX_AGENT_RESPONSE_BYTES,
    }),
    encoding: "utf8",
    timeout: timeoutMs + 1_000,
    maxBuffer: MAX_AGENT_RESPONSE_BYTES + 64 * 1024,
    windowsHide: true,
  });
  if (result.error) {
    if ((result.error as NodeJS.ErrnoException).code === "ETIMEDOUT") {
      throw new Error(`agent request timed out: ${url}`);
    }
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(result.stderr.trim() || `agent request failed: ${url}`);
  }
  try {
    return JSON.parse(result.stdout) as AgentHttpResponse;
  } catch {
    throw new Error(`agent request returned an invalid response: ${url}`);
  }
}

const AGENT_REQUEST_HELPER = String.raw`
const http = require("node:http");
let input = "";
process.stdin.setEncoding("utf8");
process.stdin.on("data", chunk => { input += chunk; });
process.stdin.on("end", () => {
  const options = JSON.parse(input);
  const target = new URL(options.url);
  const req = http.request({
    hostname: target.hostname,
    port: target.port || 80,
    path: target.pathname + target.search,
    method: options.method,
    headers: options.headers,
    timeout: options.timeoutMs,
  }, res => {
    const chunks = [];
    let size = 0;
    let rejected = false;
    res.on("data", chunk => {
      if (rejected) return;
      size += chunk.length;
      if (size > options.maxBytes) {
        rejected = true;
        process.stderr.write("agent response exceeds " + options.maxBytes + " bytes");
        process.exitCode = 1;
        res.destroy();
        return;
      }
      chunks.push(chunk);
    });
    res.on("end", () => {
      if (rejected) return;
      process.stdout.write(JSON.stringify({
        status: res.statusCode || 0,
        body: Buffer.concat(chunks).toString("utf8"),
      }));
    });
    res.on("close", () => {
      if (rejected) process.exit();
    });
  });
  req.on("timeout", () => req.destroy(new Error("request timed out")));
  req.on("error", error => {
    process.stderr.write(error.message);
    process.exitCode = 1;
  });
  if (options.body) req.write(options.body);
  req.end();
});
`;

function validatedAgentRequestUrl(value: string): URL {
  const target = new URL(value);
  validatedAgentBaseUrl(target.origin);
  return target;
}

function validatedAgentBaseUrl(value: string): string {
  const target = new URL(value.replace(/\/$/, ""));
  const hostname = target.hostname.replace(/^\[|\]$/g, "").toLowerCase();
  const addressFamily = isIP(hostname);
  const loopback =
    hostname === "localhost" ||
    hostname === "::1" ||
    (addressFamily === 4 && hostname.startsWith("127."));
  if (
    target.protocol !== "http:" ||
    target.username ||
    target.password ||
    target.search ||
    target.hash ||
    (target.pathname && target.pathname !== "/") ||
    !loopback
  ) {
    throw new TypeError(
      "ATB Agent URL must be an unauthenticated loopback HTTP origin"
    );
  }
  return target.origin;
}

function parseJson<T>(response: AgentHttpResponse, action: string): T {
  if (response.status < 200 || response.status >= 300) {
    throw agentError(response, action);
  }
  try {
    return JSON.parse(response.body) as T;
  } catch {
    throw new AgentClientError(`agent ${action}: invalid JSON response`, response.status);
  }
}

function agentError(response: AgentHttpResponse, action: string): AgentClientError {
  let message = `agent ${action} failed (${response.status})`;
  try {
    const parsed = JSON.parse(response.body) as { error?: string };
    if (parsed.error) {
      message = parsed.error;
    }
  } catch {
    // ignore parse errors
  }
  return new AgentClientError(message, response.status);
}
