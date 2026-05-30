package homebrew

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/stretchr/testify/require"
)

func TestShellQuote(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`/Users/Shared/Max 9`, `'/Users/Shared/Max 9'`},
		{`~/Documents/Max 9`, `'~/Documents/Max 9'`},
		// Apostrophe in the path (e.g. the company name "Cycling '74") must be
		// escaped so the result is still a single, valid shell argument.
		{`~/Library/Application Support/Cycling '74`, `'~/Library/Application Support/Cycling '\''74'`},
	}
	for _, c := range cases {
		require.Equal(t, c.want, shellQuote(c.in), "input %q", c.in)
	}
}

// TestProcessUninstallArtifactQuotingValidShell ensures the uninstall script
// generated for filesystem paths containing an apostrophe is valid shell that
// passes `bash -n`. Regression test for paths like "Cycling '74" that broke
// shell quoting when wrapped in bare single quotes.
func TestProcessUninstallArtifactQuotingValidShell(t *testing.T) {
	u := &brewUninstall{
		Trash: optjson.StringOr[[]string]{
			IsOther: true,
			Other: []string{
				"/Users/Shared/Max 9",
				"~/Library/Application Support/Cycling '74",
			},
		},
		Delete: optjson.StringOr[[]string]{String: "~/Library/Caches/Cycling '74"},
		RmDir:  optjson.StringOr[[]string]{String: "~/Library/Cycling '74"},
	}

	sb := newScriptBuilder()
	processUninstallArtifact(u, sb)
	script := sb.String()

	require.Contains(t, script, `trash $LOGGED_IN_USER '~/Library/Application Support/Cycling '\''74'`)

	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skipf("bash not available: %v", err)
	}
	cmd := exec.Command(bashPath, "-n")
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "bash -n failed for generated script:\n%s\noutput: %s", script, out)
}
