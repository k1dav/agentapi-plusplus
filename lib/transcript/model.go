// Package transcript tails the session transcript files that AI agents
// (Claude Code, Codex) write to disk, and normalizes their structured
// entries — thinking, tool calls, tool results, text — into TimelineEvents.
// It is a read-only sidecar: it never writes to the agent's files and never
// touches the PTY.
package transcript

import (
	"encoding/json"
	"time"

	mf "github.com/coder/agentapi/lib/msgfmt"
)

type Kind string

const (
	KindThinking   Kind = "thinking"
	KindText       Kind = "text"
	KindToolCall   Kind = "tool_call"
	KindToolResult Kind = "tool_result"
	KindSystem     Kind = "system"
)

// TimelineEvent is the normalized cross-agent structured event.
// Id is assigned by the consumer (EventEmitter), not by parsers.
type TimelineEvent struct {
	Id        int             `json:"id"`
	Kind      Kind            `json:"kind"`
	Role      string          `json:"role,omitempty"`
	Time      time.Time       `json:"time"`
	SessionId string          `json:"session_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolName  string          `json:"tool_name,omitempty"`
	ToolInput json.RawMessage `json:"tool_input,omitempty"`
	// ToolUseId joins a tool_call with its tool_result
	// (Claude tool_use id / Codex call_id).
	ToolUseId string `json:"tool_use_id,omitempty"`
	// SourceId is the transcript-native identifier of the entry
	// (Claude entry uuid / Codex payload id), for dedupe and debugging.
	SourceId string `json:"source_id,omitempty"`
}

// Handler receives normalized events as they are read off the transcript.
type Handler func(TimelineEvent)

// Parser converts one JSONL line into zero or more normalized events.
// Irrelevant or unparseable lines return (nil, nil): tailing must never
// abort because of a single bad line.
type Parser interface {
	ParseLine(line []byte) ([]TimelineEvent, error)
}

// SupportedAgent reports whether transcript tailing is implemented for the
// given agent type.
func SupportedAgent(t mf.AgentType) bool {
	return t == mf.AgentTypeClaude || t == mf.AgentTypeCodex
}
