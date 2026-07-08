package transcript

import (
	"encoding/json"
	"strings"
	"time"
)

// codexParser parses Codex rollout lines
// (~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<uuid>.jsonl).
type codexParser struct {
	now func() time.Time
	// sessionId is remembered from the session_meta line and stamped on
	// subsequent events (rollout lines don't repeat it).
	sessionId string
}

func NewCodexParser(now func() time.Time) Parser {
	return &codexParser{now: now}
}

type codexLine struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type codexPayload struct {
	Type string `json:"type"`
	Id   string `json:"id"`
	// session_meta
	SessionId  string `json:"session_id"`
	Cwd        string `json:"cwd"`
	CliVersion string `json:"cli_version"`
	// reasoning
	Summary []codexTextBlock `json:"summary"`
	Content json.RawMessage  `json:"content"`
	// function_call / custom_tool_call
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Input     string `json:"input"`
	CallId    string `json:"call_id"`
	// function_call_output / custom_tool_call_output
	Output string `json:"output"`
	// local_shell_call
	Action json.RawMessage `json:"action"`
	// event_msg agent_message / user_message
	Message string `json:"message"`
}

type codexTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (p *codexParser) ParseLine(line []byte) ([]TimelineEvent, error) {
	var l codexLine
	if err := json.Unmarshal(line, &l); err != nil {
		return nil, nil
	}
	var payload codexPayload
	if len(l.Payload) > 0 {
		if err := json.Unmarshal(l.Payload, &payload); err != nil {
			return nil, nil
		}
	}

	base := TimelineEvent{
		Time:      p.lineTime(l.Timestamp),
		SessionId: p.sessionId,
		SourceId:  payload.Id,
	}

	switch l.Type {
	case "session_meta":
		p.sessionId = payload.SessionId
		if p.sessionId == "" {
			p.sessionId = payload.Id
		}
		base.SessionId = p.sessionId
		base.Kind = KindSystem
		base.Role = "system"
		base.Content = strings.TrimSpace("session started: " + payload.Cwd + " (codex " + payload.CliVersion + ")")
		return []TimelineEvent{base}, nil
	case "response_item":
		return p.parseResponseItem(payload, base)
	case "event_msg":
		return p.parseEventMsg(payload, base)
	default:
		return nil, nil
	}
}

func (p *codexParser) parseResponseItem(payload codexPayload, base TimelineEvent) ([]TimelineEvent, error) {
	switch payload.Type {
	case "reasoning":
		// The reasoning body is encrypted; plaintext lives in content[]
		// (reasoning_text) or summary[] (summary_text) depending on version.
		text := joinCodexTextBlocks(payload.Content)
		if text == "" {
			text = joinTextBlocks(payload.Summary)
		}
		if text == "" {
			return nil, nil
		}
		base.Kind = KindThinking
		base.Role = "assistant"
		base.Content = text
		return []TimelineEvent{base}, nil
	case "function_call":
		base.Kind = KindToolCall
		base.Role = "assistant"
		base.ToolName = payload.Name
		base.ToolUseId = payload.CallId
		base.ToolInput = codexArgsToJSON(payload.Arguments)
		return []TimelineEvent{base}, nil
	case "custom_tool_call":
		base.Kind = KindToolCall
		base.Role = "assistant"
		base.ToolName = payload.Name
		base.ToolUseId = payload.CallId
		base.ToolInput = codexArgsToJSON(payload.Input)
		return []TimelineEvent{base}, nil
	case "local_shell_call":
		base.Kind = KindToolCall
		base.Role = "assistant"
		base.ToolName = "shell"
		base.ToolUseId = payload.CallId
		base.ToolInput = payload.Action
		return []TimelineEvent{base}, nil
	case "function_call_output", "custom_tool_call_output":
		base.Kind = KindToolResult
		base.ToolUseId = payload.CallId
		base.Content = payload.Output
		return []TimelineEvent{base}, nil
	default:
		// "message" items are skipped on purpose: developer/user roles carry
		// huge instruction payloads; visible text arrives via event_msg.
		return nil, nil
	}
}

func (p *codexParser) parseEventMsg(payload codexPayload, base TimelineEvent) ([]TimelineEvent, error) {
	switch payload.Type {
	case "agent_message":
		if payload.Message == "" {
			return nil, nil
		}
		base.Kind = KindText
		base.Role = "assistant"
		base.Content = payload.Message
		return []TimelineEvent{base}, nil
	case "user_message":
		if payload.Message == "" {
			return nil, nil
		}
		base.Kind = KindText
		base.Role = "user"
		base.Content = payload.Message
		return []TimelineEvent{base}, nil
	default:
		return nil, nil
	}
}

// codexArgsToJSON stores tool arguments as raw JSON when they parse
// (function_call arguments are a JSON-encoded string), otherwise as a
// JSON-quoted string (custom_tool_call input is free-form, e.g. a patch).
func codexArgsToJSON(args string) json.RawMessage {
	if args == "" {
		return nil
	}
	if json.Valid([]byte(args)) {
		return json.RawMessage(args)
	}
	quoted, err := json.Marshal(args)
	if err != nil {
		return nil
	}
	return quoted
}

func joinTextBlocks(blocks []codexTextBlock) string {
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if strings.TrimSpace(b.Text) != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func joinCodexTextBlocks(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var blocks []codexTextBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	return joinTextBlocks(blocks)
}

func (p *codexParser) lineTime(ts string) time.Time {
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t
	}
	return p.now()
}
