// Package mcp discovers Model Context Protocol servers from the config files of
// every known MCP client, and (in correlate.go) reconciles them against running
// processes. A server is "local" when launched as a stdio subprocess (has a
// command), and "remote" when reached over http/sse (has a url).
package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/paths"
	"gopkg.in/yaml.v3"
)

// Server is one discovered MCP server (declared in config and/or running).
type Server struct {
	UID, Username string
	Client        string // claude-desktop, cursor, vscode, process, ...
	Scope         string // user | project | global
	ConfigPath    string
	ServerName    string
	Transport     string // stdio | http | sse | streamable-http
	Location      string // local | remote
	Command       string
	Args          string // JSON array
	URL           string
	EnvKeys       string // JSON array of env var NAMES only (never values)
	Enabled       int    // -1 unknown, 0 disabled, 1 enabled
	Source        string // config | process | both
	Running       int
	PID           int
	ListeningPort int

	// Security posture (computed by enrichRisk).
	Capabilities string // inferred capability tags, comma-separated (fs-write, shell-exec, ...)
	RiskFlags    string // risk tokens, comma-separated (remote_fetch_exec, plaintext_secret, ...)
	SHA256       string // hash of the declaring config file (diffable identity)
	LaunchHash   string // hash of the launch spec (command+args+url) for rug-pull diffing
}

// ScanConfigs returns every MCP server declared in any client config under the
// given home directory.
func ScanConfigs(h homes.Home) []Server {
	r := paths.For(h.Dir)
	var out []Server

	files := userConfigFiles(r)
	files = append(files, projectConfigFiles(h.Dir)...)
	for _, cf := range files {
		for _, s := range parseFile(cf) {
			s.UID, s.Username = h.UID, h.Username
			if s.Client == "" {
				s.Client = cf.client
			}
			if s.Scope == "" {
				s.Scope = cf.scope
			}
			s.ConfigPath = cf.path
			out = append(out, s)
		}
	}
	out = append(out, scanContinueDir(h)...)
	for i := range out {
		out[i].enrichRisk()
	}
	return out
}

// ---- config file catalog ----

type cfgFile struct {
	client string
	path   string
	key    string // top-level key holding the server map
	scope  string
	format string // json | zed | continue | claudejson
}

func userConfigFiles(r paths.Roots) []cfgFile {
	h := r.Home
	var f []cfgFile
	add := func(client, path, key, format, scope string) {
		f = append(f, cfgFile{client: client, path: path, key: key, format: format, scope: scope})
	}

	// Claude Desktop
	switch runtime.GOOS {
	case "darwin":
		add("claude-desktop", filepath.Join(r.MacAppSupport, "Claude", "claude_desktop_config.json"), "mcpServers", "json", "user")
	case "windows":
		add("claude-desktop", filepath.Join(r.AppData, "Claude", "claude_desktop_config.json"), "mcpServers", "json", "user")
	default:
		add("claude-desktop", filepath.Join(r.XDGConfig, "claude-desktop", "claude_desktop_config.json"), "mcpServers", "json", "user")
	}

	// Claude Code
	add("claude-code", filepath.Join(h, ".claude.json"), "mcpServers", "claudejson", "user")
	add("claude-code", filepath.Join(h, ".claude", "settings.json"), "mcpServers", "json", "user")
	add("claude-code", filepath.Join(h, ".mcp.json"), "mcpServers", "json", "user")

	// Cursor (user)
	add("cursor", filepath.Join(h, ".cursor", "mcp.json"), "mcpServers", "json", "user")

	// Windsurf / Codeium
	add("windsurf", filepath.Join(h, ".codeium", "windsurf", "mcp_config.json"), "mcpServers", "json", "user")

	// VS Code native MCP (note: key is "servers", not "mcpServers")
	switch runtime.GOOS {
	case "darwin":
		add("vscode", filepath.Join(r.MacAppSupport, "Code", "User", "mcp.json"), "servers", "json", "user")
	case "windows":
		add("vscode", filepath.Join(r.AppData, "Code", "User", "mcp.json"), "servers", "json", "user")
	default:
		add("vscode", filepath.Join(r.XDGConfig, "Code", "User", "mcp.json"), "servers", "json", "user")
	}

	// Zed (context_servers; command is a nested object)
	add("zed", filepath.Join(r.XDGConfig, "zed", "settings.json"), "context_servers", "zed", "user")
	if runtime.GOOS == "darwin" {
		add("zed", filepath.Join(r.MacAppSupport, "Zed", "settings.json"), "context_servers", "zed", "user")
	}

	// Cline and Roo live under each VS Code-family editor's global storage.
	for _, base := range vscodeUserDirs(r) {
		add("cline", filepath.Join(base, "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"), "mcpServers", "json", "user")
		add("roo", filepath.Join(base, "globalStorage", "rooveterinaryinc.roo-cline", "settings", "mcp_settings.json"), "mcpServers", "json", "user")
	}

	// Continue (YAML preferred, JSON legacy) — both parsed via the continue path.
	add("continue", filepath.Join(h, ".continue", "config.yaml"), "mcpServers", "continue", "user")
	add("continue", filepath.Join(h, ".continue", "config.json"), "mcpServers", "continue", "user")

	return f
}

