//go:build !windows

package scripts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func execCmd(ctx context.Context, scriptPath string) (output []byte, exitCode int, err error) {
	// initialize to -1 in case the process never starts
	exitCode = -1

	contents, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, -1, err
	}
	directExecute, err := fleet.ValidateShebang(string(contents))
	if err != nil {
		return nil, -1, err
	}

	cmd := exec.CommandContext(ctx, "/bin/sh", scriptPath)

	if directExecute {
		err = os.Chmod(scriptPath, 0766)
		if err != nil {
			return nil, -1, fmt.Errorf("marking script as executable: %w", err)
		}
		cmd = exec.CommandContext(ctx, scriptPath)
	}

	cmd.Dir = filepath.Dir(scriptPath)
	output, err = cmd.CombinedOutput()
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	return output, exitCode, err
}
