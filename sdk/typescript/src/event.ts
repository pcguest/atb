/**
 * Canonical ATB event model aligned with the Go runtime.
 */

/**
 * Acquisition checkpoint state recorded with an imported event. Operational
 * position, not evidence truth. Mirrors the canonical Go `CheckpointInfo`.
 */
export interface AcquisitionCheckpoint {
  source_system: string;
  acquisition_stream: string;
  position: string;
  observed_at: string;
  adapter: string;
  adapter_version?: string;
}

/**
 * Optional acquisition provenance: how an event entered ATB (import/capture).
 * It is a canonical, hashed top-level envelope field when present. It is not a
 * truth claim about the acquired content, does not authorise the source, and is
 * not a trust score. Mirrors the canonical Go `AcquisitionInfo`.
 */
export interface Acquisition {
  mode?: string;
  source_system?: string;
  source_record_id?: string;
  source_timestamp?: string;
  acquired_at?: string;
  source_digest?: string;
  adapter?: string;
  adapter_version?: string;
  checkpoint?: AcquisitionCheckpoint;
}

/** Canonical ATB event shape used for hashing and bundle records. */
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
  acquisition?: Acquisition;
}

/** Optional identity and trace metadata accepted when appending events. */
export interface AppendIdentityOptions {
  actorId?: string;
  orgId?: string;
  workspaceId?: string;
  timestamp?: string;
  traceId?: string;
  spanId?: string;
  parentSpanId?: string;
}

/**
 * Canonical acquisition string fields (all optional, omitted when unset).
 */
const ACQUISITION_STRING_FIELDS = [
  "mode",
  "source_system",
  "source_record_id",
  "source_timestamp",
  "acquired_at",
  "source_digest",
  "adapter",
  "adapter_version",
] as const;

/**
 * Canonical checkpoint fields that the runtime always emits when a checkpoint
 * is present (no `omitempty` on the Go struct).
 */
const CHECKPOINT_REQUIRED_FIELDS = [
  "source_system",
  "acquisition_stream",
  "position",
  "observed_at",
  "adapter",
] as const;

function requireObject(value: unknown, label: string): Record<string, unknown> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    throw new TypeError(`${label} must be an object`);
  }
  return value as Record<string, unknown>;
}

function requireStringField(
  obj: Record<string, unknown>,
  key: string,
  label: string
): string {
  const value = obj[key];
  if (typeof value !== "string") {
    throw new TypeError(`${label}.${key} must be a string`);
  }
  return value;
}

/**
 * Validate and copy a canonical acquisition object.
 *
 * Only recognised canonical fields are preserved. Unknown acquisition fields
 * are dropped rather than passed through, so they never become canonical
 * evidence. This fails closed: a record whose stored hash included an
 * unrecognised acquisition field will not verify, rather than being silently
 * accepted. Structural types are enforced here; schema-level constraints
 * (patterns, formats, minLength) are enforced by schemas/event.v1.json.
 *
 * @param value Unknown acquisition value from parsed JSON.
 * @returns A normalised acquisition object with unset optional fields omitted.
 * @throws TypeError when the structure or a field type is invalid.
 */
export function parseAcquisition(value: unknown): Acquisition {
  const raw = requireObject(value, "event.acquisition");
  const acquisition: Record<string, unknown> = {};
  for (const field of ACQUISITION_STRING_FIELDS) {
    if (raw[field] !== undefined) {
      acquisition[field] = requireStringField(raw, field, "event.acquisition");
    }
  }
  if (raw.checkpoint !== undefined) {
    const checkpointRaw = requireObject(
      raw.checkpoint,
      "event.acquisition.checkpoint"
    );
    const checkpoint: Record<string, unknown> = {};
    for (const field of CHECKPOINT_REQUIRED_FIELDS) {
      checkpoint[field] = requireStringField(
        checkpointRaw,
        field,
        "event.acquisition.checkpoint"
      );
    }
    if (checkpointRaw.adapter_version !== undefined) {
      checkpoint.adapter_version = requireStringField(
        checkpointRaw,
        "adapter_version",
        "event.acquisition.checkpoint"
      );
    }
    acquisition.checkpoint = checkpoint;
  }
  return acquisition as unknown as Acquisition;
}

/**
 * @param value Optional identity string.
 * @returns Trimmed string, or `undefined` when unset or blank.
 */
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

/**
 * @param event Event to convert into canonical hash input.
 * @returns Plain object with unset optional fields omitted.
 */
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
  if (event.acquisition !== undefined) {
    out.acquisition = event.acquisition;
  }
  return out;
}

/**
 * @param value Unknown value from parsed JSON.
 * @returns Validated event object.
 * @throws TypeError when required fields or optional field types are invalid.
 */
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
  if (raw.acquisition !== undefined) {
    event.acquisition = parseAcquisition(raw.acquisition);
  }

  return event;
}