func vscodeUserDirs(r paths.Roots) []string {
	apps := []string{"Code", "Code - Insiders", "Cursor", "VSCodium", "Windsurf"}
	var base string
	switch runtime.GOOS {
	case "darwin":
		base = r.MacAppSupport
	case "windows":
		base = r.AppData
	default:
		base = r.XDGConfig
	}
	var dirs []string
	for _, a := range apps {
		dirs = append(dirs, filepath.Join(base, a, "User"))
	}
	return dirs
}

// projectConfigFiles does a bounded walk of common dev-project roots looking for
// repo-scoped MCP configs (.mcp.json, .cursor/mcp.json, .vscode/mcp.json,
// .roo/mcp.json). We cannot scan the whole disk, so coverage is best-effort.
func projectConfigFiles(home string) []cfgFile {
	roots := []string{
		home,
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Projects"),
		filepath.Join(home, "projects"),
		filepath.Join(home, "src"),
		filepath.Join(home, "code"),
		filepath.Join(home, "git"),
		filepath.Join(home, "dev"),
		filepath.Join(home, "workspace"),
		filepath.Join(home, "repos"),
	}
	seen := map[string]struct{}{}
	var out []cfgFile
	addIf := func(client, path, key, format string) {
		if _, ok := seen[path]; ok || !fsutil.Exists(path) {
			return
		}
		seen[path] = struct{}{}
		out = append(out, cfgFile{client: client, path: path, key: key, format: format, scope: "project"})
	}
	for _, root := range roots {
		fsutil.WalkBounded(root, 3, func(dir string) {
			addIf("claude-code", filepath.Join(dir, ".mcp.json"), "mcpServers", "json")
			addIf("cursor", filepath.Join(dir, ".cursor", "mcp.json"), "mcpServers", "json")
			addIf("vscode", filepath.Join(dir, ".vscode", "mcp.json"), "servers", "json")
			addIf("roo", filepath.Join(dir, ".roo", "mcp.json"), "mcpServers", "json")
		})
	}
	return out
}

// ---- parsing ----

func parseFile(cf cfgFile) []Server {
	switch cf.format {
	case "claudejson":
		return parseClaudeJSON(cf.path)
	case "zed":
		m, ok := extractJSONServers(cf.path, "context_servers")
		if !ok {
			return nil
		}
		return mapToServers(m)
	case "continue":
		return parseContinueFile(cf.path)
	default:
		m, ok := extractJSONServers(cf.path, cf.key)
		if !ok {
			return nil
		}
		return mapToServers(m)
	}
}

// jsonServer is the union of fields used across MCP clients. Command is a raw
// message because some clients (Zed) use a nested {path,args} object while most
// use a plain string.
type jsonServer struct {
	Command   json.RawMessage   `json:"command"`
	Args      []string          `json:"args"`
	URL       string            `json:"url"`
	ServerURL string            `json:"serverUrl"`
	Type      string            `json:"type"`
	Transport string            `json:"transport"`
	Env       map[string]string `json:"env"`
	Disabled  *bool             `json:"disabled"`
	Enabled   *bool             `json:"enabled"`
	Path      string            `json:"path"`
}

func (j jsonServer) commandAndArgs() (string, []string) {
	args := j.Args
	if len(j.Command) > 0 {
		var s string
		if err := json.Unmarshal(j.Command, &s); err == nil && s != "" {
			return s, args
		}
		var obj struct {
			Path    string   `json:"path"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
		}
		if err := json.Unmarshal(j.Command, &obj); err == nil {
			cmd := firstNonEmpty(obj.Path, obj.Command)
			if len(obj.Args) > 0 {
				args = obj.Args
			}
			return cmd, args
		}
	}
	if j.Path != "" {
		return j.Path, args
	}
	return "", args
}

func extractJSONServers(path, key string) (map[string]jsonServer, bool) {
	b, err := fsutil.ReadFileBounded(path)
	if err != nil {
		return nil, false
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		return nil, false
	}
	raw, ok := top[key]
	if !ok {
		return nil, false
	}
	var servers map[string]jsonServer
	if err := json.Unmarshal(raw, &servers); err != nil {
		return nil, false
	}
	return servers, true
}

func parseClaudeJSON(path string) []Server {
	b, err := fsutil.ReadFileBounded(path)
	if err != nil {
		return nil
	}
	var top struct {
		MCPServers map[string]jsonServer `json:"mcpServers"`
		Projects   map[string]struct {
			MCPServers map[string]jsonServer `json:"mcpServers"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(b, &top); err != nil {
		return nil
	}
	out := mapToServers(top.MCPServers)
	for _, p := range top.Projects {
		for _, s := range mapToServers(p.MCPServers) {
			s.Scope = "project"
			out = append(out, s)
		}
	}
	return out
}

func mapToServers(m map[string]jsonServer) []Server {
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]Server, 0, len(names))
	for _, n := range names {
		out = append(out, toServer(n, m[n]))
	}
	return out
}

