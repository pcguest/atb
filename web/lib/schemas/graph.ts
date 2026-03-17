import { z } from "zod";

export const graphNodeSchema = z.object({
  id: z.string(),
  label: z.string(),
  type: z.string(),
});

export const graphEdgeSchema = z.object({
  id: z.string(),
  source: z.string(),
  target: z.string(),
  label: z.string().optional(),
});

export const bundleGraphResponseSchema = z.object({
  nodes: z.array(graphNodeSchema),
  edges: z.array(graphEdgeSchema),
});

export type BundleGraphResponse = z.infer<typeof bundleGraphResponseSchema>;
