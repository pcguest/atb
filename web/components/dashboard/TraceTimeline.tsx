"use client";

import type { EventRecord } from "@/lib/types";
import { FixedSizeList as List, type ListChildComponentProps } from "react-window";

type TraceTimelineProps = {
  events: EventRecord[];
  total: number;
  hasMore: boolean;
  loadingMore: boolean;
  selectedSeq: number | null;
  onSelect: (seq: number) => void;
  onLoadMore: () => void;
  disabled?: boolean;
};

type RowData = {
  events: EventRecord[];
  selectedSeq: number | null;
  onSelect: (seq: number) => void;
  disabled: boolean;
};

const rowHeight = 72;
const nearEndThreshold = 20;

function eventTypeClass(eventType: string): string {
  if (eventType.startsWith("ai.llm")) {
    return "text-blue-200";
  }
  if (eventType.startsWith("ai.tool")) {
    return "text-amber-200";
  }
  if (eventType.startsWith("ai.chain")) {
    return "text-emerald-200";
  }
  return "text-slate-100";
}

function Row({ index, style, data }: ListChildComponentProps<RowData>) {
  const event = data.events[index];
  const isSelected = event.seq === data.selectedSeq;
  return (
    <div style={style} className="px-2 py-1">
      <button
        data-testid="event-list-item"
        type="button"
        disabled={data.disabled}
        onClick={() => data.onSelect(event.seq)}
        className={`flex h-full w-full items-center justify-between rounded-md border px-3 text-left transition ${
          isSelected
            ? "border-indigo-500 bg-indigo-950/40"
            : "border-slate-800 bg-slate-900 hover:border-slate-700"
        } disabled:cursor-not-allowed disabled:opacity-60`}
      >
        <div className="min-w-0">
          <div className={`truncate text-sm font-medium ${eventTypeClass(event.type)}`}>
            {event.type}
          </div>
          <div className="truncate text-xs text-slate-300">{event.timestamp || "no timestamp"}</div>
        </div>
        <div className="ml-3 shrink-0 rounded bg-slate-800 px-2 py-0.5 text-xs text-slate-100">
          #{event.seq}
        </div>
      </button>
    </div>
  );
}

export function TraceTimeline({
  events,
  total,
  hasMore,
  loadingMore,
  selectedSeq,
  onSelect,
  onLoadMore,
  disabled = false,
}: TraceTimelineProps) {
  if (events.length === 0) {
    return (
      <div
        data-testid="event-list"
        className="rounded-lg border border-slate-800 bg-slate-900 p-4 text-sm text-slate-200"
      >
        No events available.
      </div>
    );
  }

  const itemData: RowData = {
    events,
    selectedSeq,
    onSelect,
    disabled,
  };

  return (
    <div data-testid="event-list" className="rounded-lg border border-slate-800 bg-slate-950">
      <div className="border-b border-slate-800 px-3 py-2 text-xs uppercase tracking-wide text-slate-200">
        Timeline ({events.length}/{total})
      </div>
      <List
        height={520}
        width="100%"
        itemCount={events.length}
        itemSize={rowHeight}
        itemData={itemData}
        onItemsRendered={({ visibleStopIndex }) => {
          if (disabled || loadingMore || !hasMore) {
            return;
          }
          if (visibleStopIndex >= events.length - nearEndThreshold) {
            onLoadMore();
          }
        }}
      >
        {Row}
      </List>
      <div className="flex items-center justify-between border-t border-slate-800 px-3 py-2">
        <span className="text-xs text-slate-200">
          Loaded {events.length} of {total}
        </span>
        <button
          type="button"
          disabled={disabled || loadingMore || !hasMore}
          onClick={onLoadMore}
          className="rounded border border-slate-700 bg-slate-900 px-2 py-1 text-xs text-slate-100 hover:border-slate-600 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {loadingMore ? "Loading..." : hasMore ? "Load More" : "All Loaded"}
        </button>
      </div>
    </div>
  );
}
