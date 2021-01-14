package osquery

import (
	"context"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/fleetdm/orbit/pkg/process"
	"github.com/pkg/errors"
)

type Runner struct {
	proc   *process.Process
	cmd    *exec.Cmd
	cancel func()
}

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

func WithFlags(flags []string) func(*Runner) error {
	return func(r *Runner) error {
		r.cmd.Args = append(r.cmd.Args, flags...)
		return nil
	}
}

func WithEnv(env []string) func(*Runner) error {
	return func(r *Runner) error {
		r.cmd.Env = append(r.cmd.Env, env...)
		return nil
	}
}

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

func (r *Runner) Execute() error {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	if err := r.proc.Start(); err != nil {
		return errors.Wrap(err, "start osquery")
	}

	if err := r.proc.StopOrKill(ctx, 10*time.Second); err != nil {
		return errors.Wrap(err, "osquery exited with error")
	}

	return errors.New("osquery exited unexpectedly")
}

func (r *Runner) Interrupt(err error) {
	log.Printf("interrupt osquery")
	r.cancel()
}
