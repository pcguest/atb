"use client";

import { eventFamilyClass } from "@/lib/event-family";
import type { EventRecord } from "@/lib/types";
import { eventDisplayLabel } from "@/lib/event-labels";
import type { DashboardRole } from "@/lib/roles";
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
  role: DashboardRole;
};

type RowData = {
  events: EventRecord[];
  selectedSeq: number | null;
  onSelect: (seq: number) => void;
  disabled: boolean;
  role: DashboardRole;
};

const rowHeight = 72;
const nearEndThreshold = 20;

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
        className={`flex h-full w-full items-center justify-between rounded border px-3 text-left transition-colors ${
          isSelected
            ? "border-primary/60 bg-primary/10"
            : "border-border bg-card hover:border-ring hover:bg-muted/50"
        } disabled:cursor-not-allowed disabled:opacity-60`}
      >
        <div className="min-w-0">
          <div className={`truncate font-mono text-sm font-medium ${eventFamilyClass(event.type)}`}>
            {eventDisplayLabel(event.type, data.role)}
          </div>
          <div className="truncate text-xs text-muted-foreground">
            {event.timestamp ?? "no timestamp"}
          </div>
        </div>
        <div className="ml-3 shrink-0 rounded bg-muted px-2 py-0.5 font-mono text-xs text-muted-foreground">
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
  role,
}: TraceTimelineProps) {
  if (events.length === 0) {
    return (
      <div
        data-testid="event-list"
        className="rounded border border-border bg-card p-4 font-mono text-xs text-muted-foreground"
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
    role,
  };

  return (
    <div data-testid="event-list" className="rounded border border-border bg-card">
      <div className="border-b border-border px-3 py-2 font-mono text-xs uppercase tracking-widest text-muted-foreground">
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
      <div className="flex items-center justify-between border-t border-border px-3 py-2">
        <span className="font-mono text-xs text-muted-foreground">
          {events.length} / {total}
        </span>
        <button
          type="button"
          disabled={disabled || loadingMore || !hasMore}
          onClick={onLoadMore}
          className="rounded border border-border bg-card px-2 py-1 text-xs text-muted-foreground hover:border-ring hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40"
        >
          {loadingMore ? "Loading..." : hasMore ? "Load More" : "All Loaded"}
        </button>
      </div>
    </div>
  );
}
