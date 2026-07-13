package apps

import "testing"

func TestMatchKnown(t *testing.T) {
	cases := []struct {
		tokens  []string
		wantOK  bool
		want    string
		apiPort int
	}{
		{[]string{"Claude.app", "Claude", "com.anthropic.claude"}, true, "claude-desktop", 0},
		{[]string{"Ollama.app", "Ollama", "com.electron.ollama"}, true, "ollama", 11434},
		{[]string{"LM Studio.app", "LM Studio", "ai.lmstudio.app"}, true, "lm-studio", 1234},
		{[]string{"Slack.app", "Slack", "com.tinyspeck.slackmacgap"}, false, "", 0},
		{[]string{"Google Chrome.app", "Google Chrome"}, false, "", 0},
		{[]string{"Comet.app", "Comet", "ai.perplexity.comet"}, true, "comet", 0},
		{[]string{"Dia.app", "Dia", "company.thebrowser.dia"}, true, "dia", 0},
		{[]string{"Perplexity.app", "Perplexity", "ai.perplexity.macos"}, true, "perplexity", 0},
	}
	for _, c := range cases {
		k, ok := matchKnown(c.tokens...)
		if ok != c.wantOK {
			t.Errorf("matchKnown(%v) ok=%v want %v", c.tokens, ok, c.wantOK)
			continue
		}
		if ok && (k.name != c.want || k.apiPort != c.apiPort) {
			t.Errorf("matchKnown(%v) = name=%q apiPort=%d want name=%q apiPort=%d",
				c.tokens, k.name, k.apiPort, c.want, c.apiPort)
		}
	}
}
