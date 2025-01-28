//go:build !windows

package scripts

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestExecCmdNonWindows(t *testing.T) {
	zshPath := "/bin/zsh"
	if runtime.GOOS == "linux" {
		zshPath = "/usr/bin/zsh"
	}

	tests := []struct {
		name     string
		contents string
		output   string
		exitCode int
		error    error
	}{
		{
			name:     "no shebang",
			contents: "[ -z \"$ZSH_VERSION\" ] && echo 1",
			output:   "1",
		},
		{
			name:     "sh shebang",
			contents: "#!/bin/sh\n[ -z \"$ZSH_VERSION\" ] && echo 1",
			output:   "1",
		},
		{
			name:     "zsh shebang",
			contents: "#!" + zshPath + "\n[ -n \"$ZSH_VERSION\" ] && echo 1",
			output:   "1",
		},
		{
			name:     "zsh shebang with args",
			contents: "#!" + zshPath + " -e\n[ -n \"$ZSH_VERSION\" ] && echo 1",
			output:   "1",
		},
		{
			name:     "unsupported shebang",
			contents: "#!/bin/python",
			error:    fleet.ErrUnsupportedInterpreter,
			exitCode: -1,
		},
	}

	tmpDir := t.TempDir()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if strings.HasPrefix(tc.contents, "#!"+zshPath) {
				// skip if zsh is not installed
				if _, err := exec.LookPath(zshPath); err != nil {
					t.Skipf("zsh not installed: %s", err)
				}
			}
			scriptPath := strings.ReplaceAll(tc.name, " ", "_") + ".sh"
			scriptPath = filepath.Join(tmpDir, scriptPath)
			err := os.WriteFile(scriptPath, []byte(tc.contents), os.ModePerm) //nolint:gosec // ignore non-standard permissions
			require.NoError(t, err)

			output, exitCode, err := ExecCmd(context.Background(), scriptPath, nil)
			require.Equal(t, tc.output, strings.TrimSpace(string(output)))
			require.Equal(t, tc.exitCode, exitCode)
			require.ErrorIs(t, err, tc.error)
		})
	}
}

func writeTestScript(content string) (string, error) {
	tmpfile, err := ioutil.TempFile("", "testscript*.sh")
	if err != nil {
		return "", err
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		tmpfile.Close()
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	err = os.Chmod(tmpfile.Name(), 0o700)
	if err != nil {
		return "", err
	}

	return tmpfile.Name(), nil
}

func TestExecCmdTimeout(t *testing.T) {
	scriptContent := `#!/bin/sh
	sleep 5
	echo "Finished"`
	scriptPath, err := writeTestScript(scriptContent)
	if err != nil {
		t.Fatalf("Failed to write test script: %v", err)
	}
	defer os.Remove(scriptPath)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	output, exitCode, err := ExecCmd(ctx, scriptPath, nil)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "signal: killed")
	if exitCode != -1 {
		t.Fatalf("Expected exit code -1, got: %d", exitCode)
	}
	if len(output) != 0 {
		t.Fatalf("Expected no output, got: %s", output)
	}
	require.True(t, time.Since(start) <= 5*time.Second)
}

func TestExecCmdSuccess(t *testing.T) {
	scriptContent := `#!/bin/sh
	echo "Hello, World!"`
	scriptPath, err := writeTestScript(scriptContent)
	if err != nil {
		t.Fatalf("Failed to write test script: %v", err)
	}
	defer os.Remove(scriptPath)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, exitCode, err := ExecCmd(ctx, scriptPath, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("Expected exit code 0, got: %d", exitCode)
	}
	expectedOutput := "Hello, World!\n"
	if string(output) != expectedOutput {
		t.Fatalf("Expected output %q, got: %q", expectedOutput, output)
	}
}
