// package osquery implements a runtime for osqueryd.
package osquery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/process"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/rs/zerolog/log"
)

// Runner is a specialized runner for osquery. It is designed with Execute and
// Interrupt functions to be compatible with oklog/run.
type Runner struct {
	proc        *process.Process
	cmd         *exec.Cmd
	dataPath    string
	singleQuery bool

	ctxMu  sync.Mutex // protects the ctx and cancel
	ctx    context.Context
	cancel func()
}

type Option func(*Runner) error

// NewRunner creates a new osquery runner given the provided functional options.
func NewRunner(path string, options ...Option) (*Runner, error) {
	switch _, err := os.Stat(path); {
	case err == nil:
		// OK
	case errors.Is(err, os.ErrNotExist):
		return nil, fmt.Errorf("osqueryd doesn't exist at path %q", path)
	default:
		return nil, fmt.Errorf("failed to check for osqueryd file: %w", err)
	}

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

	// Attempt to cleanup any extension socket leftover from previous runs.
	// In some cases it's not cleaned up properly by osquery before exit.
	if err := os.Remove(r.ExtensionSocketPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error().Err(err).Msg("clean-up extension socket")
	}

	return r, nil
}

// WithFlags adds additional flags to the osqueryd invocation.
func WithFlags(flags []string) Option {
	return func(r *Runner) error {
		r.cmd.Args = append(r.cmd.Args, flags...)
		return nil
	}
}

// WithEnv adds additional environment variables to the osqueryd invocation.
// Inputs should be in the form "KEY=VAL".
func WithEnv(env []string) Option {
	return func(r *Runner) error {
		r.cmd.Env = append(r.cmd.Env, env...)
		return nil
	}
}

// SingleQuery configures the osqueryd invocation to run a SQL statement and exit.
func SingleQuery() Option {
	return func(r *Runner) error {
		r.singleQuery = true
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

// WithDataPath configures the dataPath in the *Runner and
// sets the --pidfile and --extensions_socket paths
// to the osqueryd invocation.
func WithDataPath(dataPath, extensionPathPostfix string) Option {
	return func(r *Runner) error {
		r.dataPath = dataPath

		if err := secure.MkdirAll(dataPath, constant.DefaultDirMode); err != nil {
			return fmt.Errorf("initialize osquery data path: %w", err)
		}

		r.cmd.Args = append(r.cmd.Args,
			"--pidfile="+filepath.Join(dataPath, constant.OsqueryPidfile),
			"--extensions_socket="+r.ExtensionSocketPath()+extensionPathPostfix,
		)
		return nil
	}
}

// WithStderr sets the runner's cmd's stderr to the given writer.
func WithStderr(w io.Writer) Option {
	return func(r *Runner) error {
		r.cmd.Stderr = w
		return nil
	}
}

// WithStdout sets the runner's cmd's stdout to the given writer.
func WithStdout(w io.Writer) Option {
	return func(r *Runner) error {
		r.cmd.Stdout = w
		return nil
	}
}

func WithLogPath(path string) Option {
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
	log.Info().Str("cmd", r.cmd.String()).Msg("start osqueryd")

	if r.singleQuery {
		// When running in "SQL STATEMENT" mode, start osqueryd
		// and wait for it to exit.
		if err := r.cmd.Run(); err != nil {
			return fmt.Errorf("start osqueryd shell: %w", err)
		}
	} else {
		ctx, _ := r.getContextAndCancel()

		if err := r.proc.Start(); err != nil {
			return fmt.Errorf("start osqueryd: %w", err)
		}

		if err := r.proc.WaitOrKill(ctx, 10*time.Second); err != nil {
			return fmt.Errorf("osqueryd exited with error: %w", err)
		}
	}

	return nil
}

// Runner interrupts the running osquery process.
func (r *Runner) Interrupt(err error) {
	if _, cancel := r.getContextAndCancel(); cancel != nil {
		cancel()
	}
}

func (r *Runner) ExtensionSocketPath() string {
	const (
		extensionSocketName        = "orbit-osquery.em"
		windowsExtensionSocketPath = `\\.\pipe\orbit-osquery-extension`
	)
	if runtime.GOOS == "windows" {
		return windowsExtensionSocketPath
	}
	return filepath.Join(r.dataPath, extensionSocketName)
}

func (r *Runner) getContextAndCancel() (context.Context, func()) {
	r.ctxMu.Lock()
	defer r.ctxMu.Unlock()

	if r.ctx != nil {
		return r.ctx, r.cancel
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.ctx = ctx
	r.cancel = cancel
	return r.ctx, r.cancel
}
