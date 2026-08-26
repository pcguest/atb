"use client";

import { useMemo } from "react";
import ReactFlow, { Background, Controls, MiniMap, type Edge, type Node } from "reactflow";
import "reactflow/dist/style.css";

import { eventFamily, type EventFamily } from "@/lib/event-family";
import type { BundleGraphResponse } from "@/lib/types";

type TraceGraphProps = {
  graph: BundleGraphResponse | null;
  disabled?: boolean;
  onSelectSeq?: (seq: number) => void;
  layout?: "dagre-top-down";
};

function nodeColor(type: string): string {
  const family: EventFamily =
    type === "llm" || type === "tool" || type === "chain" ? type : eventFamily(type);

  switch (family) {
    case "llm":
      return "hsl(var(--ev-llm))";
    case "tool":
      return "hsl(var(--ev-tool))";
    case "chain":
      return "hsl(var(--ev-chain))";
    case "policy":
      return "hsl(var(--ev-policy))";
    case "action":
      return "hsl(var(--ev-action))";
    case "human":
      return "hsl(var(--ev-human))";
    case "corroboration":
    case "export":
    case "retention":
      return "hsl(var(--ev-export))";
    default:
      return "hsl(var(--ev-default))";
  }
}

export function TraceGraph({ graph, disabled = false, onSelectSeq, layout }: TraceGraphProps) {
  const nodes = useMemo<Node[]>(() => {
    if (!graph) {
      return [];
    }

    return graph.nodes.map((node, idx) => {
      const position =
        layout === "dagre-top-down"
          ? { x: (idx % 3) * 260, y: Math.floor(idx / 3) * 120 }
          : { x: (idx % 5) * 220, y: Math.floor(idx / 5) * 100 };

      return {
        id: node.id,
        data: { label: node.label, eventType: node.event_type },
        position,
        style: {
          border: `1px solid ${nodeColor(node.event_type)}`,
          borderRadius: 4,
          padding: 8,
          background: "hsl(var(--card))",
          color: "hsl(var(--foreground))",
        },
      };
    });
  }, [graph, layout]);

  const edges = useMemo<Edge[]>(() => {
    if (!graph) {
      return [];
    }
    return graph.edges.map((edge) => ({
      id: edge.id,
      source: edge.source,
      target: edge.target,
      label: edge.label,
      animated: edge.label === "parent",
      style: { stroke: "hsl(var(--border))" },
      labelStyle: { fill: "hsl(var(--muted-foreground))", fontSize: 10 },
    }));
  }, [graph]);

  if (!graph || graph.nodes.length === 0) {
    return (
      <div className="flex h-full items-center justify-center border border-border bg-card text-sm text-muted-foreground">
        No graph data available.
      </div>
    );
  }

  return (
    <div className="h-full overflow-hidden border border-border bg-card">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodeClick={(_, node) => {
          if (disabled || !onSelectSeq || !node.id?.startsWith("evt-")) {
            return;
          }
          const raw = node.id.replace("evt-", "");
          const seq = Number(raw);
          if (Number.isFinite(seq)) {
            onSelectSeq(seq);
          }
        }}
        nodesDraggable={!disabled}
        nodesConnectable={false}
        elementsSelectable={!disabled}
        fitView
      >
        <MiniMap nodeColor={(node) => nodeColor(String(node.data.eventType ?? "event"))} />
        <Controls />
        <Background color="hsl(var(--border))" gap={16} />
      </ReactFlow>
    </div>
  );
}
