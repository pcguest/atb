"use client";

import { useMemo } from "react";
import ReactFlow, { Controls, type Edge, type Node } from "reactflow";
import "reactflow/dist/style.css";

import type { BundleGraphResponse } from "@/lib/types";

type TraceGraphProps = {
  graph: BundleGraphResponse | null;
  disabled?: boolean;
  onSelectSeq?: (seq: number) => void;
  layout?: "dagre-top-down";
};

export function TraceGraph({ graph, disabled = false, onSelectSeq }: TraceGraphProps) {
  const nodes = useMemo<Node[]>(() => {
    if (!graph) {
      return [];
    }

    return graph.nodes.map((node, idx) => {
      const position = { x: (idx % 2) * 340, y: Math.floor(idx / 2) * 140 };

      return {
        id: node.id,
        data: { label: node.label, eventType: node.event_type },
        position,
        style: {
          border: "1px solid hsl(var(--border))",
          borderRadius: 4,
          padding: 12,
          width: 250,
          fontSize: 12,
          textAlign: "left",
          background: "hsl(var(--card))",
          color: "hsl(var(--foreground))",
        },
      };
    });
  }, [graph]);

  const edges = useMemo<Edge[]>(() => {
    if (!graph) {
      return [];
    }
    return graph.edges.map((edge) => ({
      id: edge.id,
      source: edge.source,
      target: edge.target,
      label: edge.label,
      animated: false,
      style: {
        stroke: "hsl(var(--muted-foreground))",
        strokeDasharray: edge.label === "next" ? "4 4" : undefined,
      },
      labelStyle: { fill: "hsl(var(--foreground))", fontSize: 12 },
      labelBgStyle: { fill: "hsl(var(--card))" },
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
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={!disabled}
        fitView
      >
        <Controls showInteractive={false} />
      </ReactFlow>
    </div>
  );
}
