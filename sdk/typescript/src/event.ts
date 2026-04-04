/**
 * Canonical ATB event model aligned with the Go runtime.
 */

export interface Event {
  seq: number;
  prev_hash: string;
  type: string;
  data: unknown;
  hash_algo?: "sha256";
  actor_id?: string;
  org_id?: string;
  workspace_id?: string;
  timestamp?: string;
  trace_id?: string;
  span_id?: string;
  parent_span_id?: string;
}

export interface AppendIdentityOptions {
  actorId?: string;
  orgId?: string;
  workspaceId?: string;
  timestamp?: string;
  traceId?: string;
  spanId?: string;
  parentSpanId?: string;
}

export function normalizeOptionalIdentity(
  value: string | undefined
): string | undefined {
  if (value === undefined) {
    return undefined;
  }
  const trimmed = value.trim();
  if (trimmed === "") {
    return undefined;
  }
  return trimmed;
}

export function prepareForCanonical(event: Event): Record<string, unknown> {
  const out: Record<string, unknown> = {
    seq: event.seq,
    prev_hash: event.prev_hash,
    type: event.type,
    data: event.data,
  };
  if (event.actor_id !== undefined) {
    out.actor_id = event.actor_id;
  }
  if (event.org_id !== undefined) {
    out.org_id = event.org_id;
  }
  if (event.workspace_id !== undefined) {
    out.workspace_id = event.workspace_id;
  }
  if (event.hash_algo !== undefined) {
    out.hash_algo = event.hash_algo;
  }
  if (event.timestamp !== undefined) {
    out.timestamp = event.timestamp;
  }
  if (event.trace_id !== undefined) {
    out.trace_id = event.trace_id;
  }
  if (event.span_id !== undefined) {
    out.span_id = event.span_id;
  }
  if (event.parent_span_id !== undefined) {
    out.parent_span_id = event.parent_span_id;
  }
  return out;
}

export function parseEvent(value: unknown): Event {
  if (!value || typeof value !== "object") {
    throw new TypeError("event must be an object");
  }
  const raw = value as Record<string, unknown>;
  if (
    typeof raw.seq !== "number" ||
    typeof raw.prev_hash !== "string" ||
    typeof raw.type !== "string"
  ) {
    throw new TypeError("event must include seq, prev_hash, and type");
  }

  const event: Event = {
    seq: raw.seq,
    prev_hash: raw.prev_hash,
    type: raw.type,
    data: raw.data,
  };

  if (raw.actor_id !== undefined) {
    if (typeof raw.actor_id !== "string") {
      throw new TypeError("event.actor_id must be a string");
    }
    event.actor_id = raw.actor_id;
  }
  if (raw.org_id !== undefined) {
    if (typeof raw.org_id !== "string") {
      throw new TypeError("event.org_id must be a string");
    }
    event.org_id = raw.org_id;
  }
  if (raw.workspace_id !== undefined) {
    if (typeof raw.workspace_id !== "string") {
      throw new TypeError("event.workspace_id must be a string");
    }
    event.workspace_id = raw.workspace_id;
  }
  if (raw.hash_algo !== undefined) {
    if (raw.hash_algo !== "sha256") {
      throw new TypeError("event.hash_algo must be sha256");
    }
    event.hash_algo = raw.hash_algo;
  }
  if (raw.timestamp !== undefined) {
    if (typeof raw.timestamp !== "string") {
      throw new TypeError("event.timestamp must be a string");
    }
    event.timestamp = raw.timestamp;
  }
  if (raw.trace_id !== undefined) {
    if (typeof raw.trace_id !== "string") {
      throw new TypeError("event.trace_id must be a string");
    }
    event.trace_id = raw.trace_id;
  }
  if (raw.span_id !== undefined) {
    if (typeof raw.span_id !== "string") {
      throw new TypeError("event.span_id must be a string");
    }
    event.span_id = raw.span_id;
  }
  if (raw.parent_span_id !== undefined) {
    if (typeof raw.parent_span_id !== "string") {
      throw new TypeError("event.parent_span_id must be a string");
    }
    event.parent_span_id = raw.parent_span_id;
  }

  return event;
}
