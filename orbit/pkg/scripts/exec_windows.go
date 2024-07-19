//go:build windows

package scripts

import (
	"context"
	"os/exec"
	"path/filepath"
	"time"
)

func ExecCmd(ctx context.Context, scriptPath string, env []string) (output []byte, exitCode int, err error) {
	// initialize to -1 in case the process never starts
	exitCode = -1

	// for Windows, we execute the file with powershell.
	cmd := exec.CommandContext(ctx, "powershell", "-MTA", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	cmd.Env = env
	cmd.Dir = filepath.Dir(scriptPath)
	cmd.WaitDelay = time.Second
	output, err = cmd.CombinedOutput()

	// we still check if the context was cancelled before setting an exitCode !=
	// -1, as killing a process on Windows is not straightforward (see the
	// WaitDelay documentation) and may have timed out even if exit code is
	// reported as 1, so keep it to -1 in that case so that all user messages are
	// as expected.
	if cmd.ProcessState != nil && ctx.Err() == nil {
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
