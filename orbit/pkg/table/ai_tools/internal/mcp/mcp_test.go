package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/proc"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanConfigs(t *testing.T) {
	home := t.TempDir()

	write(t, filepath.Join(home, ".claude.json"), `{
	  "mcpServers": {
	    "fs": {"command":"npx","args":["-y","@modelcontextprotocol/server-filesystem","/tmp"],"env":{"TOKEN":"x"}},
	    "remote-api": {"type":"http","url":"https://mcp.example.com/x"}
	  },
	  "projects": {"/work/proj": {"mcpServers": {"projsrv": {"command":"node","args":["server.js"]}}}}
	}`)
	write(t, filepath.Join(home, ".codeium", "windsurf", "mcp_config.json"),
		`{"mcpServers":{"wind":{"command":"uvx","args":["mcp-server-time"]}}}`)
	write(t, filepath.Join(home, ".config", "zed", "settings.json"),
		`{"context_servers":{"zedsrv":{"command":{"path":"/usr/bin/mcp","args":["--x"]}}}}`)
	write(t, filepath.Join(home, ".vscode", "mcp.json"),
		`{"servers":{"vs":{"type":"sse","url":"https://vs.example.com/sse"}}}`)
	write(t, filepath.Join(home, ".continue", "config.yaml"),
		"mcpServers:\n  - name: cyaml\n    command: python\n    args: ['-m','mcp_server_x']\n")

	write(t, filepath.Join(home, ".hermes", "config.yaml"), `
mcp_servers:
  fleet:
    url: "http://localhost:8081/sse"
    transport: sse
    headers:
      Authorization: "Bearer SECRET_SHOULD_NOT_LEAK"
  opendesign:
    command: "npx"
    args: ["-y", "open-design-mcp"]
`)
	write(t, filepath.Join(home, ".grok", "config.toml"), `
[mcp_servers.repowise]
command = "repowise"
args = ["mcp", "/tmp/proj", "--transport", "stdio"]
enabled = true
`)
	write(t, filepath.Join(home, ".codex", "config.toml"), `
[mcp_servers.gitnexus]
command = "/opt/homebrew/bin/gitnexus"
args = ["mcp"]
`)
	// Jan paths differ by OS; seed all catalog locations so the test is portable.
	janJSON := `{"mcpServers":{"exa":{"type":"http","url":"https://mcp.exa.ai/mcp","command":""}}}`
	for _, p := range []string{
		filepath.Join(home, "Library", "Application Support", "Jan", "data", "mcp_config.json"),
		filepath.Join(home, "AppData", "Roaming", "Jan", "data", "mcp_config.json"),
		filepath.Join(home, ".config", "Jan", "data", "mcp_config.json"),
		filepath.Join(home, ".local", "share", "Jan", "data", "mcp_config.json"),
	} {
		write(t, p, janJSON)
	}
	write(t, filepath.Join(home, ".openclaw", "openclaw.json"), `{
	  "mcp": {
	    "servers": {
	      "context7": {"command":"npx","args":["-y","@upstash/context7-mcp"]},
	      "remote-http": {"type":"http","url":"https://mcp.openclaw.example/mcp"}
	    }
	  }
	}`)
	write(t, filepath.Join(home, ".gemini", "config", "mcp_config.json"),
		`{"mcpServers":{"gem":{"command":"uvx","args":["mcp-server-time"]}}}`)

	by := map[string]Server{}
	byClient := map[string][]Server{}
	for _, s := range ScanConfigs(homes.Home{Dir: home, Username: "tester"}) {
		by[s.ServerName] = s
		byClient[s.Client] = append(byClient[s.Client], s)
	}

	cases := []struct{ name, loc, transport string }{
		{"fs", "local", "stdio"},
		{"remote-api", "remote", "http"},
		{"projsrv", "local", "stdio"},
		{"wind", "local", "stdio"},
		{"zedsrv", "local", "stdio"},
		{"vs", "remote", "sse"},
		{"cyaml", "local", "stdio"},
		{"fleet", "remote", "sse"},
		{"opendesign", "local", "stdio"},
		{"repowise", "local", "stdio"},
		{"gitnexus", "local", "stdio"},
		{"exa", "remote", "http"},
		{"context7", "local", "stdio"},
		{"remote-http", "remote", "http"},
		{"gem", "local", "stdio"},
	}
	for _, c := range cases {
		s, ok := by[c.name]
		if !ok {
			t.Errorf("server %q not found (found %d total: %v)", c.name, len(by), keysOf(by))
			continue
		}
		if s.Location != c.loc {
			t.Errorf("%s: location=%q want %q", c.name, s.Location, c.loc)
		}
		if s.Transport != c.transport {
			t.Errorf("%s: transport=%q want %q", c.name, s.Transport, c.transport)
		}
	}
	if by["remote-api"].URL == "" {
		t.Error("remote-api: expected non-empty URL")
	}
	if by["fleet"].URL != "http://localhost:8081/sse" {
		t.Errorf("fleet URL=%q", by["fleet"].URL)
	}
	if by["fleet"].Client != "hermes" {
		t.Errorf("fleet client=%q want hermes", by["fleet"].Client)
	}
	if !strings.Contains(by["fleet"].RiskFlags, "cleartext_endpoint") {
		t.Errorf("fleet risk_flags=%q missing cleartext_endpoint", by["fleet"].RiskFlags)
	}
	if !strings.Contains(by["fleet"].RiskFlags, "plaintext_secret") {
		t.Errorf("fleet risk_flags=%q missing plaintext_secret (Authorization header name)", by["fleet"].RiskFlags)
	}
	if strings.Contains(by["fleet"].EnvKeys, "SECRET") || strings.Contains(by["fleet"].EnvKeys, "Bearer") {
		t.Errorf("fleet env_keys leaked a header value: %q", by["fleet"].EnvKeys)
	}
	if by["exa"].URL != "https://mcp.exa.ai/mcp" {
		t.Errorf("exa URL=%q", by["exa"].URL)
	}
	if by["exa"].Client != "jan" {
		t.Errorf("exa client=%q want jan", by["exa"].Client)
	}
	if by["repowise"].Client != "grok" {
		t.Errorf("repowise client=%q want grok", by["repowise"].Client)
	}
	if by["gitnexus"].Client != "codex" {
		t.Errorf("gitnexus client=%q want codex", by["gitnexus"].Client)
	}
	if by["remote-http"].Client != "openclaw" {
		t.Errorf("remote-http client=%q want openclaw", by["remote-http"].Client)
	}
	if by["fs"].EnvKeys != `["TOKEN"]` {
		t.Errorf("fs: env_keys=%q want [\"TOKEN\"] (names only, no values)", by["fs"].EnvKeys)
	}
	if strings.Contains(by["fs"].EnvKeys, "x") {
		t.Error("fs: env_keys leaked a value")
	}
	for _, wantClient := range []string{"hermes", "grok", "codex", "jan", "openclaw", "gemini"} {
		if len(byClient[wantClient]) == 0 {
			t.Errorf("expected at least one server from client %q", wantClient)
		}
	}
}

