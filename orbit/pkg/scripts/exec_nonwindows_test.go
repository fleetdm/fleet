//go:build !windows

package scripts

import (
	"context"
	"os"
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
			contents: "ps -o comm= -p $$",
			output:   map[bool]string{true: "/bin/sh", false: "sh"}[runtime.GOOS == "darwin"],
		},
		{
			name:     "sh shebang",
			contents: "#!/bin/sh\nps -o comm= -p $$",
			output:   map[bool]string{true: "/bin/sh", false: "sh_shebang.sh"}[runtime.GOOS == "darwin"],
		},
		{
			name:     "zsh shebang",
			contents: "#!" + zshPath + "\nps -o comm= -p $$",
			output:   map[bool]string{true: zshPath, false: "zsh"}[runtime.GOOS == "darwin"],
		},
		{
			name:     "zsh shebang with args",
			contents: "#!" + zshPath + " -e\nps -o comm= -p $$",
			output:   map[bool]string{true: zshPath, false: "zsh"}[runtime.GOOS == "darwin"],
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
			scriptPath := strings.ReplaceAll(tc.name, " ", "_") + ".sh"
			scriptPath = filepath.Join(tmpDir, scriptPath)
			err := os.WriteFile(scriptPath, []byte(tc.contents), os.ModePerm)
			require.NoError(t, err)

			output, exitCode, err := execCmd(context.Background(), scriptPath)
			require.Equal(t, tc.output, strings.TrimSpace(string(output)))
			require.Equal(t, tc.exitCode, exitCode)
			require.ErrorIs(t, tc.error, err)
		})
	}
}
