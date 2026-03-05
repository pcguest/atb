/**
 * Canonical ATB event model with optional multi-tenant identity fields.
 */

export interface Event {
  seq: number;
  prev_hash: string;
  type: string;
  data: unknown;
  actor_id?: string;
  org_id?: string;
  workspace_id?: string;
}

export interface AppendIdentityOptions {
  actorId?: string;
  orgId?: string;
  workspaceId?: string;
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

  return event;
}
