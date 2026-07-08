package transcript

import (
	"encoding/json"
	"strings"
	"time"
)

// claudeParser parses Claude Code session transcript lines
// (~/.claude/projects/<sanitized-cwd>/<session-uuid>.jsonl).
type claudeParser struct {
	// now returns the fallback timestamp for entries without one.
	now func() time.Time
}

func NewClaudeParser(now func() time.Time) Parser {
	return &claudeParser{now: now}
}

// claudeEntry is the envelope of one transcript line. Only conversation
// entry types (user/assistant/system) are decoded further; everything else
// (mode, permission-mode, file-history-snapshot, summary, ai-title,
// last-prompt, attachment, ...) is skipped.
type claudeEntry struct {
	Type        string          `json:"type"`
	Uuid        string          `json:"uuid"`
	Timestamp   string          `json:"timestamp"`
	SessionId   string          `json:"sessionId"`
	IsSidechain bool            `json:"isSidechain"`
	IsMeta      bool            `json:"isMeta"`
	Message     json.RawMessage `json:"message"`
	// system entries carry their text in top-level fields
	Content string `json:"content"`
	Subtype string `json:"subtype"`
}

type claudeMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type claudeContentBlock struct {
	Type      string          `json:"type"`
	Thinking  string          `json:"thinking"`
	Text      string          `json:"text"`
	Id        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseId string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
}

func (p *claudeParser) ParseLine(line []byte) ([]TimelineEvent, error) {
	var entry claudeEntry
	if err := json.Unmarshal(line, &entry); err != nil {
		return nil, nil
	}
	if entry.IsSidechain || entry.IsMeta {
		// Sidechain entries are subagent transcripts; interleaving them with
		// the main conversation would be confusing. Skipped by design.
		return nil, nil
	}

	base := TimelineEvent{
		Time:      p.entryTime(entry.Timestamp),
		SessionId: entry.SessionId,
		SourceId:  entry.Uuid,
	}

	switch entry.Type {
	case "assistant":
		return p.parseAssistant(entry, base)
	case "user":
		return p.parseUser(entry, base)
	case "system":
		content := entry.Content
		if content == "" {
			content = entry.Subtype
		}
		if content == "" || isClaudeCommandMeta(content) {
			return nil, nil
		}
		base.Kind = KindSystem
		base.Role = "system"
		base.Content = content
		return []TimelineEvent{base}, nil
	default:
		return nil, nil
	}
}

func (p *claudeParser) parseAssistant(entry claudeEntry, base TimelineEvent) ([]TimelineEvent, error) {
	var msg claudeMessage
	if err := json.Unmarshal(entry.Message, &msg); err != nil {
		return nil, nil
	}
	var blocks []claudeContentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return nil, nil
	}
	var events []TimelineEvent
	for _, b := range blocks {
		ev := base
		ev.Role = "assistant"
		switch b.Type {
		case "thinking":
			// Signature-only redacted blocks have an empty thinking string.
			if strings.TrimSpace(b.Thinking) == "" {
				continue
			}
			ev.Kind = KindThinking
			ev.Content = b.Thinking
		case "text":
			if b.Text == "" {
				continue
			}
			ev.Kind = KindText
			ev.Content = b.Text
		case "tool_use":
			ev.Kind = KindToolCall
			ev.ToolName = b.Name
			ev.ToolInput = b.Input
			ev.ToolUseId = b.Id
		default:
			continue
		}
		events = append(events, ev)
	}
	return events, nil
}

func (p *claudeParser) parseUser(entry claudeEntry, base TimelineEvent) ([]TimelineEvent, error) {
	var msg claudeMessage
	if err := json.Unmarshal(entry.Message, &msg); err != nil {
		return nil, nil
	}
	// A typed user prompt has a plain string content.
	var text string
	if err := json.Unmarshal(msg.Content, &text); err == nil {
		if text == "" || isClaudeCommandMeta(text) {
			return nil, nil
		}
		base.Kind = KindText
		base.Role = "user"
		base.Content = text
		return []TimelineEvent{base}, nil
	}
	// Otherwise it is an array of blocks (tool results, images, ...).
	var blocks []claudeContentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return nil, nil
	}
	var events []TimelineEvent
	for _, b := range blocks {
		switch b.Type {
		case "tool_result":
			ev := base
			ev.Kind = KindToolResult
			ev.Role = "user"
			ev.ToolUseId = b.ToolUseId
			ev.Content = flattenClaudeToolResult(b.Content)
			events = append(events, ev)
		case "text":
			if b.Text == "" || isClaudeCommandMeta(b.Text) {
				continue
			}
			ev := base
			ev.Kind = KindText
			ev.Role = "user"
			ev.Content = b.Text
			events = append(events, ev)
		}
	}
	return events, nil
}

// isClaudeCommandMeta reports whether a user-entry text is slash-command
// bookkeeping that Claude Code logs into the transcript (e.g. after /clear),
// rather than something a user actually said.
func isClaudeCommandMeta(text string) bool {
	trimmed := strings.TrimSpace(text)
	return strings.HasPrefix(trimmed, "<command-name>") ||
		strings.HasPrefix(trimmed, "<local-command-stdout>")
}

// flattenClaudeToolResult normalizes a tool_result content field, which is
// either a plain string or an array of content blocks.
func flattenClaudeToolResult(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []claudeContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		switch b.Type {
		case "text":
			parts = append(parts, b.Text)
		case "image":
			parts = append(parts, "[image]")
		}
	}
	return strings.Join(parts, "\n")
}

func (p *claudeParser) entryTime(ts string) time.Time {
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t
	}
	return p.now()
}
