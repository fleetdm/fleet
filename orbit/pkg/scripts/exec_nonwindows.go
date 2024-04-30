//go:build !windows

package scripts

import (
	"context"
	"os/exec"
	"path/filepath"
)

func ExecCmd(ctx context.Context, scriptPath string) (output []byte, exitCode int, err error) {
	// initialize to -1 in case the process never starts
	exitCode = -1

	cmd := exec.CommandContext(ctx, "/bin/sh", scriptPath)
	cmd.Dir = filepath.Dir(scriptPath)
	output, err = cmd.CombinedOutput()
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	return output, exitCode, err
}
