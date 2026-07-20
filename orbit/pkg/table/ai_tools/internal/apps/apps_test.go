package apps

import (
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/proc"
)

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

func TestMarkRunningWordBoundary(t *testing.T) {
	dia := knownApp{name: "dia", processNames: []string{"dia"}}

	// A short token like "dia" must not match unrelated processes by substring
	// (e.g. macOS "mediaanalysisd" contains "dia").
	var falsePos App
	markRunning(&falsePos, dia, &proc.Snapshot{Procs: map[int]proc.Process{
		42: {PID: 42, Name: "mediaanalysisd"},
	}})
	if falsePos.Running != 0 {
		t.Errorf("dia falsely matched mediaanalysisd: Running=%d PID=%d", falsePos.Running, falsePos.PID)
	}

	// A genuine match (exact, case-insensitive) is still detected.
	var match App
	markRunning(&match, dia, &proc.Snapshot{Procs: map[int]proc.Process{
		7: {PID: 7, Name: "Dia"},
	}})
	if match.Running != 1 || match.PID != 7 {
		t.Errorf("dia should match process \"Dia\": Running=%d PID=%d", match.Running, match.PID)
	}
}
