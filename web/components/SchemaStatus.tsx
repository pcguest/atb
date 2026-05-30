"use client";

import React from 'react';
import { useQuery } from '@tanstack/react-query';

// Mirrors pkg/api/v1 EventTypeStatusDTO.
interface EventTypeStatus {
  type: string;
  criticality: string;
  declared: boolean;
  observed: number;
  required_fields: string[];
  incomplete: number;
  missing_fields?: string[];
}

// Mirrors pkg/api/v1 SchemaStatusResponse.
interface SchemaStatusResponse {
  schema_source: string;
  declared_types: number;
  observed_types: number;
  total_events: number;
  incomplete_events: number;
  undeclared_types: string[];
  types: EventTypeStatus[];
}

const fetchSchemaStatus = async (): Promise<SchemaStatusResponse> => {
  const response = await fetch('/api/v1/schema/status');
  if (!response.ok) {
    throw new Error('Failed to fetch schema status');
  }
  return response.json();
};

interface RowState {
  label: string;
  className: string;
}

// rowState classifies a type's contract health so an operator can scan for
// drift at a glance: undeclared (a producer emitted a type the schema does not
// know), incomplete (a required field is missing), complete, or not observed.
function rowState(t: EventTypeStatus): RowState {
  if (!t.declared) {
    return { label: 'undeclared', className: 'text-red-400' };
  }
  if (t.incomplete > 0) {
    return { label: `${t.incomplete} incomplete`, className: 'text-amber-400' };
  }
  if (t.observed > 0) {
    return { label: 'complete', className: 'text-emerald-400' };
  }
  return { label: 'not observed', className: 'text-gray-500' };
}

const SummaryCard: React.FC<{ label: string; value: number; emphasis?: boolean }> = ({
  label,
  value,
  emphasis,
}) => (
  <div className="bg-gray-800 rounded-md p-3 border border-gray-700">
    <div className={`text-2xl font-bold ${emphasis && value > 0 ? 'text-red-400' : 'text-gray-100'}`}>
      {value}
    </div>
    <div className="text-xs text-gray-400 uppercase tracking-wide">{label}</div>
  </div>
);

const SchemaStatus: React.FC = () => {
  const { data, isLoading, error } = useQuery<SchemaStatusResponse>({
    queryKey: ['schemaStatus'],
    queryFn: fetchSchemaStatus,
  });

  if (isLoading) return <div className="text-gray-300">Loading schema status...</div>;
  if (error) return <div className="text-red-400">Error: {(error as Error).message}</div>;
  if (!data) return <div className="text-gray-400">No schema status available.</div>;

  return (
    <div className="bg-gray-900 text-gray-100 p-4 rounded-lg shadow-lg">
      <div className="flex items-baseline justify-between mb-4">
        <h2 className="text-xl font-semibold">Contract status</h2>
        <span className="text-xs text-gray-500 font-mono">{data.schema_source}</span>
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-5">
        <SummaryCard label="Declared types" value={data.declared_types} />
        <SummaryCard label="Observed types" value={data.observed_types} />
        <SummaryCard label="Total events" value={data.total_events} />
        <SummaryCard label="Incomplete events" value={data.incomplete_events} emphasis />
      </div>

      {data.undeclared_types.length > 0 && (
        <div className="mb-4 rounded-md border border-red-800 bg-red-950/40 p-3 text-sm">
          <span className="font-semibold text-red-300">Undeclared types observed:</span>{' '}
          <span className="font-mono text-red-200">{data.undeclared_types.join(', ')}</span>
          <p className="text-red-300/80 mt-1 text-xs">
            A producer emitted an event type the schema does not declare. Add it to
            <span className="font-mono"> schemas/event.v1.json</span> or investigate the producer.
          </p>
        </div>
      )}

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-gray-400 border-b border-gray-700">
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
                <tr key={t.type} className="border-b border-gray-800 hover:bg-gray-800/50">
                  <td className="py-1.5 pr-2 font-mono text-gray-200">{t.type}</td>
                  <td className="py-1.5 px-2 text-gray-400">{t.criticality || '—'}</td>
                  <td className="py-1.5 px-2 text-right tabular-nums text-gray-300">{t.observed}</td>
                  <td className="py-1.5 px-2 font-mono text-xs text-gray-500">
                    {t.required_fields.length > 0 ? t.required_fields.join(', ') : '—'}
                  </td>
                  <td className={`py-1.5 pl-2 ${state.className}`}>
                    {state.label}
                    {t.missing_fields && t.missing_fields.length > 0 && (
                      <span className="block text-xs text-amber-300/80 font-mono">
                        missing: {t.missing_fields.join(', ')}
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
