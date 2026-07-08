package transcript

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/codex_rollout.jsonl
var codexFixture []byte

func TestCodexParser(t *testing.T) {
	t.Parallel()

	events := parseFixture(t, NewCodexParser(fixedNow), codexFixture)

	kinds := make([]Kind, 0, len(events))
	for _, ev := range events {
		kinds = append(kinds, ev.Kind)
	}
	require.Equal(t, []Kind{
		KindSystem,     // session_meta
		KindText,       // user_message
		KindText,       // agent_message
		KindToolCall,   // function_call
		KindToolResult, // function_call_output
		KindThinking,   // reasoning with summary (encrypted-only skipped)
		KindToolCall,   // custom_tool_call
		KindToolResult, // custom_tool_call_output
	}, kinds)

	meta := events[0]
	assert.Equal(t, "codex-sess-1", meta.SessionId)
	assert.Contains(t, meta.Content, "/workspaces/demo")
	assert.Contains(t, meta.Content, "0.142.5")

	userMsg := events[1]
	assert.Equal(t, "user", userMsg.Role)
	assert.Equal(t, "please list the files", userMsg.Content)
	// session id remembered from session_meta is stamped on later events
	assert.Equal(t, "codex-sess-1", userMsg.SessionId)

	agentMsg := events[2]
	assert.Equal(t, "assistant", agentMsg.Role)
	assert.Equal(t, "I'll list the files first.", agentMsg.Content)

	toolCall := events[3]
	assert.Equal(t, "exec_command", toolCall.ToolName)
	assert.Equal(t, "call_01", toolCall.ToolUseId)
	assert.JSONEq(t, `{"cmd":"ls","workdir":"/workspaces/demo"}`, string(toolCall.ToolInput))

	toolResult := events[4]
	assert.Equal(t, "call_01", toolResult.ToolUseId)
	assert.Equal(t, "README.md\nmain.go\n", toolResult.Content)

	thinking := events[5]
	assert.Equal(t, "Considering the file list output.", thinking.Content)

	patchCall := events[6]
	assert.Equal(t, "apply_patch", patchCall.ToolName)
	assert.Equal(t, "call_02", patchCall.ToolUseId)
	// free-form input is stored as a JSON-quoted string
	assert.Contains(t, string(patchCall.ToolInput), "Begin Patch")

	patchResult := events[7]
	assert.Equal(t, "call_02", patchResult.ToolUseId)
	assert.Equal(t, "Exit code: 0", patchResult.Content)
}
