// package osquery implements a runtime for osqueryd.
package osquery

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/process"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/rs/zerolog/log"
)

const (
	extensionSocketName        = "orbit-osquery.em"
	windowsExtensionSocketPath = `\\.\pipe\orbit-osquery-extension`
)

// Runner is a specialized runner for osquery. It is designed with Execute and
// Interrupt functions to be compatible with oklog/run.
type Runner struct {
	proc     *process.Process
	cmd      *exec.Cmd
	dataPath string
	cancel   func()
}

// NewRunner creates a new osquery runner given the provided functional options.
func NewRunner(path string, options ...func(*Runner) error) (*Runner, error) {
	r := &Runner{}

	cmd := exec.Command(path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	r.cmd = cmd
	r.proc = process.NewWithCmd(cmd)

	for _, option := range options {
		err := option(r)
		if err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	return r, nil
}

// WithFlags adds additional flags to the osqueryd invocation.
func WithFlags(flags []string) func(*Runner) error {
	return func(r *Runner) error {
		r.cmd.Args = append(r.cmd.Args, flags...)
		return nil
	}
}

// WithEnv adds additional environment variables to the osqueryd invocation.
// Inputs should be in the form "KEY=VAL".
func WithEnv(env []string) func(*Runner) error {
	return func(r *Runner) error {
		r.cmd.Env = append(r.cmd.Env, env...)
		return nil
	}
}

// WithShell adds the -S flag to run an osqueryi shell.
func WithShell() func(*Runner) error {
	return func(r *Runner) error {
		r.cmd.Args = append(r.cmd.Args, "-S")
		r.cmd.Stdin = os.Stdin
		return nil
	}
}

func WithDataPath(path string) func(*Runner) error {
	return func(r *Runner) error {
		r.dataPath = path

		if err := secure.MkdirAll(filepath.Join(path, "logs"), constant.DefaultDirMode); err != nil {
			return fmt.Errorf("initialize osquery data path: %w", err)
		}

		r.cmd.Args = append(r.cmd.Args,
			"--pidfile="+filepath.Join(path, "osquery.pid"),
			"--database_path="+filepath.Join(path, "osquery.db"),
			"--extensions_socket="+r.ExtensionSocketPath(),
		)
		return nil
	}
}

func WithLogPath(path string) func(*Runner) error {
	return func(r *Runner) error {
		if err := secure.MkdirAll(path, constant.DefaultDirMode); err != nil {
			return fmt.Errorf("initialize osquery log path: %w", err)
		}

		r.cmd.Args = append(r.cmd.Args,
			"--logger_path="+path,
		)

		return nil
	}
}

// Execute begins running osqueryd and returns when the process exits. The
// process may not be restarted after exit. Instead create a new one with
// NewRunner.
func (r *Runner) Execute() error {
	log.Info().Str("cmd", r.cmd.String()).Msg("run osqueryd")

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	if err := r.proc.Start(); err != nil {
		return fmt.Errorf("start osqueryd: %w", err)
	}

	if err := r.proc.WaitOrKill(ctx, 10*time.Second); err != nil {
		return fmt.Errorf("osqueryd exited with error: %w", err)
	}

	return nil
}

// Runner interrupts the running osquery process.
func (r *Runner) Interrupt(err error) {
	log.Debug().Msg("interrupt osquery")
	r.cancel()
}

func (r *Runner) ExtensionSocketPath() string {
	if runtime.GOOS == "windows" {
		return windowsExtensionSocketPath
	}

	return filepath.Join(r.dataPath, extensionSocketName)
}
