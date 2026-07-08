// Package mcpconfig reads and replaces the MCP server configuration of
// supported agents on disk. Agents load MCP config at process startup, so
// callers that need changes to take effect immediately must restart the
// agent process afterwards.
package mcpconfig

import (
	"encoding/json"

	mf "github.com/coder/agentapi/lib/msgfmt"
	"golang.org/x/xerrors"
)

var ErrUnsupportedAgent = xerrors.New("MCP configuration is not supported for this agent type")

// Servers maps an MCP server name to its agent-native configuration object
// (command/args/env for stdio servers, url/headers for remote ones). The
// content is passed through verbatim; validation is left to the agent.
type Servers map[string]json.RawMessage

// Store reads and fully replaces an agent's MCP server set.
type Store interface {
	// Path returns the config file being managed.
	Path() string
	// Read returns the currently configured MCP servers. A missing config
	// file yields an empty set.
	Read() (Servers, error)
	// Replace overwrites the MCP server set with exactly the given servers,
	// preserving all non-MCP content of the config file.
	Replace(servers Servers) error
}

// NewStore builds the Store for the given agent type. workDir is the
// directory the agent runs in (used for Claude's project-scope .mcp.json).
func NewStore(agentType mf.AgentType, workDir string) (Store, error) {
	switch agentType {
	case mf.AgentTypeClaude:
		return newClaudeStore(workDir)
	case mf.AgentTypeCodex:
		return newCodexStore()
	default:
		return nil, ErrUnsupportedAgent
	}
}

// SupportedAgent reports whether MCP config management is implemented for
// the given agent type.
func SupportedAgent(t mf.AgentType) bool {
	return t == mf.AgentTypeClaude || t == mf.AgentTypeCodex
}
