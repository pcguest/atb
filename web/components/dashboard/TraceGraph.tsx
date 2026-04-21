"use client";

import { useMemo } from "react";
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  type Edge,
  type Node,
} from "reactflow";
import "reactflow/dist/style.css";

import type { BundleGraphResponse } from "@/lib/types";

type TraceGraphProps = {
  graph: BundleGraphResponse | null;
  disabled?: boolean;
  onSelectSeq?: (seq: number) => void;
  layout?: "dagre-top-down";
};

function nodeColor(type: string): string {
  switch (type) {
    case "llm":
      return "#2563eb";
    case "tool":
      return "#d97706";
    case "chain":
      return "#059669";
    default:
      return "#475569";
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
        data: { label: node.label, eventType: node.type },
        position,
        style: {
          border: `1px solid ${nodeColor(node.type)}`,
          borderRadius: 4,
          padding: 8,
          background: "#0f172a",
          color: "#e2e8f0",
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
      style: { stroke: "#475569" },
      labelStyle: { fill: "#94a3b8", fontSize: 10 },
    }));
  }, [graph]);

  if (!graph || graph.nodes.length === 0) {
    return (
      <div className="flex h-full items-center justify-center border border-slate-800 bg-slate-900 text-sm text-slate-500">
        No graph data available.
      </div>
    );
  }

  return (
    <div className="h-full overflow-hidden border border-slate-800 bg-slate-950">
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
        <Background color="#1f2937" gap={16} />
      </ReactFlow>
    </div>
  );
}
