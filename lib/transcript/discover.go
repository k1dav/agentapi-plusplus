package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	mf "github.com/coder/agentapi/lib/msgfmt"
	"golang.org/x/xerrors"
)

// Discoverer locates the newest candidate session file for an agent.
type Discoverer interface {
	// Newest returns the newest session file whose mtime is at or after
	// notBefore, or "" if none exists yet (which is normal before the
	// agent's first write).
	Newest(notBefore time.Time) (string, error)
}

// newDiscoverer builds the Discoverer for the given agent type. dirOverride,
// when set, bypasses auto-detection and globs *.jsonl in that directory.
func newDiscoverer(agentType mf.AgentType, workDir string, dirOverride string) (Discoverer, error) {
	if dirOverride != "" {
		return &dirDiscoverer{dir: dirOverride, pattern: "*.jsonl"}, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, xerrors.Errorf("resolve home dir: %w", err)
	}
	switch agentType {
	case mf.AgentTypeClaude:
		root := os.Getenv("CLAUDE_CONFIG_DIR")
		if root == "" {
			root = filepath.Join(home, ".claude")
		}
		return &dirDiscoverer{
			dir:     filepath.Join(root, "projects", sanitizeCwd(workDir)),
			pattern: "*.jsonl",
		}, nil
	case mf.AgentTypeCodex:
		root := os.Getenv("CODEX_HOME")
		if root == "" {
			root = filepath.Join(home, ".codex")
		}
		return &codexDiscoverer{root: root, workDir: workDir}, nil
	default:
		return nil, ErrUnsupportedAgent
	}
}

// sanitizeCwd mirrors Claude Code's project directory naming: every rune
// outside [A-Za-z0-9-] becomes '-' (verified: /workspaces/coder-templates
// -> -workspaces-coder-templates).
func sanitizeCwd(cwd string) string {
	out := []rune(cwd)
	for i, r := range out {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-':
		default:
			out[i] = '-'
		}
	}
	return string(out)
}

// dirDiscoverer picks the newest matching file in a single directory.
type dirDiscoverer struct {
	dir     string
	pattern string
}

func (d *dirDiscoverer) Newest(notBefore time.Time) (string, error) {
	return newestMatch([]string{d.dir}, d.pattern, notBefore, nil)
}

// codexDiscoverer scans the dated session directories for today and
// yesterday (both UTC and local, to survive midnight/timezone skew) and
// verifies file ownership via the session_meta cwd, since multiple Codex
// instances may share the machine.
type codexDiscoverer struct {
	root    string
	workDir string
}

func (d *codexDiscoverer) Newest(notBefore time.Time) (string, error) {
	now := time.Now()
	dirSet := map[string]struct{}{}
	for _, t := range []time.Time{now.UTC(), now.UTC().AddDate(0, 0, -1), now, now.AddDate(0, 0, -1)} {
		dirSet[filepath.Join(d.root, "sessions", t.Format("2006/01/02"))] = struct{}{}
	}
	dirs := make([]string, 0, len(dirSet))
	for dir := range dirSet {
		dirs = append(dirs, dir)
	}
	return newestMatch(dirs, "rollout-*.jsonl", notBefore, func(path string) bool {
		return codexSessionCwd(path) == d.workDir
	})
}

// codexSessionCwd reads the cwd from a rollout file's session_meta first
// line, or "" when it can't be determined.
func codexSessionCwd(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	head := make([]byte, 64*1024)
	n, _ := f.Read(head)
	head = head[:n]
	if i := indexByte(head, '\n'); i >= 0 {
		head = head[:i]
	}
	var line struct {
		Type    string `json:"type"`
		Payload struct {
			Cwd string `json:"cwd"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(head, &line); err != nil || line.Type != "session_meta" {
		return ""
	}
	return line.Payload.Cwd
}

func indexByte(b []byte, c byte) int {
	for i := range b {
		if b[i] == c {
			return i
		}
	}
	return -1
}

// newestMatch returns the newest file matching pattern across dirs with
// mtime >= notBefore, subject to an optional ownership check.
func newestMatch(dirs []string, pattern string, notBefore time.Time, owns func(string) bool) (string, error) {
	var best string
	var bestTime time.Time
	for _, dir := range dirs {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return "", xerrors.Errorf("glob %s: %w", dir, err)
		}
		for _, m := range matches {
			info, err := os.Stat(m)
			if err != nil || info.IsDir() {
				continue
			}
			mtime := info.ModTime()
			if mtime.Before(notBefore) || mtime.Before(bestTime) {
				continue
			}
			if owns != nil && !owns(m) {
				continue
			}
			best, bestTime = m, mtime
		}
	}
	return best, nil
}
