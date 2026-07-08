package mcpconfig

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/xerrors"
)

// claudeStore manages the project-scope .mcp.json in the agent's working
// directory: {"mcpServers": {name: {...}}}. Other top-level keys are
// preserved on replace.
type claudeStore struct {
	path string
}

func newClaudeStore(workDir string) (*claudeStore, error) {
	if workDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, xerrors.Errorf("resolve working dir: %w", err)
		}
		workDir = wd
	}
	return &claudeStore{path: filepath.Join(workDir, ".mcp.json")}, nil
}

func (s *claudeStore) Path() string {
	return s.path
}

func (s *claudeStore) readFile() (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]json.RawMessage{}, nil
		}
		return nil, xerrors.Errorf("read %s: %w", s.path, err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, xerrors.Errorf("parse %s: %w", s.path, err)
	}
	return doc, nil
}

func (s *claudeStore) Read() (Servers, error) {
	doc, err := s.readFile()
	if err != nil {
		return nil, err
	}
	servers := Servers{}
	if raw, ok := doc["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &servers); err != nil {
			return nil, xerrors.Errorf("parse mcpServers in %s: %w", s.path, err)
		}
	}
	return servers, nil
}

func (s *claudeStore) Replace(servers Servers) error {
	doc, err := s.readFile()
	if err != nil {
		return err
	}
	if servers == nil {
		servers = Servers{}
	}
	serversRaw, err := json.Marshal(servers)
	if err != nil {
		return xerrors.Errorf("encode mcpServers: %w", err)
	}
	doc["mcpServers"] = serversRaw
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return xerrors.Errorf("encode %s: %w", s.path, err)
	}
	if err := os.WriteFile(s.path, append(data, '\n'), 0o644); err != nil {
		return xerrors.Errorf("write %s: %w", s.path, err)
	}
	return nil
}
