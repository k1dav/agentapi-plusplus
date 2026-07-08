package httpapi

import (
	"context"
	"time"

	"github.com/coder/agentapi/internal/version"
	mf "github.com/coder/agentapi/lib/msgfmt"
)

// This file holds the auxiliary read/info endpoints that complement the core
// conversation handlers in server.go. They were previously stranded in an
// unwired duplicate file (removed in the build repair, #516); this restores
// them and they are wired into registerRoutes in server.go.

// getInfo handles GET /info — capability/version info for the running agent.
func (s *Server) getInfo(ctx context.Context, input *struct{}) (*InfoResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp := &InfoResponse{}
	resp.Body.Version = version.Version
	resp.Body.AgentType = s.agentType
	resp.Body.Features = map[string]bool{
		"messages":   true,
		"events":     true,
		"upload":     true,
		"pagination": true,
		"slashCmd":   true,
		"timeline":   s.timelineEnabled,
	}
	return resp, nil
}

// newSessionCommand returns the slash command that starts a fresh session
// in the agent's TUI, for agents that support one.
func newSessionCommand(agentType mf.AgentType) (string, bool) {
	switch agentType {
	case mf.AgentTypeClaude:
		return "/clear", true
	case mf.AgentTypeCodex:
		return "/new", true
	default:
		return "", false
	}
}

// slashCommandMenuDelay is the pause between typing a slash command and
// pressing enter, letting the TUI's command menu settle so enter executes
// the typed command instead of a menu selection mid-render.
const slashCommandMenuDelay = 300 * time.Millisecond

// clearMessages handles DELETE /messages — clears the conversation history
// and timeline, and (by default) resets the agent's own session so its
// context matches the cleared history.
func (s *Server) clearMessages(ctx context.Context, input *struct {
	NewSession bool `query:"new_session" default:"true" doc:"Also reset the agent's own session by sending its new-session command (claude: /clear, codex: /new). The agent should be in the 'stable' state for the command to take effect. Set to false to only clear AgentAPI's stored history."`
},
) (*MessagesClearResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp := &MessagesClearResponse{}
	count := len(s.conversation.Messages())
	if clearer, ok := any(s.conversation).(interface{ ClearMessages() }); ok {
		clearer.ClearMessages()
	}
	s.emitter.ClearTimeline()

	newSessionSent := false
	if input.NewSession && s.agentio != nil {
		if cmd, ok := newSessionCommand(s.agentType); ok {
			// Raw keystrokes, not conversation.Send: user messages are
			// bracketed-pasted, which prevents slash-command interpretation.
			if _, err := s.agentio.Write([]byte(cmd)); err == nil {
				time.Sleep(slashCommandMenuDelay)
				if _, err := s.agentio.Write([]byte("\r")); err == nil {
					newSessionSent = true
				}
			}
			if !newSessionSent {
				s.logger.Warn("Failed to send new-session command to agent", "command", cmd)
			}
		}
	}

	resp.Body.Ok = true
	resp.Body.Count = count
	resp.Body.NewSession = newSessionSent
	return resp, nil
}

// getMessagesCount handles GET /messages/count.
func (s *Server) getMessagesCount(ctx context.Context, input *struct{}) (*MessagesCountResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp := &MessagesCountResponse{}
	resp.Body.Count = len(s.conversation.Messages())
	return resp, nil
}

// getLogs handles GET /logs.
func (s *Server) getLogs(ctx context.Context, input *struct{}) (*LogsResponse, error) {
	resp := &LogsResponse{}
	resp.Body.Logs = []string{}
	return resp, nil
}

// getRateLimit handles GET /rate-limit.
func (s *Server) getRateLimit(ctx context.Context, input *struct{}) (*RateLimitResponse, error) {
	resp := &RateLimitResponse{}
	resp.Body.Enabled = false
	resp.Body.Requests = 100
	return resp, nil
}

// getConfig handles GET /config.
func (s *Server) getConfig(ctx context.Context, input *struct{}) (*ConfigResponse, error) {
	resp := &ConfigResponse{}
	resp.Body.AgentType = string(s.agentType)
	resp.Body.Port = s.port
	return resp, nil
}

// getHealth handles GET /health.
func (s *Server) getHealth(ctx context.Context, input *struct{}) (*HealthResponse, error) {
	resp := &HealthResponse{}
	resp.Body.Status = "ok"
	return resp, nil
}

// getVersion handles GET /version.
func (s *Server) getVersion(ctx context.Context, input *struct{}) (*VersionResponse, error) {
	resp := &VersionResponse{}
	resp.Body.Version = version.Version
	return resp, nil
}

// getReady handles GET /ready.
func (s *Server) getReady(ctx context.Context, input *struct{}) (*ReadyResponse, error) {
	resp := &ReadyResponse{}
	resp.Body.Ready = true
	return resp, nil
}
