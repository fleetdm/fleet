//go:build !windows

package scripts

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func ExecCmd(ctx context.Context, scriptPath string, env []string) (output []byte, exitCode int, err error) {
	// initialize to -1 in case the process never starts
	exitCode = -1

	contents, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, -1, ctxerr.Wrapf(ctx, err, "opening script for validation %s", scriptPath)
	}
	directExecute, err := fleet.ValidateShebang(string(contents))
	if err != nil {
		return nil, -1, ctxerr.Wrapf(ctx, err, "validating script %s", scriptPath)
	}

	cmd := exec.CommandContext(ctx, "/bin/sh", scriptPath)

	if directExecute {
		err = os.Chmod(scriptPath, 0o700)
		if err != nil {
			return nil, -1, ctxerr.Wrapf(ctx, err, "marking script as executable %s", scriptPath)
		}
		cmd = exec.CommandContext(ctx, scriptPath)
	}

	if env != nil {
		cmd.Env = env
	}

	cmd.Dir = filepath.Dir(scriptPath)

	// WaitDelay is necessary to ensure that the process is killed when the
	// context is cancelled
	cmd.WaitDelay = time.Second
	output, err = cmd.CombinedOutput()
	if cmd.ProcessState != nil && ctx.Err() == nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	return output, exitCode, err
}
