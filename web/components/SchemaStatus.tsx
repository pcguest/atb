"use client";

import React from "react";

import { useSchemaStatusQuery } from "@/lib/api-client";
import type { EventTypeStatus } from "@/lib/types";

interface RowState {
  label: string;
  className: string;
}

// rowState classifies a type's contract health so an operator can scan for
// drift at a glance: undeclared (a producer emitted a type the schema does not
// know), incomplete (a required field is missing), complete, or not observed.
function rowState(t: EventTypeStatus): RowState {
  if (!t.declared) {
    return { label: "undeclared", className: "text-danger" };
  }
  if (t.incomplete > 0) {
    return { label: `${t.incomplete} incomplete`, className: "text-warning" };
  }
  if (t.observed > 0) {
    return { label: "complete", className: "text-verified" };
  }
  return { label: "not observed", className: "text-muted-foreground" };
}

const SummaryCard: React.FC<{ label: string; value: number; emphasis?: boolean }> = ({
  label,
  value,
  emphasis,
}) => (
  <div className="bg-surface-2 rounded-md p-3 border border-border">
    <div className={`text-2xl font-bold ${emphasis && value > 0 ? "text-danger" : "text-foreground"}`}>
      {value}
    </div>
    <div className="text-xs text-muted-foreground uppercase tracking-wide">{label}</div>
  </div>
);

// SchemaStatus renders contract health from GET /api/v1/schema/status. It reads
// through the shared api-client so the viewer session token is attached; a raw
// fetch() would be rejected by the authenticated viewer.
const SchemaStatus: React.FC = () => {
  const { data, isLoading, error } = useSchemaStatusQuery(true);

  if (isLoading) return <div className="text-muted-foreground">Loading schema status…</div>;
  if (error) return <div className="text-danger">Error: {error.message}</div>;
  if (!data) return <div className="text-muted-foreground">No schema status available.</div>;

  return (
    <div className="bg-surface-1 text-foreground p-4 rounded-lg shadow-lg">
      <div className="flex items-baseline justify-between mb-4">
        <h2 className="text-xl font-semibold">Contract status</h2>
        <span className="text-xs text-muted-foreground font-mono">{data.schema_source}</span>
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-5">
        <SummaryCard label="Declared types" value={data.declared_types} />
        <SummaryCard label="Observed types" value={data.observed_types} />
        <SummaryCard label="Total events" value={data.total_events} />
        <SummaryCard label="Incomplete events" value={data.incomplete_events} emphasis />
      </div>

      {data.undeclared_types.length > 0 && (
        <div className="mb-4 rounded-md border border-danger/40 bg-danger/10 p-3 text-sm">
          <span className="font-semibold text-danger">Undeclared types observed:</span>{" "}
          <span className="font-mono text-danger">{data.undeclared_types.join(", ")}</span>
          <p className="text-danger mt-1 text-xs">
            A producer emitted an event type the schema does not declare. Add it to
            <span className="font-mono"> schemas/event.v1.json</span> or investigate the producer.
          </p>
        </div>
      )}

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-muted-foreground border-b border-border">
              <th className="py-2 pr-2 font-semibold">Event type</th>
              <th className="py-2 px-2 font-semibold">Criticality</th>
              <th className="py-2 px-2 font-semibold text-right">Observed</th>
              <th className="py-2 px-2 font-semibold">Required fields</th>
              <th className="py-2 pl-2 font-semibold">Status</th>
            </tr>
          </thead>
          <tbody>
            {data.types.map((t) => {
              const state = rowState(t);
              return (
                <tr key={t.type} className="border-b border-border hover:bg-muted/50">
                  <td className="py-1.5 pr-2 font-mono text-foreground">{t.type}</td>
                  <td className="py-1.5 px-2 text-muted-foreground">{t.criticality || "—"}</td>
                  <td className="py-1.5 px-2 text-right tabular-nums text-muted-foreground">{t.observed}</td>
                  <td className="py-1.5 px-2 font-mono text-xs text-muted-foreground">
                    {t.required_fields.length > 0 ? t.required_fields.join(", ") : "—"}
                  </td>
                  <td className={`py-1.5 pl-2 ${state.className}`}>
                    {state.label}
                    {t.missing_fields && t.missing_fields.length > 0 && (
                      <span className="block text-xs text-warning font-mono">
                        missing: {t.missing_fields.join(", ")}
                      </span>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default SchemaStatus;
