package mcpconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"golang.org/x/xerrors"
)

// codexStore manages the [mcp_servers.*] tables in ~/.codex/config.toml
// (CODEX_HOME respected). Replace performs line-level surgery so comments
// and unrelated settings — e.g. coder-managed blocks — are preserved
// verbatim; only the mcp_servers sections are rewritten.
type codexStore struct {
	path string
}

func newCodexStore() (*codexStore, error) {
	root := os.Getenv("CODEX_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, xerrors.Errorf("resolve home dir: %w", err)
		}
		root = filepath.Join(home, ".codex")
	}
	return &codexStore{path: filepath.Join(root, "config.toml")}, nil
}

func (s *codexStore) Path() string {
	return s.path
}

func (s *codexStore) Read() (Servers, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Servers{}, nil
		}
		return nil, xerrors.Errorf("read %s: %w", s.path, err)
	}
	var doc struct {
		McpServers map[string]any `toml:"mcp_servers"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, xerrors.Errorf("parse %s: %w", s.path, err)
	}
	servers := Servers{}
	for name, cfg := range doc.McpServers {
		raw, err := json.Marshal(cfg)
		if err != nil {
			return nil, xerrors.Errorf("encode mcp server %q: %w", name, err)
		}
		servers[name] = raw
	}
	return servers, nil
}

// mcpHeaderRe matches [mcp_servers], [mcp_servers.name], and nested
// sub-table headers like [mcp_servers.name.env].
var mcpHeaderRe = regexp.MustCompile(`^\s*\[{1,2}\s*(?:"mcp_servers"|mcp_servers)\s*[\].]`)

// anyHeaderRe matches any TOML table header line.
var anyHeaderRe = regexp.MustCompile(`^\s*\[`)

// rootMcpAssignRe matches a root-level inline assignment `mcp_servers = {...}`.
var rootMcpAssignRe = regexp.MustCompile(`^\s*(?:"mcp_servers"|mcp_servers)\s*=`)

// managedMarkerRe matches the comment markers wrapping the block this store
// appends, so repeated Replace calls don't accumulate them.
var managedMarkerRe = regexp.MustCompile(`^\s*# (?:>>>|<<<) managed by agentapi`)

func (s *codexStore) Replace(servers Servers) error {
	data, err := os.ReadFile(s.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return xerrors.Errorf("read %s: %w", s.path, err)
	}

	// Strip existing mcp_servers sections, keep everything else verbatim.
	var kept []string
	inMcpSection := false
	inRootTable := true
	for _, line := range strings.Split(string(data), "\n") {
		if managedMarkerRe.MatchString(line) {
			continue
		}
		if anyHeaderRe.MatchString(line) {
			inRootTable = false
			inMcpSection = mcpHeaderRe.MatchString(line)
		} else if inRootTable && rootMcpAssignRe.MatchString(line) {
			continue
		}
		if inMcpSection {
			continue
		}
		kept = append(kept, line)
	}
	content := strings.TrimRight(strings.Join(kept, "\n"), "\n")

	encoded, err := encodeCodexServers(servers)
	if err != nil {
		return err
	}
	if encoded != "" {
		if content != "" {
			content += "\n"
		}
		content += "\n# >>> managed by agentapi (PUT /mcp) >>>\n" + encoded + "# <<< managed by agentapi <<<"
	}
	content += "\n"

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return xerrors.Errorf("create %s: %w", filepath.Dir(s.path), err)
	}
	if err := os.WriteFile(s.path, []byte(content), 0o644); err != nil {
		return xerrors.Errorf("write %s: %w", s.path, err)
	}
	return nil
}

func encodeCodexServers(servers Servers) (string, error) {
	if len(servers) == 0 {
		return "", nil
	}
	decoded := map[string]any{}
	for name, raw := range servers {
		var cfg any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		// Preserve integers (plain Unmarshal turns them into float64,
		// which the TOML encoder would render as 1.0).
		decoder.UseNumber()
		if err := decoder.Decode(&cfg); err != nil {
			return "", xerrors.Errorf("parse mcp server %q: %w", name, err)
		}
		decoded[name] = normalizeJSONNumbers(cfg)
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(map[string]any{"mcp_servers": decoded}); err != nil {
		return "", xerrors.Errorf("encode mcp_servers as TOML: %w", err)
	}
	out := buf.String()
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out, nil
}

func normalizeJSONNumbers(v any) any {
	switch val := v.(type) {
	case map[string]any:
		for k, item := range val {
			val[k] = normalizeJSONNumbers(item)
		}
		return val
	case []any:
		for i, item := range val {
			val[i] = normalizeJSONNumbers(item)
		}
		return val
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return i
		}
		if f, err := val.Float64(); err == nil {
			return f
		}
		return val.String()
	default:
		return v
	}
}
