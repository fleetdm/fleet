//go:build !windows

package scripts

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

func execCmd(ctx context.Context, scriptPath string) (output []byte, exitCode int, err error) {
	// initialize to -1 in case the process never starts
	exitCode = -1

	err = os.Chmod(scriptPath, 0766)
	if err != nil {
		return nil, -1, ctxerr.Wrap(ctx, err, "marking script as executable")
	}
	cmd := exec.CommandContext(ctx, scriptPath)
	cmd.Dir = filepath.Dir(scriptPath)
	output, err = cmd.CombinedOutput()
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	return output, exitCode, err
}
