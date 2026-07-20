package mcp

import (
	"os"
	"path/filepath"
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

	by := map[string]Server{}
	for _, s := range ScanConfigs(homes.Home{Dir: home, Username: "tester"}) {
		by[s.ServerName] = s
	}

	cases := []struct{ name, loc, transport string }{
		{"fs", "local", "stdio"},
		{"remote-api", "remote", "http"},
		{"projsrv", "local", "stdio"},
		{"wind", "local", "stdio"},
		{"zedsrv", "local", "stdio"},
		{"vs", "remote", "sse"},
		{"cyaml", "local", "stdio"},
	}
	for _, c := range cases {
		s, ok := by[c.name]
		if !ok {
			t.Errorf("server %q not found (found %d total)", c.name, len(by))
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
	if by["fs"].EnvKeys != `["TOKEN"]` {
		t.Errorf("fs: env_keys=%q want [\"TOKEN\"] (names only, no values)", by["fs"].EnvKeys)
	}
	if strings.Contains(by["fs"].EnvKeys, "x") {
		t.Error("fs: env_keys leaked a value")
	}
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
