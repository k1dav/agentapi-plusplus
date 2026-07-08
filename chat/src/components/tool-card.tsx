"use client";

import { useState } from "react";
import { CheckCircle2, ChevronRight, Loader2, Wrench } from "lucide-react";
import {
  ToolCallPair,
  formatToolInput,
  summarizeToolInput,
} from "@/lib/timeline";
import { cn } from "@/lib/utils";

// ToolCard renders one tool invocation as a collapsible card: the header
// shows the tool name, a one-line input preview, and a running/done state;
// expanding reveals the full input and the paired result output.
export function ToolCard({ pair }: { pair: ToolCallPair }) {
  const [expanded, setExpanded] = useState(false);
  const { call, result } = pair;
  const summary = summarizeToolInput(call.tool_input);

  return (
    <div className="rounded-md border border-border bg-muted/40 text-xs overflow-hidden">
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        className="w-full flex items-center gap-2 px-2.5 py-1.5 text-left hover:bg-muted/80 transition-colors"
      >
        <ChevronRight
          className={cn(
            "size-3.5 shrink-0 text-muted-foreground transition-transform",
            expanded && "rotate-90"
          )}
        />
        <Wrench className="size-3.5 shrink-0 text-muted-foreground" />
        <span className="font-medium shrink-0">{call.tool_name || "tool"}</span>
        {summary && (
          <span className="font-mono text-muted-foreground truncate min-w-0">
            {summary}
          </span>
        )}
        <span className="ml-auto shrink-0">
          {result ? (
            <CheckCircle2 className="size-3.5 text-emerald-600 dark:text-emerald-500" />
          ) : (
            <Loader2 className="size-3.5 animate-spin text-amber-600 dark:text-amber-500" />
          )}
        </span>
      </button>
      {expanded && (
        <div className="border-t border-border px-2.5 py-2 space-y-2">
          {call.tool_input !== undefined && call.tool_input !== null && (
            <div>
              <div className="text-muted-foreground mb-1">Input</div>
              <pre className="font-mono whitespace-pre-wrap break-words bg-background rounded p-2 max-h-48 overflow-y-auto">
                {formatToolInput(call.tool_input)}
              </pre>
            </div>
          )}
          <div>
            <div className="text-muted-foreground mb-1">
              {result ? "Result" : "Running…"}
            </div>
            {result && (
              <pre className="font-mono whitespace-pre-wrap break-words bg-background rounded p-2 max-h-64 overflow-y-auto">
                {result.content || "(empty)"}
              </pre>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// ToolCardGroup renders the tool invocations of one conversation turn.
export function ToolCardGroup({ pairs }: { pairs: ToolCallPair[] }) {
  if (pairs.length === 0) return null;
  return (
    <div className="flex flex-col gap-1.5 max-w-[80ch] mb-2">
      {pairs.map((pair) => (
        <ToolCard key={pair.call.id} pair={pair} />
      ))}
    </div>
  );
}
