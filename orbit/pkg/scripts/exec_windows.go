//go:build windows

package scripts

import (
	"context"
	"os/exec"
	"path/filepath"
)

func ExecCmd(ctx context.Context, scriptPath string, env []string) (output []byte, exitCode int, err error) {
	// initialize to -1 in case the process never starts
	exitCode = -1

	// for Windows, we execute the file with powershell.
	cmd := exec.CommandContext(ctx, "powershell", "-MTA", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	cmd.Env = env
	cmd.Dir = filepath.Dir(scriptPath)
	output, err = cmd.CombinedOutput()
	if cmd.ProcessState != nil {
		// The windows exit code is a 32-bit unsigned integer, but the
		// interpreter treats it like a signed integer. When a process
		// is killed, it returns 0xFFFFFFFF (interpreted as -1). We
		// convert the integer to an signed 32-bit integer to flip it
		// to a -1 to match our expectations, and fit in our db column.
		//
		// https://en.wikipedia.org/wiki/Exit_status#Windows
		exitCode = int(int32(cmd.ProcessState.ExitCode()))
	}
	return output, exitCode, err
}