func keysOf(m map[string]Server) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestCorrelate(t *testing.T) {
	declared := []Server{{
		ServerName: "fs", Command: "npx",
		Args:   `["-y","@modelcontextprotocol/server-filesystem","/tmp"]`,
		Source: "config", Location: "local",
	}}
	snap := &proc.Snapshot{Procs: map[int]proc.Process{
		42: {PID: 42, Name: "node", Cmdline: "node /x/npx @modelcontextprotocol/server-filesystem /tmp"},
		7:  {PID: 7, Name: "node", Cmdline: "node /opt/mcp-server-weather/index.js"},
	}}

	out := Correlate(declared, snap)

	var fs *Server
	gotProcess := false
	for i := range out {
		switch {
		case out[i].ServerName == "fs":
			fs = &out[i]
		case out[i].Source == "process":
			gotProcess = true
		}
	}
	if fs == nil || fs.Running != 1 || fs.PID != 42 || fs.Source != "both" {
		t.Fatalf("fs not correlated to running process: %+v", fs)
	}
	if !gotProcess {
		t.Error("undeclared running MCP server (mcp-server-weather) not discovered")
	}
}

// TestCorrelateInlineEvalLauncher guards against the claude-mem bug: a
// launcher invoked as `node -e "<script>"` has its script body joined into
// Cmdline as if it were several separate arguments, so naively re-splitting
// Cmdline on whitespace shreds the script into arbitrary JS fragments. With
// CmdlineSlice preserving the true argv boundaries, the -e script is one
// element and the launcher filename can still be recovered from it.
func TestCorrelateInlineEvalLauncher(t *testing.T) {
	script := `(function(){const p=require('path');const s=p.join(__dirname,'.claude-mem','mcp-server.cjs');require(s).start();})()`
	argv := []string{"node", "-e", script}
	snap := &proc.Snapshot{Procs: map[int]proc.Process{
		99: {PID: 99, Name: "node", Cmdline: strings.Join(argv, " "), CmdlineSlice: argv},
	}}

	out := Correlate(nil, snap)
	if len(out) != 1 {
		t.Fatalf("expected 1 heuristic server row, got %d: %+v", len(out), out)
	}
	got := out[0].ServerName
	if got != "mcp-server.cjs" {
		t.Errorf("ServerName = %q, want %q (must not be a raw JS code fragment)", got, "mcp-server.cjs")
	}
	if strings.ContainsAny(got, "(){};=") {
		t.Errorf("ServerName %q looks like a code fragment, not a clean name", got)
	}

	var args []string
	if err := json.Unmarshal([]byte(out[0].Args), &args); err != nil {
		t.Fatalf("Args is not valid JSON: %v (%q)", err, out[0].Args)
	}
	if len(args) != 2 || args[0] != "-e" || args[1] != script {
		t.Errorf("Args = %v, want [\"-e\", %q] as two elements, not shredded on whitespace", args, script)
	}
}
