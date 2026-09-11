import { z } from "zod";

import { profileReportSummarySchema } from "@/lib/schemas/profile";

export const investigationOverviewSchema = z.object({
  bundle_path: z.string(),
  event_count: z.number().int().nonnegative(),
  integrity_valid: z.boolean(),
  integrity_status: z.string(),
  profile: profileReportSummarySchema.nullish(),
  finding_count: z.number().int().nonnegative(),
  critical_findings: z.number().int().nonnegative(),
  custody_state: z.string(),
  summary: z.string(),
});

export const investigationFindingSchema = z.object({
  flag: z.string(),
  severity: z.string(),
  title: z.string(),
  detail: z.string(),
  basis: z.string(),
  session_id: z.string().optional(),
  profile_id: z.string().optional(),
  rule_id: z.string().optional(),
  boundedness: z.string(),
  what_atb_can_conclude: z.string(),
  what_atb_cannot_conclude: z.string(),
  event_seqs: z.array(z.number().int()).optional().default([]),
});

export const investigationFindingsSchema = z.object({
  findings: z.array(investigationFindingSchema),
});

export const timelineEventSchema = z.object({
  seq: z.number().int().nonnegative(),
  type: z.string(),
  label: z.string(),
  timestamp: z.string().optional(),
  hash: z.string(),
  family: z.string(),
  causal_edge: z.boolean(),
});

export const investigationTimelineSchema = z.object({ events: z.array(timelineEventSchema) });

const provenanceValueSchema = z.enum(["asserted", "derived", "attested"]);
export const contextUnitSchema = z.object({
  id: z.string(),
  kind: z.string(),
  digest: z.string(),
  source: z.string().optional(),
  parent_ids: z.array(z.string()),
  scope: z.string().optional(),
  sensitivity: z.string().optional(),
  created_at: z.string().optional(),
  metadata_provenance: z.record(z.string(), provenanceValueSchema).optional(),
  event_sequence: z.number().int(),
});

export const contextOperationSchema = z.object({
  id: z.string(),
  type: z.string(),
  input_unit_ids: z.array(z.string()),
  output_unit_ids: z.array(z.string()),
  method: z.string().optional(),
  processor_id: z.string().optional(),
  processor_version: z.string().optional(),
  input_token_count: z.number().int().nonnegative().optional(),
  output_token_count: z.number().int().nonnegative().optional(),
  preserved_references: z.array(z.string()).optional(),
  dropped_references: z.array(z.string()).optional(),
  invocation_id: z.string().optional(),
  event_sequence: z.number().int(),
});

export const evidenceCapabilitySchema = z.object({
  name: z.string(),
  mapping_version: z.string(),
  event_sequence: z.number().int(),
  raw_event_type: z.string(),
  fields: z.record(z.string(), z.unknown()),
});

export const investigationContextSchema = z.object({
  lineage: z.object({
    units: z.array(contextUnitSchema),
    operations: z.array(contextOperationSchema),
    warnings: z.array(z.string()),
  }),
  capabilities: z.array(evidenceCapabilitySchema),
});

export const relationshipSchema = z.object({
  id: z.string(),
  source_seq: z.number().int(),
  target_seq: z.number().int(),
  kind: z.string(),
  evidence_value: z.string(),
  strength: z.string(),
});

export const investigationRelationshipsSchema = z.object({
  relationships: z.array(relationshipSchema),
});

export const investigationTrustSchema = z.object({
  proof_statement: z.string(),
  integrity_valid: z.boolean(),
  canonicalisation: z.string(),
  signature_status: z.string(),
  anchor_status: z.string(),
  profile_id: z.string().optional(),
  profile_pass: z.boolean(),
  coverage_score: z.number().optional().default(0),
  coverage_grade: z.string().optional().default(""),
  assessment_coverage: z.number().optional().default(0),
  assurance_valid: z.boolean(),
  external_corroboration: z.boolean(),
  custody_state: z.string(),
  limitations: z.array(z.string()),
});
