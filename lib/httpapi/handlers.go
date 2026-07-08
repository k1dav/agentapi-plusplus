package httpapi

import (
	"context"

	"github.com/coder/agentapi/internal/version"
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

// clearMessages handles DELETE /messages — clears the conversation history.
func (s *Server) clearMessages(ctx context.Context, input *struct{}) (*MessagesClearResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp := &MessagesClearResponse{}
	count := len(s.conversation.Messages())
	if clearer, ok := any(s.conversation).(interface{ ClearMessages() }); ok {
		clearer.ClearMessages()
	}
	resp.Body.Ok = true
	resp.Body.Count = count
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
