package transcript

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mf "github.com/coder/agentapi/lib/msgfmt"
)

func TestSanitizeCwd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want string
	}{
		{"/workspaces/coder-templates", "-workspaces-coder-templates"},
		{"/home/user/my.project", "-home-user-my-project"},
		{"/a_b/c d", "-a-b-c-d"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, sanitizeCwd(tt.in))
	}
}

func TestClaudeDiscoverer(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "projects", "-workspaces-demo")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	old := filepath.Join(dir, "old.jsonl")
	fresh := filepath.Join(dir, "fresh.jsonl")
	require.NoError(t, os.WriteFile(old, []byte("{}\n"), 0o644))
	require.NoError(t, os.WriteFile(fresh, []byte("{}\n"), 0o644))
	past := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(old, past, past))

	t.Setenv("CLAUDE_CONFIG_DIR", root)
	d, err := newDiscoverer(mf.AgentTypeClaude, "/workspaces/demo", "")
	require.NoError(t, err)

	// mtime filter excludes the stale file from a previous run.
	got, err := d.Newest(time.Now().Add(-time.Minute))
	require.NoError(t, err)
	assert.Equal(t, fresh, got)

	// Nothing matches when everything predates notBefore.
	got, err = d.Newest(time.Now().Add(time.Minute))
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestClaudeDiscovererMissingDir(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(t.TempDir(), "nope"))
	d, err := newDiscoverer(mf.AgentTypeClaude, "/workspaces/demo", "")
	require.NoError(t, err)
	got, err := d.Newest(time.Time{})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestCodexDiscovererOwnership(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "sessions", time.Now().UTC().Format("2006/01/02"))
	require.NoError(t, os.MkdirAll(dir, 0o755))

	mine := filepath.Join(dir, "rollout-2026-07-08T10-00-00-aaa.jsonl")
	theirs := filepath.Join(dir, "rollout-2026-07-08T11-00-00-bbb.jsonl")
	require.NoError(t, os.WriteFile(mine,
		[]byte(`{"type":"session_meta","payload":{"id":"a","cwd":"/workspaces/demo"}}`+"\n"), 0o644))
	require.NoError(t, os.WriteFile(theirs,
		[]byte(`{"type":"session_meta","payload":{"id":"b","cwd":"/somewhere/else"}}`+"\n"), 0o644))

	t.Setenv("CODEX_HOME", root)
	d, err := newDiscoverer(mf.AgentTypeCodex, "/workspaces/demo", "")
	require.NoError(t, err)

	// The newer file belongs to another session's cwd and is skipped.
	got, err := d.Newest(time.Time{})
	require.NoError(t, err)
	assert.Equal(t, mine, got)
}

func TestDirOverride(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "whatever.jsonl")
	require.NoError(t, os.WriteFile(file, []byte("{}\n"), 0o644))

	d, err := newDiscoverer(mf.AgentTypeClaude, "/ignored", dir)
	require.NoError(t, err)
	got, err := d.Newest(time.Time{})
	require.NoError(t, err)
	assert.Equal(t, file, got)
}

func TestUnsupportedAgent(t *testing.T) {
	t.Parallel()

	_, err := newDiscoverer(mf.AgentTypeGoose, "/x", "")
	assert.ErrorIs(t, err, ErrUnsupportedAgent)
}
