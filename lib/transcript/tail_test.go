package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTailerIncremental(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "session.jsonl")
	tl := newTailer(path)

	// Missing file is not an error.
	lines, err := tl.Drain()
	require.NoError(t, err)
	assert.Empty(t, lines)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	require.NoError(t, err)
	defer f.Close()

	_, err = f.WriteString("line1\nline2\n")
	require.NoError(t, err)
	lines, err = tl.Drain()
	require.NoError(t, err)
	require.Equal(t, [][]byte{[]byte("line1"), []byte("line2")}, lines)

	// A torn line is held back until its newline arrives.
	_, err = f.WriteString("par")
	require.NoError(t, err)
	lines, err = tl.Drain()
	require.NoError(t, err)
	assert.Empty(t, lines)

	_, err = f.WriteString("tial\nline4\n")
	require.NoError(t, err)
	lines, err = tl.Drain()
	require.NoError(t, err)
	require.Equal(t, [][]byte{[]byte("partial"), []byte("line4")}, lines)
}

func TestTailerTruncation(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "session.jsonl")
	tl := newTailer(path)

	require.NoError(t, os.WriteFile(path, []byte("old1\nold2\n"), 0o644))
	lines, err := tl.Drain()
	require.NoError(t, err)
	require.Len(t, lines, 2)

	// File replaced with shorter content: offset resets to 0.
	require.NoError(t, os.WriteFile(path, []byte("new\n"), 0o644))
	lines, err = tl.Drain()
	require.NoError(t, err)
	require.Equal(t, [][]byte{[]byte("new")}, lines)
}

func TestTailerOversizedLine(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "session.jsonl")
	tl := newTailer(path)

	huge := strings.Repeat("x", maxLineSize+1)
	require.NoError(t, os.WriteFile(path, []byte(huge+"\nafter\n"), 0o644))
	lines, err := tl.Drain()
	require.NoError(t, err)
	require.Equal(t, [][]byte{[]byte("after")}, lines)
}
