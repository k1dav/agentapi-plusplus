package httpapi

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
)

const defaultRecentLogLimit = 200

type recentLogStore struct {
	mu    sync.RWMutex
	limit int
	lines []string
}

func newRecentLogStore(limit int) *recentLogStore {
	if limit <= 0 {
		limit = defaultRecentLogLimit
	}
	return &recentLogStore{
		limit: limit,
		lines: make([]string, 0, limit),
	}
}

func (s *recentLogStore) append(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lines = append(s.lines, line)
	if len(s.lines) > s.limit {
		copy(s.lines, s.lines[len(s.lines)-s.limit:])
		s.lines = s.lines[:s.limit]
	}
}

func (s *recentLogStore) snapshot() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lines := make([]string, len(s.lines))
	copy(lines, s.lines)
	return lines
}

type recentLogHandler struct {
	next   slog.Handler
	store  *recentLogStore
	attrs  []slog.Attr
	groups []string
}

func newRecentLogHandler(next slog.Handler, store *recentLogStore) slog.Handler {
	return &recentLogHandler{next: next, store: store}
}

func (h *recentLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *recentLogHandler) Handle(ctx context.Context, record slog.Record) error {
	if err := h.next.Handle(ctx, record); err != nil {
		return err
	}

	var buf bytes.Buffer
	var handler slog.Handler = slog.NewJSONHandler(&buf, nil)
	for _, group := range h.groups {
		handler = handler.WithGroup(group)
	}
	if len(h.attrs) > 0 {
		handler = handler.WithAttrs(h.attrs)
	}
	if err := handler.Handle(ctx, record); err != nil {
		return err
	}
	h.store.append(buf.String())
	return nil
}

func (h *recentLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nextAttrs := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	nextAttrs = append(nextAttrs, h.attrs...)
	nextAttrs = append(nextAttrs, attrs...)
	return &recentLogHandler{
		next:   h.next.WithAttrs(attrs),
		store:  h.store,
		attrs:  nextAttrs,
		groups: append([]string(nil), h.groups...),
	}
}

func (h *recentLogHandler) WithGroup(name string) slog.Handler {
	nextGroups := append(append([]string(nil), h.groups...), name)
	return &recentLogHandler{
		next:   h.next.WithGroup(name),
		store:  h.store,
		attrs:  append([]slog.Attr(nil), h.attrs...),
		groups: nextGroups,
	}
}
