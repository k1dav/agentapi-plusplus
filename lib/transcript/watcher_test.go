package transcript

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mf "github.com/coder/agentapi/lib/msgfmt"
)

type eventCollector struct {
	mu     sync.Mutex
	events []TimelineEvent
}

func (c *eventCollector) handler(ev TimelineEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, ev)
}

func (c *eventCollector) snapshot() []TimelineEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]TimelineEvent(nil), c.events...)
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	require.Eventually(t, cond, 5*time.Second, 10*time.Millisecond)
}

func TestWatcherEndToEnd(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	collector := &eventCollector{}

	w, err := NewWatcher(Config{
		AgentType:    mf.AgentTypeClaude,
		WorkDir:      "/ignored",
		DirOverride:  dir,
		NotBefore:    time.Now().Add(-time.Minute),
		PollInterval: 10 * time.Millisecond,
		Handler:      collector.handler,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)

	// Session file appears after the watcher started.
	session1 := filepath.Join(dir, "session-1.jsonl")
	line := `{"type":"user","message":{"role":"user","content":"hello"},"uuid":"u-1","timestamp":"2026-07-08T06:00:00Z","sessionId":"s1"}` + "\n"
	require.NoError(t, os.WriteFile(session1, []byte(line), 0o644))

	waitFor(t, func() bool { return len(collector.snapshot()) >= 1 })
	first := collector.snapshot()[0]
	assert.Equal(t, KindText, first.Kind)
	assert.Equal(t, "hello", first.Content)

	// Appended lines are picked up incrementally.
	f, err := os.OpenFile(session1, os.O_WRONLY|os.O_APPEND, 0o644)
	require.NoError(t, err)
	_, err = f.WriteString(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi there"}]},"uuid":"a-1","timestamp":"2026-07-08T06:00:01Z","sessionId":"s1"}` + "\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	waitFor(t, func() bool { return len(collector.snapshot()) >= 2 })
	assert.Equal(t, "hi there", collector.snapshot()[1].Content)

	// A newer session file triggers a switch with a system boundary event.
	session2 := filepath.Join(dir, "session-2.jsonl")
	line2 := `{"type":"user","message":{"role":"user","content":"new session"},"uuid":"u-2","timestamp":"2026-07-08T06:10:00Z","sessionId":"s2"}` + "\n"
	require.NoError(t, os.WriteFile(session2, []byte(line2), 0o644))
	// Ensure the new file's mtime is strictly newer than session1's.
	future := time.Now().Add(2 * time.Second)
	require.NoError(t, os.Chtimes(session2, future, future))

	waitFor(t, func() bool {
		events := collector.snapshot()
		for _, ev := range events {
			if ev.SessionId == "s2" {
				return true
			}
		}
		return false
	})
	events := collector.snapshot()
	var sawSwitch bool
	for _, ev := range events {
		if ev.Kind == KindSystem && ev.Content == "session switched" {
			sawSwitch = true
		}
	}
	assert.True(t, sawSwitch, "expected a session-switched system event, got %+v", events)
}

func TestWatcherRequiresHandler(t *testing.T) {
	t.Parallel()

	_, err := NewWatcher(Config{AgentType: mf.AgentTypeClaude})
	assert.Error(t, err)
}

func TestWatcherUnsupportedAgent(t *testing.T) {
	t.Parallel()

	_, err := NewWatcher(Config{
		AgentType: mf.AgentTypeAider,
		WorkDir:   "/x",
		Handler:   func(TimelineEvent) {},
	})
	assert.ErrorIs(t, err, ErrUnsupportedAgent)
}
