//go:build windows

package scripts

import (
	"context"
	"os/exec"
	"path/filepath"
)

func execCmd(ctx context.Context, scriptPath string) ([]byte, error) {
	// initialize to -1 in case the process never starts
	exitCode = -1

	// for Windows, we execute the file directly. It has the '.ps1' extension and
	// as such will be executed as a PowerShell script.
	cmd := exec.CommandContext(ctx, scriptPath)
	cmd.Dir = filepath.Dir(scriptPath)
	output, err = cmd.CombinedOutput()
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	return output, exitCode, err
}
