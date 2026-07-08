package mcpconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mf "github.com/coder/agentapi/lib/msgfmt"
)

func TestClaudeStore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := NewStore(mf.AgentTypeClaude, dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, ".mcp.json"), store.Path())

	// Missing file reads as empty.
	servers, err := store.Read()
	require.NoError(t, err)
	assert.Empty(t, servers)

	// Replace creates the file.
	require.NoError(t, store.Replace(Servers{
		"github": json.RawMessage(`{"command":"npx","args":["-y","@modelcontextprotocol/server-github"],"env":{"GITHUB_TOKEN":"x"}}`),
	}))
	servers, err = store.Read()
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.JSONEq(t,
		`{"command":"npx","args":["-y","@modelcontextprotocol/server-github"],"env":{"GITHUB_TOKEN":"x"}}`,
		string(servers["github"]))

	// Replace preserves unrelated top-level keys.
	require.NoError(t, os.WriteFile(store.Path(),
		[]byte(`{"otherSetting": true, "mcpServers": {"old": {"command": "x"}}}`), 0o644))
	require.NoError(t, store.Replace(Servers{
		"remote": json.RawMessage(`{"type":"http","url":"https://mcp.example.com"}`),
	}))
	data, err := os.ReadFile(store.Path())
	require.NoError(t, err)
	var doc map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &doc))
	assert.Contains(t, doc, "otherSetting")
	servers, err = store.Read()
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Contains(t, servers, "remote")

	// Replace with empty set clears servers.
	require.NoError(t, store.Replace(Servers{}))
	servers, err = store.Read()
	require.NoError(t, err)
	assert.Empty(t, servers)
}

const codexFixture = `# >>> coder-managed: codex module >>>
sandbox_mode = "danger-full-access"
approval_policy = "never"

[projects."/workspaces/demo"]
trust_level = "trusted"

[mcp_servers.old]
command = "old-server"

[mcp_servers.old.env]
KEY = "value"

[tui.model_availability_nux]
"gpt-5.5" = 1
# <<< coder-managed: codex module <<<
`

func TestCodexStore(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	require.NoError(t, os.WriteFile(filepath.Join(root, "config.toml"), []byte(codexFixture), 0o644))

	store, err := NewStore(mf.AgentTypeCodex, "")
	require.NoError(t, err)

	servers, err := store.Read()
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.JSONEq(t, `{"command":"old-server","env":{"KEY":"value"}}`, string(servers["old"]))

	require.NoError(t, store.Replace(Servers{
		"github": json.RawMessage(`{"command":"npx","args":["-y","server-github"],"startup_timeout_sec":20}`),
	}))

	data, err := os.ReadFile(store.Path())
	require.NoError(t, err)
	content := string(data)
	// Non-MCP content survives verbatim, including comments.
	assert.Contains(t, content, "# >>> coder-managed: codex module >>>")
	assert.Contains(t, content, `sandbox_mode = "danger-full-access"`)
	assert.Contains(t, content, "[tui.model_availability_nux]")
	assert.Contains(t, content, `trust_level = "trusted"`)
	// Old MCP sections are gone (including nested sub-tables).
	assert.NotContains(t, content, "old-server")
	assert.NotContains(t, content, `KEY = "value"`)
	// Integers stay integers.
	assert.Contains(t, content, "startup_timeout_sec = 20")

	servers, err = store.Read()
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.JSONEq(t, `{"command":"npx","args":["-y","server-github"],"startup_timeout_sec":20}`, string(servers["github"]))

	// Round-trip again to ensure the managed block is replaced, not stacked.
	require.NoError(t, store.Replace(Servers{
		"other": json.RawMessage(`{"url":"https://mcp.example.com"}`),
	}))
	data, err = os.ReadFile(store.Path())
	require.NoError(t, err)
	assert.Equal(t, 1, len(regexp.MustCompile(`>>> managed by agentapi`).FindAllString(string(data), -1)))
	servers, err = store.Read()
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Contains(t, servers, "other")

	// Clearing removes the managed block entirely.
	require.NoError(t, store.Replace(Servers{}))
	data, err = os.ReadFile(store.Path())
	require.NoError(t, err)
	assert.NotContains(t, string(data), "managed by agentapi")
	assert.Contains(t, string(data), `sandbox_mode = "danger-full-access"`)
}

func TestCodexStoreMissingFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)

	store, err := NewStore(mf.AgentTypeCodex, "")
	require.NoError(t, err)
	servers, err := store.Read()
	require.NoError(t, err)
	assert.Empty(t, servers)

	require.NoError(t, store.Replace(Servers{
		"a": json.RawMessage(`{"command":"a"}`),
	}))
	servers, err = store.Read()
	require.NoError(t, err)
	assert.Len(t, servers, 1)
}

func TestUnsupportedAgent(t *testing.T) {
	t.Parallel()
	_, err := NewStore(mf.AgentTypeGoose, "")
	assert.ErrorIs(t, err, ErrUnsupportedAgent)
	assert.False(t, SupportedAgent(mf.AgentTypeGoose))
	assert.True(t, SupportedAgent(mf.AgentTypeClaude))
	assert.True(t, SupportedAgent(mf.AgentTypeCodex))
}
