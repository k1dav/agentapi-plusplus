"use client";

import { useEffect, useRef, useState } from "react";
import {
  Brain,
  Info,
  MessageSquare,
  Wrench,
  X,
} from "lucide-react";
import {
  TimelineEvent,
  TimelineKind,
  formatEventTime,
  formatToolInput,
  summarizeToolInput,
} from "@/lib/timeline";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

type KindFilter = "all" | "tool" | "thinking" | "text" | "system";

const FILTERS: { key: KindFilter; label: string }[] = [
  { key: "all", label: "All" },
  { key: "tool", label: "Tools" },
  { key: "thinking", label: "Thinking" },
  { key: "text", label: "Text" },
  { key: "system", label: "System" },
];

function matchesFilter(kind: TimelineKind, filter: KindFilter): boolean {
  if (filter === "all") return true;
  if (filter === "tool") return kind === "tool_call" || kind === "tool_result";
  return kind === filter;
}

const KIND_STYLE: Record<TimelineKind, { label: string; className: string }> = {
  thinking: {
    label: "thinking",
    className:
      "bg-violet-100 text-violet-700 dark:bg-violet-950 dark:text-violet-300",
  },
  text: {
    label: "text",
    className: "bg-sky-100 text-sky-700 dark:bg-sky-950 dark:text-sky-300",
  },
  tool_call: {
    label: "tool call",
    className:
      "bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-300",
  },
  tool_result: {
    label: "result",
    className:
      "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300",
  },
  system: {
    label: "system",
    className: "bg-muted text-muted-foreground",
  },
};

function kindIcon(kind: TimelineKind) {
  switch (kind) {
    case "thinking":
      return <Brain className="size-3.5" />;
    case "tool_call":
    case "tool_result":
      return <Wrench className="size-3.5" />;
    case "system":
      return <Info className="size-3.5" />;
    default:
      return <MessageSquare className="size-3.5" />;
  }
}

function eventPreview(event: TimelineEvent): string {
  if (event.kind === "tool_call") {
    const summary = summarizeToolInput(event.tool_input);
    return summary
      ? `${event.tool_name}: ${summary}`
      : event.tool_name || "tool call";
  }
  return event.content || "";
}

function TimelineRow({ event }: { event: TimelineEvent }) {
  const [expanded, setExpanded] = useState(false);
  const style = KIND_STYLE[event.kind];

  return (
    <button
      type="button"
      onClick={() => setExpanded((v) => !v)}
      className="w-full text-left px-3 py-2 border-b border-border/60 hover:bg-muted/50 transition-colors"
    >
      <div className="flex items-center gap-2 text-[11px]">
        <span
          className={cn(
            "inline-flex items-center gap-1 rounded px-1.5 py-0.5 font-medium shrink-0",
            style.className
          )}
        >
          {kindIcon(event.kind)}
          {style.label}
        </span>
        {event.role && (
          <span className="text-muted-foreground shrink-0">{event.role}</span>
        )}
        <span className="ml-auto text-muted-foreground shrink-0 tabular-nums">
          {formatEventTime(event.time)}
        </span>
      </div>
      <div
        className={cn(
          "mt-1 text-xs font-mono whitespace-pre-wrap break-words",
          !expanded && "line-clamp-2 text-muted-foreground"
        )}
      >
        {eventPreview(event)}
      </div>
      {expanded && event.kind === "tool_call" && event.tool_input != null && (
        <pre className="mt-1.5 text-xs font-mono whitespace-pre-wrap break-words bg-muted/60 rounded p-2 max-h-48 overflow-y-auto">
          {formatToolInput(event.tool_input)}
        </pre>
      )}
    </button>
  );
}

// TimelinePanel is a side panel listing every structured timeline event with
// kind filters. It renders as a static right column on desktop and as an
// overlay drawer on small screens.
export function TimelinePanel({
  timeline,
  onClose,
}: {
  timeline: TimelineEvent[];
  onClose: () => void;
}) {
  const [filter, setFilter] = useState<KindFilter>("all");
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const filtered = timeline.filter((e) => matchesFilter(e.kind, filter));

  // Keep the newest events in view as they stream in.
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const nearBottom =
      el.scrollTop + el.clientHeight >= el.scrollHeight - el.clientHeight / 2;
    if (nearBottom) {
      el.scrollTop = el.scrollHeight;
    }
  }, [filtered.length]);

  return (
    <aside className="absolute md:static inset-y-0 right-0 z-20 w-full max-w-sm md:w-96 md:max-w-none flex flex-col border-l border-border bg-background shadow-lg md:shadow-none">
      <div className="flex items-center gap-2 px-3 py-2 border-b border-border">
        <span className="text-sm font-medium">Timeline</span>
        <span className="text-xs text-muted-foreground">
          {filtered.length}
        </span>
        <Button
          variant="ghost"
          size="icon"
          className="ml-auto size-7"
          onClick={onClose}
          aria-label="Close timeline"
        >
          <X className="size-4" />
        </Button>
      </div>
      <div className="flex flex-wrap gap-1 px-3 py-2 border-b border-border">
        {FILTERS.map(({ key, label }) => (
          <button
            key={key}
            type="button"
            onClick={() => setFilter(key)}
            className={cn(
              "rounded-full px-2.5 py-0.5 text-[11px] border transition-colors",
              filter === key
                ? "bg-accent-foreground text-accent border-transparent"
                : "border-border text-muted-foreground hover:bg-muted"
            )}
          >
            {label}
          </button>
        ))}
      </div>
      <div ref={scrollRef} className="flex-1 overflow-y-auto">
        {filtered.length === 0 ? (
          <div className="p-4 text-xs text-muted-foreground">
            No timeline events yet.
          </div>
        ) : (
          filtered.map((event) => <TimelineRow key={event.id} event={event} />)
        )}
      </div>
    </aside>
  );
}
