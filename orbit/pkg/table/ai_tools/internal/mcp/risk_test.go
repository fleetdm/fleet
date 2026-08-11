package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
)

func TestIsPinned(t *testing.T) {
	cases := map[string]bool{
		"@modelcontextprotocol/server-filesystem":       false, // scoped, no version
		"@modelcontextprotocol/server-filesystem@1.2.3": true,
		"mcp-server-time":       false, // bare, no version
		"mcp-server-time@0.4.0": true,
		"foo@latest":            false,
		"foo@next":              false,
		"":                      false,
	}
	for pkg, want := range cases {
		if got := isPinned(pkg); got != want {
			t.Errorf("isPinned(%q)=%v want %v", pkg, got, want)
		}
	}
}

func TestFetchSpec(t *testing.T) {
	runner, pkg, ok := fetchSpec("npx", `["-y","@modelcontextprotocol/server-filesystem","/tmp"]`)
	if !ok || runner != "npx" || pkg != "@modelcontextprotocol/server-filesystem" {
		t.Errorf("fetchSpec npx = (%q,%q,%v)", runner, pkg, ok)
	}
	if _, _, ok := fetchSpec("node", `["server.js"]`); ok {
		t.Error("node should not be a fetch runner")
	}
	if _, _, ok := fetchSpec("/usr/local/bin/uvx", `["mcp-server-time"]`); !ok {
		t.Error("absolute-path uvx should be detected as a fetch runner")
	}
}

func TestHasSecretEnv(t *testing.T) {
	if !hasSecretEnv(`["GITHUB_TOKEN","FOO"]`) {
		t.Error("GITHUB_TOKEN should flag")
	}
	if !hasSecretEnv(`["ANTHROPIC_API_KEY"]`) {
		t.Error("ANTHROPIC_API_KEY should flag")
	}
	if hasSecretEnv(`["PATH","HOME","REGION"]`) {
		t.Error("non-secret names should not flag")
	}
	if hasSecretEnv("") {
		t.Error("empty should not flag")
	}
}

func TestEnrichRiskFlags(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, ".claude.json")
	content := `{
	  "mcpServers": {
	    "fs": {"command":"npx","args":["-y","@modelcontextprotocol/server-filesystem","/"],"env":{"GITHUB_TOKEN":"ghp_x"}},
	    "shell": {"command":"node","args":["mcp-server-commands/index.js"]},
	    "remote": {"type":"http","url":"http://insecure.example.com/mcp"},
	    "pinned": {"command":"npx","args":["mcp-server-time@1.0.0"]}
	  }
	}`
	if err := os.WriteFile(cfg, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	by := map[string]Server{}
	for _, s := range ScanConfigs(homes.Home{Dir: home, Username: "t"}) {
		by[s.ServerName] = s
	}

	fs := by["fs"]
	for _, want := range []string{"remote_fetch_exec", "unpinned_dependency", "plaintext_secret", "mcp_fs_write", "world_readable_config"} {
		if !strings.Contains(fs.RiskFlags, want) {
			t.Errorf("fs.RiskFlags=%q missing %q", fs.RiskFlags, want)
		}
	}
	if fs.SHA256 == "" {
		t.Error("fs.SHA256 should be set (config hash)")
	}
	if fs.LaunchHash == "" {
		t.Error("fs.LaunchHash should be set")
	}
	if !strings.Contains(fs.Capabilities, "fs-write") {
		t.Errorf("fs.Capabilities=%q missing fs-write", fs.Capabilities)
	}

	if sh := by["shell"]; !strings.Contains(sh.RiskFlags, "mcp_shell_exec") {
		t.Errorf("shell.RiskFlags=%q missing mcp_shell_exec", sh.RiskFlags)
	}
	if rem := by["remote"]; !strings.Contains(rem.RiskFlags, "cleartext_endpoint") {
		t.Errorf("remote.RiskFlags=%q missing cleartext_endpoint", rem.RiskFlags)
	}
	if p := by["pinned"]; strings.Contains(p.RiskFlags, "unpinned_dependency") {
		t.Errorf("pinned.RiskFlags=%q should not be unpinned", p.RiskFlags)
	}
}
