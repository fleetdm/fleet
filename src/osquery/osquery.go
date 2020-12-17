package osquery

import (
	"context"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/fleetdm/orbit/src/process"
	"github.com/pkg/errors"
)

type Runner struct {
	proc   *process.Process
	cmd    *exec.Cmd
	cancel func()
}

func NewRunner() (*Runner, error) {
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

	return r, nil
}

func (r *Runner) Execute() error {
	if err := r.proc.Start(); err != nil {
		return errors.Wrap(err, "start osquery")
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	if err := r.proc.StopOrKill(ctx, 10*time.Second); err != nil {
		return errors.Wrap(err, "osquery exited with error")
	}

	return errors.New("osquery exited unexpectedly")
}

func (r *Runner) Interrupt(err error) {
	log.Printf("interrupt osquery")
	r.cancel()
}
