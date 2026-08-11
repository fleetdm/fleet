package execuser

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadEnvFromProcFile(t *testing.T) {
	dir := t.TempDir()
	environPath := filepath.Join(dir, "environ")

	testCases := []struct {
		name     string
		environ  string
		envVar   string
		expected string
	}{
		{
			name:     "DISPLAY found",
			environ:  "HOME=/home/foo\x00DISPLAY=:0\x00LANG=en_US.UTF-8",
			envVar:   "DISPLAY",
			expected: ":0",
		},
		{
			name:     "DISPLAY :1",
			environ:  "HOME=/home/foo\x00DISPLAY=:1\x00LANG=en_US.UTF-8",
			envVar:   "DISPLAY",
			expected: ":1",
		},
		{
			name:     "DISPLAY not present",
			environ:  "HOME=/home/foo\x00LANG=en_US.UTF-8",
			envVar:   "DISPLAY",
			expected: "",
		},
		{
			name:     "empty environ",
			environ:  "",
			envVar:   "DISPLAY",
			expected: "",
		},
		{
			name:     "does not match prefix substring",
			environ:  "DISPLAY_NUM=5\x00DISPLAY=:2",
			envVar:   "DISPLAY",
			expected: ":2",
		},
		{
			name:     "other env var",
			environ:  "HOME=/home/foo\x00DISPLAY=:0\x00WAYLAND_DISPLAY=wayland-0",
			envVar:   "WAYLAND_DISPLAY",
			expected: "wayland-0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(environPath, []byte(tc.environ), 0o644))
			result, err := readEnvFromProcFile(environPath, tc.envVar)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestReadEnvFromProcFileMissing(t *testing.T) {
	_, err := readEnvFromProcFile("/nonexistent/path/environ", "DISPLAY")
	require.Error(t, err)
}

// TestBracketScriptRoundTrip runs the script under a real /bin/sh, so the markers
// it writes and the ones parseBracketedOutput looks for are checked against each
// other rather than against a hand-built string.
func TestBracketScriptRoundTrip(t *testing.T) {
	nonce, err := newOutputNonce()
	require.NoError(t, err)
	require.Len(t, nonce, 32)

	testCases := []struct {
		name     string
		args     []string
		expected string
		exitCode int
	}{
		{
			name:     "output and a zero status",
			args:     []string{"printf", "secret\n"},
			expected: "secret\n",
		},
		{
			name: "no output",
			args: []string{"true"},
		},
		{
			// The wrapper's own status is echo's, so a non-zero one only survives
			// through the closing marker.
			name:     "output and a non-zero status",
			args:     []string{"sh", "-c", "printf partial; exit 3"},
			expected: "partial",
			exitCode: 3,
		},
		{
			name:     "argument containing spaces",
			args:     []string{"printf", "%s", "two words"},
			expected: "two words",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("sh", append([]string{"-s"}, tc.args...)...) // #nosec G204
			cmd.Stdin = strings.NewReader(bracketScript(nonce))
			raw, err := cmd.Output()
			require.NoError(t, err)

			output, exitCode, ok := parseBracketedOutput(raw, nonce)
			require.True(t, ok)
			require.Equal(t, tc.expected, string(output))
			require.Equal(t, tc.exitCode, exitCode)
		})
	}
}

// TestBracketScriptSurvivesAnOuterShell mimics `sudo -i`, which passes the whole
// invocation through the login user's shell.
//
// The script has to reach the inner shell unexpanded. When it was passed as an
// argument instead of over stdin, the login shell substituted its own positional
// parameters first, so the inner shell was told to run the login shell's argv[0]
// and the command never ran at all.
func TestBracketScriptSurvivesAnOuterShell(t *testing.T) {
	nonce, err := newOutputNonce()
	require.NoError(t, err)

	cmd := exec.Command("sh", "-c", "sh -s printf secret") // #nosec G204
	cmd.Stdin = strings.NewReader(bracketScript(nonce))
	raw, err := cmd.Output()
	require.NoError(t, err)

	output, exitCode, ok := parseBracketedOutput(raw, nonce)
	require.True(t, ok)
	require.Equal(t, "secret", string(output))
	require.Equal(t, 0, exitCode)
}

func TestParseBracketedOutput(t *testing.T) {
	const nonce = "0123456789abcdef"

	testCases := []struct {
		name     string
		raw      string
		expected string
		exitCode int
		ok       bool
	}{
		{
			name:     "nothing around the markers",
			raw:      "B-" + nonce + "\nsecret\nE-" + nonce + ":0\n",
			expected: "secret\n",
			ok:       true,
		},
		{
			// What the login shell's profile scripts produce.
			name:     "output before",
			raw:      "hello from profile\nB-" + nonce + "\nsecret\nE-" + nonce + ":0\n",
			expected: "secret\n",
			ok:       true,
		},
		{
			// ~/.bash_logout runs when the login shell exits.
			name:     "output after the status",
			raw:      "B-" + nonce + "\nsecret\nE-" + nonce + ":0\ngoodbye\n",
			expected: "secret\n",
			ok:       true,
		},
		{
			// The dialog was canceled, so it wrote nothing and exited 1.
			name:     "no output and a non-zero status",
			raw:      "hello\nB-" + nonce + "\nE-" + nonce + ":1\n",
			expected: "",
			exitCode: 1,
			ok:       true,
		},
		{
			name:     "surrounding output contains an earlier begin marker",
			raw:      "B-" + nonce + "\ndecoy\nB-" + nonce + "\nsecret\nE-" + nonce + ":0\n",
			expected: "secret\n",
			ok:       true,
		},
		{
			name: "markers are for another invocation",
			raw:  "B-someothernonce\nsecret\nE-someothernonce:0\n",
		},
		{
			name: "no markers at all",
			raw:  "hello from profile\nsecret\n",
		},
		{
			name: "begin marker only",
			raw:  "B-" + nonce + "\nsecret\n",
		},
		{
			name: "status is not a number",
			raw:  "B-" + nonce + "\nsecret\nE-" + nonce + ":oops\n",
		},
		{
			name: "empty",
			raw:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, exitCode, ok := parseBracketedOutput([]byte(tc.raw), nonce)
			require.Equal(t, tc.ok, ok)
			if !tc.ok {
				return
			}
			require.Equal(t, tc.expected, string(output))
			require.Equal(t, tc.exitCode, exitCode)
		})
	}
}
