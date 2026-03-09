export interface APIError {
  error: string;
}

export interface VerificationResponse {
  status: "valid" | "invalid";
  message: string;
  bundle_path: string;
  chain_length: number;
  head_hash?: string;
}

export interface BundleMetaResponse {
  bundle_path: string;
  event_count: number;
  type_counts: Record<string, number>;
  first_timestamp?: string;
  last_timestamp?: string;
  verified: boolean;
  verification_message: string;
}

export interface EventRecord {
  seq: number;
  type: string;
  hash: string;
  prev_hash: string;
  timestamp?: string;
  trace_id?: string;
  span_id?: string;
  parent_span_id?: string;
  data: unknown;
}

export interface BundleEventsResponse {
  offset: number;
  limit: number;
  total: number;
  events: EventRecord[];
}

export interface GraphNode {
  id: string;
  label: string;
  type: string;
}

export interface GraphEdge {
  id: string;
  source: string;
  target: string;
  label?: string;
}

export interface BundleGraphResponse {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface PrivacyRevealRequest {
  seq: number;
  field_path: string;
  reason?: string;
}

export interface PrivacyRevealResponse {
  seq: number;
  field_path: string;
  value: unknown;
}
