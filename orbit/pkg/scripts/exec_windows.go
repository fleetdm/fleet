//go:build windows

package scripts

import (
	"context"
	"os/exec"
	"path/filepath"
)

func execCmd(ctx context.Context, scriptPath string) (output []byte, exitCode int, err error) {
	// initialize to -1 in case the process never starts
	exitCode = -1

	// for Windows, we execute the file with powershell.
	cmd := exec.CommandContext(ctx, "powershell", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	cmd.Dir = filepath.Dir(scriptPath)
	output, err = cmd.CombinedOutput()
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	return output, exitCode, err
}
