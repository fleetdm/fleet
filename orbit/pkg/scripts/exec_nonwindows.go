//go:build !windows

package scripts

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
		err = os.Chmod(scriptPath, 0o700) // nolint:gosec // G302
		if err != nil {
			return nil, -1, ctxerr.Wrapf(ctx, err, "marking script as executable %s", scriptPath)
		}
		cmd = exec.CommandContext(ctx, scriptPath)

		// NixOS has no /bin/bash or /usr/bin/python3; run the script with the
		// same-named interpreter from the system profile when the shebang's path
		// is missing. PATH is deliberately not consulted since this runs as root.
		if isNixOS() {
			interp, arg := shebangInterpreter(string(contents))
			if interp == "" {
				// unreachable: ValidateShebang only allows direct execution with an interpreter
				return nil, -1, ctxerr.Errorf(ctx, "no shebang interpreter in script %s", scriptPath)
			}
			if _, statErr := os.Stat(interp); errors.Is(statErr, fs.ErrNotExist) {
				resolved := filepath.Join(nixosSystemBinDir, filepath.Base(interp))
				if _, err := os.Stat(resolved); err != nil {
					return nil, -1, ctxerr.Wrapf(ctx, err, "shebang interpreter %s not found on this host", interp)
				}
				args := []string{scriptPath}
				if arg != "" {
					args = []string{arg, scriptPath}
				}
				cmd = exec.CommandContext(ctx, resolved, args...)
			}
		}
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

// nixosMarkerFile is the file nixos-rebuild itself checks to identify NixOS.
// nixosSystemBinDir is the root-owned, read-only store-backed system profile.
var (
	nixosMarkerFile   = "/etc/NIXOS"
	nixosSystemBinDir = "/run/current-system/sw/bin"
)

func isNixOS() bool {
	_, err := os.Stat(nixosMarkerFile)
	return err == nil
}

// shebangInterpreter returns the interpreter path from the script's shebang
// line and, like the Linux kernel, everything after it as a single argument.
func shebangInterpreter(contents string) (interp, arg string) {
	line, _, _ := strings.Cut(contents, "\n")
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "#!") {
		return "", ""
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, "#!"))
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return "", ""
	}
	interp = fields[0]
	arg = strings.TrimSpace(strings.TrimPrefix(rest, interp))
	return interp, arg
}
