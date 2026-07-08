package httpapi

import (
	"context"
	"encoding/json"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/xerrors"

	"github.com/coder/agentapi/lib/mcpconfig"
)

// McpServers is the API-level MCP server map. Values are passed through to
// the agent's config file verbatim; validation is left to the agent.
type McpServers map[string]any

type McpGetResponse struct {
	Body struct {
		Servers McpServers `json:"servers" nullable:"false" doc:"Currently configured MCP servers, keyed by name"`
		Path    string     `json:"path" doc:"Config file the servers were read from"`
	}
}

type McpUpdateRequestBody struct {
	Servers McpServers `json:"servers" nullable:"false" doc:"Complete MCP server set to install. Existing MCP servers not listed here are removed; unrelated config content is preserved."`
}

type McpUpdateResponse struct {
	Body struct {
		Ok        bool   `json:"ok" doc:"Whether the config was written"`
		Path      string `json:"path" doc:"Config file that was written"`
		Restarted bool   `json:"restarted" doc:"Whether the agent process was restarted to apply the change"`
	}
}

func toConfigServers(servers McpServers) (mcpconfig.Servers, error) {
	out := mcpconfig.Servers{}
	for name, cfg := range servers {
		raw, err := json.Marshal(cfg)
		if err != nil {
			return nil, xerrors.Errorf("encode mcp server %q: %w", name, err)
		}
		out[name] = raw
	}
	return out, nil
}

func fromConfigServers(servers mcpconfig.Servers) (McpServers, error) {
	out := McpServers{}
	for name, raw := range servers {
		var cfg any
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, xerrors.Errorf("decode mcp server %q: %w", name, err)
		}
		out[name] = cfg
	}
	return out, nil
}

// getMcp handles GET /mcp.
func (s *Server) getMcp(ctx context.Context, input *struct{}) (*McpGetResponse, error) {
	if s.mcpStore == nil {
		return nil, huma.Error404NotFound("MCP config management is not supported for this agent type or transport")
	}
	servers, err := s.mcpStore.Read()
	if err != nil {
		return nil, xerrors.Errorf("failed to read MCP config: %w", err)
	}
	apiServers, err := fromConfigServers(servers)
	if err != nil {
		return nil, err
	}
	resp := &McpGetResponse{}
	resp.Body.Servers = apiServers
	resp.Body.Path = s.mcpStore.Path()
	return resp, nil
}

// updateMcp handles PUT /mcp.
func (s *Server) updateMcp(ctx context.Context, input *struct {
	Restart bool `query:"restart" doc:"Restart the agent process after writing the config so the change takes effect immediately. The conversation context is lost; AgentAPI itself keeps running. Without this the change applies on the agent's next session."`
	Body    McpUpdateRequestBody
},
) (*McpUpdateResponse, error) {
	if s.mcpStore == nil {
		return nil, huma.Error404NotFound("MCP config management is not supported for this agent type or transport")
	}
	if input.Restart && s.restartAgent == nil {
		return nil, huma.Error400BadRequest("agent restart is not supported in this server mode")
	}

	servers, err := toConfigServers(input.Body.Servers)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	if err := s.mcpStore.Replace(servers); err != nil {
		return nil, xerrors.Errorf("failed to write MCP config: %w", err)
	}

	resp := &McpUpdateResponse{}
	resp.Body.Ok = true
	resp.Body.Path = s.mcpStore.Path()

	if input.Restart {
		if err := s.restartAgent(ctx); err != nil {
			return nil, xerrors.Errorf("MCP config written to %s, but agent restart failed: %w", s.mcpStore.Path(), err)
		}
		resp.Body.Restarted = true
	}
	return resp, nil
}
