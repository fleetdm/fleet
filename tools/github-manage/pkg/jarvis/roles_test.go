package jarvis

import (
	"strings"
	"testing"
)

func TestNormalizeRole(t *testing.T) {
	tests := map[string]string{
		"":             RoleDeveloper,
		"developer":    RoleDeveloper,
		"Manager":      RoleManager,
		"  QA  ":       RoleQA,
		"design":       RoleDesign,
		"unrecognized": RoleDeveloper,
	}
	for in, want := range tests {
		if got := normalizeRole(in); got != want {
			t.Errorf("normalizeRole(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderStartPromptDefaults(t *testing.T) {
	data := PromptData{Issue: 42, Title: "windows mdm", URL: "https://x/42", Branch: "gk-42"}

	dev := renderStartPrompt(RoleDeveloper, nil, data)
	if !strings.Contains(dev, "#42") || !strings.Contains(dev, "windows mdm") || !strings.Contains(dev, "https://x/42") {
		t.Errorf("developer prompt missing issue/title/url: %q", dev)
	}

	qa := renderStartPrompt(RoleQA, nil, data)
	if !strings.Contains(qa, "awaiting QA") || !strings.Contains(qa, "reproduce") {
		t.Errorf("qa prompt should mention QA/reproduce: %q", qa)
	}
	if !strings.Contains(qa, "gk-42") {
		t.Errorf("qa prompt should reference the branch when set: %q", qa)
	}

	// Unknown role falls back to the developer default.
	if got := renderStartPrompt("bogus", nil, data); got != dev {
		t.Errorf("unknown role should fall back to developer prompt: %q", got)
	}
}

func TestRenderStartPromptOverride(t *testing.T) {
	cfg := &Config{StartPrompts: map[string]string{
		RoleQA: "verify #{{.Issue}} ({{.Title}})",
	}}
	got := renderStartPrompt(RoleQA, cfg, PromptData{Issue: 7, Title: "bug"})
	if got != "verify #7 (bug)" {
		t.Errorf("override not applied: %q", got)
	}

	// A blank override falls through to the built-in default.
	cfg2 := &Config{StartPrompts: map[string]string{RoleDeveloper: "   "}}
	dev := renderStartPrompt(RoleDeveloper, nil, PromptData{Issue: 7, Title: "bug"})
	if got := renderStartPrompt(RoleDeveloper, cfg2, PromptData{Issue: 7, Title: "bug"}); got != dev {
		t.Errorf("blank override should use default: %q", got)
	}
}

func TestRenderStartPromptBadTemplateFallsBack(t *testing.T) {
	cfg := &Config{StartPrompts: map[string]string{RoleDeveloper: "{{.Nope"}}
	got := renderStartPrompt(RoleDeveloper, cfg, PromptData{Issue: 9, Title: "t"})
	if got != "Let's work on issue #9: t" {
		t.Errorf("bad template should fall back to plain prompt, got %q", got)
	}
}

func TestConfigEffectiveRole(t *testing.T) {
	if (&Config{}).EffectiveRole() != RoleDeveloper {
		t.Error("empty config role should default to developer")
	}
	if (&Config{Role: "qa"}).EffectiveRole() != RoleQA {
		t.Error("role qa should resolve to RoleQA")
	}
	var nilCfg *Config
	if nilCfg.EffectiveRole() != RoleDeveloper {
		t.Error("nil config should default to developer")
	}
}
