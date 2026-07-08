package transcript

import (
	"bytes"
	_ "embed"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/claude_session.jsonl
var claudeFixture []byte

func parseFixture(t *testing.T, parser Parser, fixture []byte) []TimelineEvent {
	t.Helper()
	var events []TimelineEvent
	for _, line := range bytes.Split(fixture, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		evs, err := parser.ParseLine(line)
		require.NoError(t, err)
		events = append(events, evs...)
	}
	return events
}

func fixedNow() time.Time {
	return time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC)
}

func TestClaudeParser(t *testing.T) {
	t.Parallel()

	events := parseFixture(t, NewClaudeParser(fixedNow), claudeFixture)

	kinds := make([]Kind, 0, len(events))
	for _, ev := range events {
		kinds = append(kinds, ev.Kind)
	}
	require.Equal(t, []Kind{
		KindText,       // user prompt
		KindThinking,   // non-empty thinking (empty one skipped)
		KindToolCall,   // Bash
		KindToolResult, // string content
		KindToolResult, // array content with image
		KindText,       // assistant answer
		KindSystem,     // turn_duration
	}, kinds)

	userPrompt := events[0]
	assert.Equal(t, "user", userPrompt.Role)
	assert.Equal(t, "list the files", userPrompt.Content)
	assert.Equal(t, "sess-1", userPrompt.SessionId)
	assert.Equal(t, "u-1", userPrompt.SourceId)
	assert.Equal(t, 2026, userPrompt.Time.Year())

	thinking := events[1]
	assert.Equal(t, "assistant", thinking.Role)
	assert.Equal(t, "The user wants a file listing. I should run ls.", thinking.Content)

	toolCall := events[2]
	assert.Equal(t, "Bash", toolCall.ToolName)
	assert.Equal(t, "toolu_01", toolCall.ToolUseId)
	assert.JSONEq(t, `{"command":"ls","description":"List files"}`, string(toolCall.ToolInput))

	stringResult := events[3]
	assert.Equal(t, "toolu_01", stringResult.ToolUseId)
	assert.Equal(t, "README.md\nmain.go", stringResult.Content)

	arrayResult := events[4]
	assert.Equal(t, "toolu_02", arrayResult.ToolUseId)
	assert.Equal(t, "part one\n[image]\npart two", arrayResult.Content)

	answer := events[5]
	assert.Equal(t, "assistant", answer.Role)
	assert.Equal(t, "There are two files: README.md and main.go.", answer.Content)

	system := events[6]
	assert.Equal(t, "turn_duration", system.Content)
}

func TestClaudeParserFallbackTime(t *testing.T) {
	t.Parallel()

	parser := NewClaudeParser(fixedNow)
	evs, err := parser.ParseLine([]byte(`{"type":"user","message":{"role":"user","content":"hi"},"uuid":"u","sessionId":"s"}`))
	require.NoError(t, err)
	require.Len(t, evs, 1)
	assert.Equal(t, fixedNow(), evs[0].Time)
}
