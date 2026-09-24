package classify

import (
	"strings"
	"testing"
)

func TestVSCodePlugin(t *testing.T) {
	cases := []struct {
		id, name string
		wantAI   bool
	}{
		{"github.copilot", "GitHub Copilot", true},
		{"saoudrizwan.claude-dev", "Cline", true},
		{"continue.continue", "Continue", true},
		{"ms-python.python", "Python", false},
		{"esbenp.prettier-vscode", "Prettier", false},
	}
	for _, c := range cases {
		ai, cat := VSCodePlugin(c.id, c.name)
		if ai != c.wantAI {
			t.Errorf("VSCodePlugin(%q): ai=%v want %v (cat=%q)", c.id, ai, c.wantAI, cat)
		}
		if ai && cat == "" {
			t.Errorf("VSCodePlugin(%q): AI plugin has empty category", c.id)
		}
	}
}

func TestCmdline(t *testing.T) {
	cases := []struct {
		cmd     string
		wantAI  bool
		wantCat string
	}{
		{"npx -y @modelcontextprotocol/server-filesystem /tmp", true, "mcp-server"},
		{"node /opt/mcp-server-weather/index.js", true, "mcp-server"},
		{"ollama serve", true, "inference-api-local"},
		{"python -m aider", true, "agent-runtime"},
		{"/usr/bin/nginx -g daemon off;", false, ""},
	}
	for _, c := range cases {
		ai, cat := Cmdline(c.cmd)
		if ai != c.wantAI || (c.wantAI && cat != c.wantCat) {
			t.Errorf("Cmdline(%q) = (%v, %q), want (%v, %q)", c.cmd, ai, cat, c.wantAI, c.wantCat)
		}
	}
}

func TestLocalPortService(t *testing.T) {
	if svc, ok := LocalPortService(11434); !ok || svc != "ollama" {
		t.Errorf("LocalPortService(11434) = (%q,%v) want (ollama,true)", svc, ok)
	}
	if _, ok := LocalPortService(3000); ok {
		t.Error("LocalPortService(3000): generic port should not classify")
	}
}

func TestMCPCapabilities(t *testing.T) {
	cases := []struct {
		hay  string
		want []string // tags that must be present
	}{
		{"npx -y @modelcontextprotocol/server-filesystem /", []string{"fs-read", "fs-write"}},
		{"node mcp-server-commands/index.js", []string{"shell-exec"}},
		{"uvx mcp-server-git", []string{"repo-write"}},
		{"npx @modelcontextprotocol/server-puppeteer", []string{"browser", "network"}},
	}
	for _, c := range cases {
		got := strings.Join(MCPCapabilities(c.hay), ",")
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("MCPCapabilities(%q)=%q missing %q", c.hay, got, w)
			}
		}
	}
	if caps := MCPCapabilities("node plain-server.js"); caps != nil {
		t.Errorf("unknown server should infer no capabilities, got %v", caps)
	}
}

func TestBrowserExtension(t *testing.T) {
	cases := []struct {
		id, name string
		wantAI   bool
	}{
		{"mfgnpcdebmgmmbjmhmboieiipghabkjf", "ChatGPT for Google", true},    // curated id
		{"unknownidunknownidunknownidunkno", "Monica - AI Assistant", true}, // name fallback
		{"unknownidunknownidunknownidunkno", "perplexity", true},            // name fallback
		{"abcabcabcabcabcabcabcabcabcabcab", "Prettier", false},             // non-AI
		{"hdokiejnpimakedhajhdlcegeplioahd", "LastPass", false},             // non-AI
	}
	for _, c := range cases {
		ai, cat := BrowserExtension(c.id, c.name)
		if ai != c.wantAI {
			t.Errorf("BrowserExtension(%q,%q) ai=%v want %v (cat=%q)", c.id, c.name, ai, c.wantAI, cat)
		}
		if ai && cat == "" {
			t.Errorf("BrowserExtension(%q,%q): AI ext has empty category", c.id, c.name)
		}
	}
}
