// Package scripts implements support to execute scripts on the host when
// requested by the Fleet server.
package scripts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const scriptExecTimeout = 30 * time.Second

// Client defines the methods required for the API requests to the server. The
// fleet.OrbitClient type satisfies this interface.
type Client interface {
	GetHostScript(execID string) (*fleet.HostScriptResult, error)
	SaveHostScriptResult(result *fleet.HostScriptResultPayload) error
}

// Runner is the type that processes scripts to execute, taking care of
// retrieving each script, saving it in a temporary directory, executing it and
// saving the results.
type Runner struct {
	Client                 Client
	ScriptExecutionEnabled bool

	// tempDirFn is the function to call to get the temporary directory to use,
	// inside of which the script-specific subdirectories will be created. If nil,
	// the user's temp dir will be used (can be set to t.TempDir in tests).
	tempDirFn func() string

	// execCmdFn can be set for tests to mock actual execution of the script. If
	// nil, execCmd will be used, which has a different implementation on Windows
	// and non-Windows platforms.
	execCmdFn func(ctx context.Context, scriptPath string) ([]byte, int, error)

	// can be set for tests to replace os.RemoveAll, which is called to remove
	// the script's temporary directory after execution.
	removeAllFn func(string) error
}

// Run processes all scripts identified by the execution IDs.
func (r *Runner) Run(execIDs []string) error {
	var errs []error

	for _, execID := range execIDs {
		if !r.ScriptExecutionEnabled {
			if err := r.runOneDisabled(execID); err != nil {
				errs = append(errs, err)
			}
			continue
		}

		if err := r.runOne(execID); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		// NOTE: when we upgrade to Go1.20, we can use errors.Join, but for now we
		// just concatenate the error messages in a single error that will be logged
		// by orbit.
		var sb strings.Builder
		for i, e := range errs {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(e.Error())
		}
		return errors.New(sb.String())
	}
	return nil
}

func (r *Runner) runOne(execID string) (finalErr error) {
	const maxOutputRuneLen = 10000

	script, err := r.Client.GetHostScript(execID)
	if err != nil {
		return fmt.Errorf("get host script: %w", err)
	}

	if script.ExitCode != nil {
		// already a result stored for this execution, skip, it shouldn't be sent
		// again by Fleet.
		return nil
	}

	runDir, err := r.createRunDir(execID)
	if err != nil {
		return fmt.Errorf("create run directory: %w", err)
	}
	// prevent destruction of dir if this env var is set
	if os.Getenv("FLEET_PREVENT_SCRIPT_TEMPDIR_DELETION") == "" {
		defer func() {
			fn := os.RemoveAll
			if r.removeAllFn != nil {
				fn = r.removeAllFn
			}
			err := fn(runDir)
			if finalErr == nil && err != nil {
				finalErr = fmt.Errorf("remove temp dir: %w", err)
			}
		}()
	}

	ext := ".sh"
	if runtime.GOOS == "windows" {
		ext = ".ps1"
	}
	scriptFile := filepath.Join(runDir, "script"+ext)
	// the file does not need the executable bit set, it will be executed as
	// argument to powershell or /bin/sh.
	if err := os.WriteFile(scriptFile, []byte(script.ScriptContents), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write script file: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), scriptExecTimeout)
	defer cancel()

	execCmdFn := r.execCmdFn
	if execCmdFn == nil {
		execCmdFn = execCmd
	}
	start := time.Now()
	output, exitCode, execErr := execCmdFn(ctx, scriptFile)
	duration := time.Since(start)

	// report the output or the error
	if execErr != nil {
		output = append(output, []byte(fmt.Sprintf("\nscript execution error: %v", execErr))...)
	}

	// sanity-check the size of the output sent to the server, the actual
	// trimming to 10K chars is done by the API endpoint, we just make sure not
	// to send a ridiculously big payload that is sure to be over 10K chars.
	if len(output) > (utf8.UTFMax * maxOutputRuneLen) {
		output = output[len(output)-(utf8.UTFMax*maxOutputRuneLen):]
	}

	err = r.Client.SaveHostScriptResult(&fleet.HostScriptResultPayload{
		ExecutionID: execID,
		Output:      string(output),
		Runtime:     int(duration.Seconds()),
		ExitCode:    exitCode,
	})
	if err != nil {
		return fmt.Errorf("save script result: %w", err)
	}
	return nil
}

func (r *Runner) createRunDir(execID string) (string, error) {
	var tempDir string // empty tempDir means use system default
	if r.tempDirFn != nil {
		tempDir = r.tempDirFn()
	}
	// MkdirTemp will only allow read/write by current user (root), which is what
	// we want.
	return os.MkdirTemp(tempDir, "fleet-"+execID+"-*")
}

func (r *Runner) runOneDisabled(execID string) error {
	err := r.Client.SaveHostScriptResult(&fleet.HostScriptResultPayload{
		ExecutionID: execID,
		Output:      "Scripts are disabled",
		ExitCode:    -2, // fleetctl knows that -2 means script was disabled on host
	})
	if err != nil {
		return fmt.Errorf("save script result: %w", err)
	}
	return nil
}
