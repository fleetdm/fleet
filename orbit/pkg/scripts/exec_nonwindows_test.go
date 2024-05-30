//go:build !windows

package scripts

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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
			err := os.WriteFile(scriptPath, []byte(tc.contents), os.ModePerm)
			require.NoError(t, err)

			output, exitCode, err := ExecCmd(context.Background(), scriptPath, nil)
			require.Equal(t, tc.output, strings.TrimSpace(string(output)))
			require.Equal(t, tc.exitCode, exitCode)
			require.ErrorIs(t, err, tc.error)
		})
	}
}
