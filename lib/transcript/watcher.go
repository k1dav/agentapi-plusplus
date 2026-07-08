package transcript

import (
	"context"
	"log/slog"
	"os"
	"time"

	mf "github.com/coder/agentapi/lib/msgfmt"
	"golang.org/x/xerrors"
)

var ErrUnsupportedAgent = xerrors.New("transcript tailing is not supported for this agent type")

const (
	defaultPollInterval = 300 * time.Millisecond
	// discoverEvery re-runs session-file discovery every Nth poll tick
	// (~2s at the default interval) to pick up new session files created
	// mid-run (Claude /clear, Codex new thread).
	discoverEvery = 7
)

type Config struct {
	AgentType mf.AgentType
	// WorkDir is the directory the agent runs in; defaults to os.Getwd().
	WorkDir string
	// DirOverride skips auto-detection and tails *.jsonl in this directory.
	DirOverride string
	// NotBefore filters out session files from previous runs by mtime.
	NotBefore    time.Time
	PollInterval time.Duration
	Logger       *slog.Logger
	Handler      Handler
}

// Watcher tails the agent's transcript files and pushes normalized events
// to the configured Handler from a single goroutine.
type Watcher struct {
	cfg        Config
	discoverer Discoverer
	parser     Parser
	tail       *tailer
}

func NewWatcher(cfg Config) (*Watcher, error) {
	if cfg.Handler == nil {
		return nil, xerrors.New("transcript: Handler is required")
	}
	if cfg.WorkDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, xerrors.Errorf("resolve working dir: %w", err)
		}
		cfg.WorkDir = wd
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultPollInterval
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	discoverer, err := newDiscoverer(cfg.AgentType, cfg.WorkDir, cfg.DirOverride)
	if err != nil {
		return nil, err
	}
	var parser Parser
	switch cfg.AgentType {
	case mf.AgentTypeClaude:
		parser = NewClaudeParser(time.Now)
	case mf.AgentTypeCodex:
		parser = NewCodexParser(time.Now)
	default:
		return nil, ErrUnsupportedAgent
	}
	return &Watcher{cfg: cfg, discoverer: discoverer, parser: parser}, nil
}

// Start launches the watch loop in a goroutine; it exits when ctx is done.
func (w *Watcher) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *Watcher) loop(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	w.discover()
	tick := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick++
			if w.tail == nil {
				if tick%discoverEvery == 0 {
					w.discover()
				}
				continue
			}
			w.drain()
			if tick%discoverEvery == 0 {
				w.discover()
			}
		}
	}
}

// discover checks for the newest session file and switches to it when it
// differs from the one currently tailed.
func (w *Watcher) discover() {
	newest, err := w.discoverer.Newest(w.cfg.NotBefore)
	if err != nil {
		w.cfg.Logger.Debug("transcript discovery failed", "error", err)
		return
	}
	if newest == "" || (w.tail != nil && newest == w.tail.path) {
		return
	}
	if w.tail != nil {
		// Flush what remains of the previous session before switching.
		w.drain()
		w.cfg.Logger.Info("transcript session switched", "path", newest)
		w.cfg.Handler(TimelineEvent{
			Kind:    KindSystem,
			Role:    "system",
			Time:    time.Now(),
			Content: "session switched",
		})
	} else {
		w.cfg.Logger.Info("transcript session found", "path", newest)
	}
	w.tail = newTailer(newest)
}

func (w *Watcher) drain() {
	lines, err := w.tail.Drain()
	if err != nil {
		w.cfg.Logger.Debug("transcript drain failed", "path", w.tail.path, "error", err)
	}
	for _, line := range lines {
		events, err := w.parser.ParseLine(line)
		if err != nil {
			w.cfg.Logger.Debug("transcript parse failed", "error", err)
			continue
		}
		for _, ev := range events {
			if ev.Time.IsZero() {
				ev.Time = time.Now()
			}
			w.cfg.Handler(ev)
		}
	}
}
