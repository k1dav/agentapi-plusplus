"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  Brain,
  Check,
  Copy,
  ExternalLink,
  Info,
  Link2,
  MessageSquare,
  Server,
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
import { useChat } from "./chat-provider";

type KindFilter =
  | "all"
  | "tool"
  | "thinking"
  | "text"
  | "system"
  | "links"
  | "mcp";

const FILTERS: { key: KindFilter; label: string }[] = [
  { key: "all", label: "All" },
  { key: "tool", label: "Tools" },
  { key: "thinking", label: "Thinking" },
  { key: "text", label: "Text" },
  { key: "system", label: "System" },
  { key: "links", label: "Links" },
  { key: "mcp", label: "MCP" },
];

interface ExtractedLink {
  url: string;
  source: string;
  id: number;
}

interface McpState {
  servers: Record<string, unknown>;
  path: string;
  supported: boolean;
}

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

// LinkRow renders one extracted URL with copy and open actions. The whole
// URL is shown unwrapped-safe (break-all) so it can also be selected by hand.
function LinkRow({ link }: { link: ExtractedLink }) {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(link.url);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch (e) {
      console.error("Failed to copy link:", e);
    }
  };

  return (
    <div className="flex items-start gap-2 px-3 py-2 border-b border-border/60 hover:bg-muted/50 transition-colors">
      <div className="flex-1 min-w-0">
        <div className="text-xs font-mono break-all">{link.url}</div>
        <div className="mt-0.5 text-[10px] text-muted-foreground">
          {link.source === "timeline" ? "exact" : "de-wrapped from screen"}
        </div>
      </div>
      <Button
        variant="ghost"
        size="icon"
        className="size-6 shrink-0"
        onClick={copy}
        aria-label="Copy link"
        title="Copy"
      >
        {copied ? (
          <Check className="size-3.5 text-emerald-600 dark:text-emerald-500" />
        ) : (
          <Copy className="size-3.5" />
        )}
      </Button>
      <Button
        variant="ghost"
        size="icon"
        className="size-6 shrink-0"
        asChild
        aria-label="Open link"
        title="Open in new tab"
      >
        <a href={link.url.startsWith("www.") ? `https://${link.url}` : link.url} target="_blank" rel="noopener noreferrer">
          <ExternalLink className="size-3.5" />
        </a>
      </Button>
    </div>
  );
}

// mcpServerSummary picks the most descriptive field of an MCP server config
// for the collapsed row.
function mcpServerSummary(config: unknown): string {
  if (config === null || typeof config !== "object") return "";
  const obj = config as Record<string, unknown>;
  if (typeof obj.url === "string") return obj.url;
  if (typeof obj.command === "string") {
    const args = Array.isArray(obj.args) ? obj.args.join(" ") : "";
    return `${obj.command} ${args}`.trim();
  }
  return "";
}

// McpServerRow renders one configured MCP server; expanding shows the full
// JSON config.
function McpServerRow({ name, config }: { name: string; config: unknown }) {
  const [expanded, setExpanded] = useState(false);
  return (
    <button
      type="button"
      onClick={() => setExpanded((v) => !v)}
      className="w-full text-left px-3 py-2 border-b border-border/60 hover:bg-muted/50 transition-colors"
    >
      <div className="flex items-center gap-2 text-xs">
        <Server className="size-3.5 shrink-0 text-muted-foreground" />
        <span className="font-medium shrink-0">{name}</span>
        <span className="font-mono text-muted-foreground truncate min-w-0">
          {mcpServerSummary(config)}
        </span>
      </div>
      {expanded && (
        <pre className="mt-1.5 text-xs font-mono whitespace-pre-wrap break-words bg-muted/60 rounded p-2 max-h-48 overflow-y-auto">
          {JSON.stringify(config, null, 2)}
        </pre>
      )}
    </button>
  );
}

// TimelinePanel is a side panel listing every structured timeline event with
// kind filters, plus a Links view backed by GET /links (copy-safe URLs even
// when the terminal wrapped them) and an MCP view backed by GET /mcp. It
// renders as a static right column on desktop and as an overlay drawer on
// small screens.
export function TimelinePanel({
  timeline,
  onClose,
}: {
  timeline: TimelineEvent[];
  onClose: () => void;
}) {
  const [filter, setFilter] = useState<KindFilter>("all");
  const [links, setLinks] = useState<ExtractedLink[]>([]);
  const [mcp, setMcp] = useState<McpState | null>(null);
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const { agentAPIUrl } = useChat();
  const filtered =
    filter === "links" || filter === "mcp"
      ? []
      : timeline.filter((e) => matchesFilter(e.kind, filter));

  const fetchLinks = useCallback(async () => {
    try {
      const response = await fetch(`${agentAPIUrl}/links`);
      if (!response.ok) return;
      const data = (await response.json()) as { links: ExtractedLink[] };
      setLinks(data.links ?? []);
    } catch (e) {
      console.error("Failed to fetch links:", e);
    }
  }, [agentAPIUrl]);

  const fetchMcp = useCallback(async () => {
    try {
      const response = await fetch(`${agentAPIUrl}/mcp`);
      if (response.status === 404) {
        setMcp({ servers: {}, path: "", supported: false });
        return;
      }
      if (!response.ok) return;
      const data = (await response.json()) as {
        servers: Record<string, unknown>;
        path: string;
      };
      setMcp({ servers: data.servers ?? {}, path: data.path, supported: true });
    } catch (e) {
      console.error("Failed to fetch MCP config:", e);
    }
  }, [agentAPIUrl]);

  // Refresh the links whenever the Links view is active and new timeline
  // events arrive (a cheap signal that new content may contain URLs).
  useEffect(() => {
    if (filter === "links") {
      fetchLinks();
    }
  }, [filter, timeline.length, fetchLinks]);

  useEffect(() => {
    if (filter === "mcp") {
      fetchMcp();
    }
  }, [filter, fetchMcp]);

  const mcpServerNames = mcp ? Object.keys(mcp.servers).sort() : [];

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
        <span className="text-sm font-medium">
          {filter === "links" ? "Links" : filter === "mcp" ? "MCP servers" : "Timeline"}
        </span>
        <span className="text-xs text-muted-foreground">
          {filter === "links"
            ? links.length
            : filter === "mcp"
              ? mcpServerNames.length
              : filtered.length}
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
        {filter === "mcp" ? (
          !mcp ? (
            <div className="p-4 text-xs text-muted-foreground">Loading…</div>
          ) : !mcp.supported ? (
            <div className="p-4 text-xs text-muted-foreground">
              MCP config management is not supported for this agent type.
            </div>
          ) : (
            <>
              {mcpServerNames.length === 0 ? (
                <div className="p-4 text-xs text-muted-foreground">
                  No MCP servers configured.
                </div>
              ) : (
                mcpServerNames.map((name) => (
                  <McpServerRow
                    key={name}
                    name={name}
                    config={mcp.servers[name]}
                  />
                ))
              )}
              {mcp.path && (
                <div className="px-3 py-2 text-[10px] text-muted-foreground font-mono break-all">
                  {mcp.path}
                </div>
              )}
            </>
          )
        ) : filter === "links" ? (
          links.length === 0 ? (
            <div className="p-4 text-xs text-muted-foreground flex items-center gap-1.5">
              <Link2 className="size-3.5" /> No links found yet.
            </div>
          ) : (
            links.map((link) => <LinkRow key={link.url} link={link} />)
          )
        ) : filtered.length === 0 ? (
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