func toServer(name string, j jsonServer) Server {
	cmd, args := j.commandAndArgs()
	url := firstNonEmpty(j.URL, j.ServerURL)
	s := Server{ServerName: name, Enabled: -1, Source: "config"}

	switch {
	case strings.EqualFold(j.Type, "stdio"):
		s.Location, s.Transport, s.Command = "local", "stdio", cmd
	case cmd != "":
		s.Location, s.Transport, s.Command = "local", "stdio", cmd
	case url != "":
		s.Location, s.Transport, s.URL = "remote", normalizeTransport(j.Type, j.Transport), url
	default:
		s.Location, s.Transport = "local", "stdio"
	}
	if len(args) > 0 && s.Command != "" {
		if b, err := json.Marshal(args); err == nil {
			s.Args = string(b)
		}
	}
	switch {
	case j.Enabled != nil:
		s.Enabled = boolToInt(*j.Enabled)
	case j.Disabled != nil:
		s.Enabled = boolToInt(!*j.Disabled)
	}
	s.EnvKeys = envKeyNames(j.Env)
	return s
}

func normalizeTransport(t, tr string) string {
	switch strings.ToLower(firstNonEmpty(t, tr)) {
	case "sse":
		return "sse"
	case "streamable-http", "streamablehttp", "streamable_http", "http-stream":
		return "streamable-http"
	case "http":
		return "http"
	case "stdio":
		return "stdio"
	case "":
		return "http"
	default:
		return strings.ToLower(firstNonEmpty(t, tr))
	}
}

// ---- Continue (YAML or JSON; map or list) ----

func parseContinueFile(path string) []Server {
	b, err := fsutil.ReadFileBounded(path)
	if err != nil {
		return nil
	}
	var doc struct {
		MCPServers yaml.Node `yaml:"mcpServers"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil { // YAML is a JSON superset, so .json parses too
		return nil
	}
	return continueNodeToServers(doc.MCPServers)
}

func scanContinueDir(h homes.Home) []Server {
	dir := filepath.Join(h.Dir, ".continue", "mcpServers")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Server
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}
		p := filepath.Join(dir, name)
		b, err := fsutil.ReadFileBounded(p)
		if err != nil {
			continue
		}
		var rs yamlServer
		if err := yaml.Unmarshal(b, &rs); err != nil {
			continue
		}
		srv := rs.toServer(firstNonEmpty(rs.Name, strings.TrimSuffix(name, filepath.Ext(name))))
		srv.UID, srv.Username = h.UID, h.Username
		srv.Client, srv.Scope, srv.ConfigPath = "continue", "user", p
		out = append(out, srv)
	}
	return out
}

type yamlServer struct {
	Name      string            `yaml:"name"`
	Command   string            `yaml:"command"`
	Args      []string          `yaml:"args"`
	URL       string            `yaml:"url"`
	ServerURL string            `yaml:"serverUrl"`
	Type      string            `yaml:"type"`
	Env       map[string]string `yaml:"env"`
}

func (rs yamlServer) toServer(name string) Server {
	url := firstNonEmpty(rs.URL, rs.ServerURL)
	s := Server{ServerName: name, Enabled: -1, Source: "config"}
	switch {
	case rs.Command != "":
		s.Location, s.Transport, s.Command = "local", "stdio", rs.Command
		if len(rs.Args) > 0 {
			if b, err := json.Marshal(rs.Args); err == nil {
				s.Args = string(b)
			}
		}
	case url != "":
		s.Location, s.Transport, s.URL = "remote", normalizeTransport(rs.Type, ""), url
	default:
		s.Location, s.Transport = "local", "stdio"
	}
	s.EnvKeys = envKeyNames(rs.Env)
	return s
}

func continueNodeToServers(n yaml.Node) []Server {
	var out []Server
	switch n.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			name := n.Content[i].Value
			var rs yamlServer
			_ = n.Content[i+1].Decode(&rs)
			out = append(out, rs.toServer(firstNonEmpty(rs.Name, name)))
		}
	case yaml.SequenceNode:
		for _, item := range n.Content {
			var rs yamlServer
			_ = item.Decode(&rs)
			out = append(out, rs.toServer(rs.Name))
		}
	}
	return out
}

// ---- small helpers ----

func envKeyNames(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	b, err := json.Marshal(keys)
	if err != nil {
		return ""
	}
	return string(b)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
