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

		// Windows uninstall entries and MSIX packages carry a bare display name
		// with no trailing delimiter; a genuine install must still match.
		{[]string{"Dia"}, true, "dia", 0},
		{[]string{"Jan"}, true, "jan", 1337},
		{[]string{"Comet"}, true, "comet", 0},
		{[]string{"Trae"}, true, "trae", 0},
		{[]string{"lms"}, true, "lm-studio-cli", 0},

		// A bounded occurrence after a mid-word one must still match: the
		// boundary scan may not stop at the first hit.
		{[]string{"NVIDIA Dia"}, true, "dia", 0},

		// Short tokens ("dia", "lms") must not match mid-word inside unrelated
		// software names.
		{[]string{"NVIDIA Control Panel"}, false, "", 0},
		{[]string{"NVIDIA Graphics Driver 591.86"}, false, "", 0},
		{[]string{"NVIDIA Install Application"}, false, "", 0},
		{[]string{"VLC media player"}, false, "", 0},
		{[]string{"Plex Media Server 1.43.1.10611 (x64)"}, false, "", 0},
		{[]string{"vs_minshellmsi"}, false, "", 0},
		{[]string{"vs_minshellmsires"}, false, "", 0},
		{[]string{"Windows Media Player.app"}, false, "", 0},

		// Exact bundle ids. The display name alone is not enough, and a longer
		// or different id must not match.
		{[]string{"Granola.app", "Granola", "com.granola.app"}, true, "granola", 0},
		{[]string{"com.granola.app"}, true, "granola", 0},
		{[]string{"Granola.app", "Granola"}, false, "", 0},
		{[]string{"com.granola.app.helper"}, false, "", 0},
		{[]string{"com.example.granola"}, false, "", 0},
		{[]string{"My Granola Notes"}, false, "", 0},

		{[]string{"Grok Bot.app", "Grok Bot", "com.anysphere.sand"}, true, "grok-bot", 0},
		{[]string{"com.anysphere.sand"}, true, "grok-bot", 0},
		// The Grok CLI (~/.grok, process "grok") is a different product.
		{[]string{"grok"}, false, "", 0},
		{[]string{".grok"}, false, "", 0},
		{[]string{"grok.exe"}, false, "", 0},
		{[]string{"Grok Bot.app", "Grok Bot"}, false, "", 0},
		{[]string{"com.anysphere.sand.helper"}, false, "", 0},

		{[]string{"Muse.app", "Muse", "com.meta.endo"}, true, "meta-muse", 0},
		{[]string{"com.meta.endo"}, true, "meta-muse", 0},
		{[]string{"Muse.app", "Muse"}, false, "", 0},
		{[]string{"com.meta.endo.helper"}, false, "", 0},
		{[]string{"Museeks.app", "Museeks"}, false, "", 0},
		{[]string{"MuseScore 4.app", "MuseScore 4"}, false, "", 0},
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

func TestMatchWindowsPrograms(t *testing.T) {
	cases := []struct {
		display   string
		publisher string
		wantOK    bool
		want      string
	}{
		{"Granola 7.722.1", "Granola", true, "granola"},
		{"granola 7.722.1", "Granola", true, "granola"},
		{"Grok Bot 0.78.3", "SpaceXAI", true, "grok-bot"},
		{"grok bot 0.78.3", "SpaceXAI", true, "grok-bot"},

		// Name without the catalog's required space-separated suffix.
		{"Granola", "Granola", false, ""},
		{"Grok Bot", "SpaceXAI", false, ""},
		// Right name, wrong publisher.
		{"Granola 7.722.1", "Someone Else", false, ""},
		{"Grok Bot 0.78.3", "xAI", false, ""},
		{"Grok Bot 0.78.3", "spacexai", false, ""},
		// Product word in the wrong place, or a different app.
		{"My Granola 1.0", "Granola", false, ""},
		{"GranolaBar 1.0", "Granola", false, ""},
		{"Not Grok Bot 1.0", "SpaceXAI", false, ""},
		{"Grok 1.0", "SpaceXAI", false, ""},
		{"Museeks", "Museeks", false, ""},
		{"MuseScore 4.4", "MuseScore", false, ""},
		{"Muse 10.1", "Meta Platforms, Inc.", false, ""},
		{"", "Granola", false, ""},
		{"Granola 1.0", "", false, ""},
	}
	for _, c := range cases {
		k, ok := matchWindowsProgram(c.display, c.publisher)
		if ok != c.wantOK {
			t.Errorf("matchWindowsProgram(%q, %q) ok=%v want %v", c.display, c.publisher, ok, c.wantOK)
			continue
		}
		if ok && k.name != c.want {
			t.Errorf("matchWindowsProgram(%q, %q) = %q want %q", c.display, c.publisher, k.name, c.want)
		}
	}
}

func TestExactAppsRunningProcess(t *testing.T) {
	grokBot, ok := knownByName("grok-bot")
	if !ok {
		t.Fatal("grok-bot missing from known apps")
	}
	var cli App
	markRunning(&cli, grokBot, &proc.Snapshot{Procs: map[int]proc.Process{
		3: {PID: 3, Name: "grok"},
	}})
	if cli.Running != 0 {
		t.Errorf("grok-bot falsely matched the Grok CLI process: Running=%d PID=%d", cli.Running, cli.PID)
	}
	var desktop App
	markRunning(&desktop, grokBot, &proc.Snapshot{Procs: map[int]proc.Process{
		9: {PID: 9, Name: "Grok Bot.exe"},
	}})
	if desktop.Running != 1 || desktop.PID != 9 {
		t.Errorf("grok-bot should match \"Grok Bot.exe\": Running=%d PID=%d", desktop.Running, desktop.PID)
	}

	muse, ok := knownByName("meta-muse")
	if !ok {
		t.Fatal("meta-muse missing from known apps")
	}
	for _, name := range []string{"MuseScore", "Museeks", "mscore"} {
		var other App
		markRunning(&other, muse, &proc.Snapshot{Procs: map[int]proc.Process{
			4: {PID: 4, Name: name},
		}})
		if other.Running != 0 {
			t.Errorf("meta-muse falsely matched process %q", name)
		}
	}
	var museProc App
	markRunning(&museProc, muse, &proc.Snapshot{Procs: map[int]proc.Process{
		5: {PID: 5, Name: "Muse"},
	}})
	if museProc.Running != 1 || museProc.PID != 5 {
		t.Errorf("meta-muse should match process \"Muse\": Running=%d PID=%d", museProc.Running, museProc.PID)
	}
}
