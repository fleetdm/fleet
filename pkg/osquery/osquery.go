// package osquery implements a runtime for osqueryd.
package osquery

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/fleetdm/orbit/pkg/process"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Runner is a specialized runner for osquery. It is designed with Execute and
// Interrupt functions to be compatible with oklog/run.
type Runner struct {
	proc   *process.Process
	cmd    *exec.Cmd
	cancel func()
}

// NewRunner creates a new osquery runner given the provided functional options.
func NewRunner(options ...func(*Runner) error) (*Runner, error) {
	r := &Runner{}

	// TODO set path and flags appropriately
	cmd := exec.Command(
		"osqueryd",
		"--pidfile=/tmp/osquery.pid",
		"--database_path=/tmp/osquery.test.db",
		"--extensions_socket=/tmp/osquery.em",
		"--config_path=/tmp/osquery.conf",
		"--logger_path=/tmp",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	r.cmd = cmd
	r.proc = process.NewWithCmd(cmd)

	for _, option := range options {
		err := option(r)
		if err != nil {
			return nil, errors.Wrap(err, "apply option")
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

// WithPath sets the path of the osqueryd binary to execute.
func WithPath(path string) func(*Runner) error {
	return func(r *Runner) error {
		r.cmd.Path = path
		return nil
	}
}

// WithShell adds the -S flag to run an osqueryi shell.
func WithShell() func(*Runner) error {
	return func(r *Runner) error {
		r.cmd.Args = append(r.cmd.Args, "-S")
		r.cmd.Stdout = os.Stdout
		r.cmd.Stderr = os.Stderr
		r.cmd.Stdin = os.Stdin
		return nil
	}
}

// Execute begins running osqueryd and returns when the process exits. The
// process may not be restarted after exit. Instead create a new one with
// NewRunner.
func (r *Runner) Execute() error {
	log.Debug().Str("cmd", r.cmd.String()).Msg("run osqueryd")

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	if err := r.proc.Start(); err != nil {
		return errors.Wrap(err, "start osqueryd")
	}

	if err := r.proc.WaitOrKill(ctx, 10*time.Second); err != nil {
		return errors.Wrap(err, "osqueryd exited with error")
	}

	return nil
}

// Runner interrupts the running osquery process.
func (r *Runner) Interrupt(err error) {
	log.Debug().Msg("interrupt osquery")
	r.cancel()
}
