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

export function TraceGraph({ graph, disabled = false, onSelectSeq }: TraceGraphProps) {
  const nodes = useMemo<Node[]>(() => {
    if (!graph) {
      return [];
    }

    return graph.nodes.map((node, idx) => ({
      id: node.id,
      data: { label: node.label, eventType: node.type },
      position: {
        x: (idx % 5) * 220,
        y: Math.floor(idx / 5) * 100,
      },
      style: {
        border: `1px solid ${nodeColor(node.type)}`,
        borderRadius: 8,
        padding: 8,
        background: "#0f172a",
        color: "#e2e8f0",
      },
    }));
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
      animated: edge.label === "parent",
      style: { stroke: "#475569" },
      labelStyle: { fill: "#94a3b8", fontSize: 10 },
    }));
  }, [graph]);

  if (!graph || graph.nodes.length === 0) {
    return (
      <div className="rounded-lg border border-slate-800 bg-slate-900 p-4 text-sm text-slate-400">
        No graph data available.
      </div>
    );
  }

  return (
    <div className="h-[320px] overflow-hidden rounded-lg border border-slate-800 bg-slate-950">
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
