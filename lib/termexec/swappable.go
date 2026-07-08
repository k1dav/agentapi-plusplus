package termexec

import (
	"sync"
)

// SwappableProcess is an AgentIO facade whose underlying *Process can be
// replaced at runtime (agent restart). Holders — the conversation loop, the
// HTTP handlers — keep the facade; only the supervisor swaps the inner
// process.
type SwappableProcess struct {
	mu      sync.RWMutex
	current *Process
}

func NewSwappableProcess(p *Process) *SwappableProcess {
	return &SwappableProcess{current: p}
}

func (s *SwappableProcess) Current() *Process {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

func (s *SwappableProcess) Set(p *Process) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = p
}

func (s *SwappableProcess) Write(data []byte) (int, error) {
	return s.Current().Write(data)
}

func (s *SwappableProcess) ReadScreen() string {
	return s.Current().ReadScreen()
}
