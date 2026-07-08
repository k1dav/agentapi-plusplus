// Types and helpers for the structured timeline captured from the agent's
// transcript files (SSE `timeline_event` / GET /timeline).

export type TimelineKind =
  | "thinking"
  | "text"
  | "tool_call"
  | "tool_result"
  | "system";

export interface TimelineEvent {
  id: number;
  kind: TimelineKind;
  role?: string;
  time: string;
  session_id?: string;
  content?: string;
  tool_name?: string;
  tool_input?: unknown;
  tool_use_id?: string;
  source_id?: string;
}

// A tool call joined with its result (matched by tool_use_id). The result is
// undefined while the tool is still running.
export interface ToolCallPair {
  call: TimelineEvent;
  result?: TimelineEvent;
}

// groupToolsByTurn splits the timeline into conversation turns (a turn starts
// at each user text event) and pairs tool calls with their results. Turn k
// holds the tools used while answering the k-th user message (1-based; turn 0
// is anything before the first user message).
export function groupToolsByTurn(timeline: TimelineEvent[]): ToolCallPair[][] {
  const turns: ToolCallPair[][] = [[]];
  const pairsByToolUseId = new Map<string, ToolCallPair>();

  for (const event of timeline) {
    if (event.kind === "text" && event.role === "user") {
      turns.push([]);
      continue;
    }
    if (event.kind === "tool_call") {
      const pair: ToolCallPair = { call: event };
      turns[turns.length - 1].push(pair);
      if (event.tool_use_id) {
        pairsByToolUseId.set(event.tool_use_id, pair);
      }
      continue;
    }
    if (event.kind === "tool_result" && event.tool_use_id) {
      const pair = pairsByToolUseId.get(event.tool_use_id);
      if (pair) {
        pair.result = event;
      }
    }
  }
  return turns;
}

// summarizeToolInput produces the one-line preview shown on a collapsed tool
// card, picking the most informative field of the input.
export function summarizeToolInput(input: unknown): string {
  if (input === null || input === undefined) return "";
  if (typeof input === "string") return input;
  if (typeof input !== "object") return String(input);

  const obj = input as Record<string, unknown>;
  for (const key of ["command", "cmd", "file_path", "path", "url", "pattern", "description"]) {
    const value = obj[key];
    if (typeof value === "string" && value.trim() !== "") {
      return value;
    }
  }
  const firstString = Object.values(obj).find(
    (v) => typeof v === "string" && v.trim() !== ""
  );
  if (typeof firstString === "string") return firstString;
  try {
    return JSON.stringify(input);
  } catch {
    return "";
  }
}

export function formatToolInput(input: unknown): string {
  if (input === null || input === undefined) return "";
  if (typeof input === "string") return input;
  try {
    return JSON.stringify(input, null, 2);
  } catch {
    return String(input);
  }
}

// timelineToJsonl serializes timeline events one per line, exactly as
// GET /timeline returned them — no fields added, dropped, or reordered.
export function timelineToJsonl(events: TimelineEvent[]): string {
  return events.map((event) => JSON.stringify(event)).join("\n") + "\n";
}

// timelineToMarkdown renders timeline events as a readable Markdown log: one
// section per event with kind, role, timestamp, and the content or tool input
// fenced as JSON. Used by the timeline panel's export action.
export function timelineToMarkdown(events: TimelineEvent[]): string {
  const sections = events.map((event) => {
    const meta = [event.role, event.time].filter(Boolean).join(" · ");
    const lines = [`## #${event.id} ${event.kind}${meta ? ` (${meta})` : ""}`];
    if (event.kind === "tool_call") {
      lines.push(`Tool: \`${event.tool_name || "unknown"}\``);
      if (event.tool_input != null) {
        lines.push(["```json", formatToolInput(event.tool_input), "```"].join("\n"));
      }
    }
    if (event.content) {
      lines.push(event.content);
    }
    return lines.join("\n\n");
  });
  return ["# Timeline", ...sections].join("\n\n") + "\n";
}

// downloadTextFile triggers a browser download of the given text content. The
// object URL is revoked after the click so repeated exports don't leak.
export function downloadTextFile(
  filename: string,
  content: string,
  mimeType: string
): void {
  const url = URL.createObjectURL(new Blob([content], { type: mimeType }));
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

export function timelineExportFilename(filter: string, ext: string): string {
  const stamp = new Date().toISOString().replace(/[:.]/g, "-").slice(0, 19);
  return `timeline-${filter}-${stamp}.${ext}`;
}

export function formatEventTime(time: string): string {
  const date = new Date(time);
  if (isNaN(date.getTime())) return "";
  return date.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}
