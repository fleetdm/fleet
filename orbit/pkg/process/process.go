package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

type ExecCmd interface {
	Start() error
	Wait() error
	OsProcess() OsProcess
}

type OsProcess interface {
	Signal(os.Signal) error
	Kill() error
}

type execCmdWrapper struct {
	*exec.Cmd
}

func (e *execCmdWrapper) OsProcess() OsProcess {
	return e.Process
}

type Process struct {
	ExecCmd
}

func NewWithCmd(cmd *exec.Cmd) *Process {
	return &Process{ExecCmd: &execCmdWrapper{Cmd: cmd}}
}

func newWithMock(cmd ExecCmd) *Process {
	return &Process{ExecCmd: cmd}
}

// WaitOrKill waits for the already-started process by calling its Wait method.
//
// If the process does not return before ctx is done, WaitOrKill sends it the
// given interrupt signal. If killDelay is positive, WaitOrKill waits that
// additional period for Wait to return before sending os.Kill.
//
// Adapted from Go core:
// https://github.com/golang/go/blob/8981092d71aee273d27b0e11cf932a34d4d365c1/src/cmd/go/script_test.go#L1131-L1190
func (p *Process) WaitOrKill(ctx context.Context, killDelay time.Duration) error {
	if p.OsProcess() == nil {
		return fmt.Errorf("WaitOrKill requires a non-nil OsProcess - missing Start call?")
	}

	errc := make(chan error)
	go func() {
		select {
		case errc <- nil:
			return
		case <-ctx.Done():
		}

		err := p.OsProcess().Signal(stopSignal())
		if err == nil {
			err = ctx.Err() // Report ctx.Err() as the reason we interrupted.
		} else if err.Error() == "os: process already finished" {
			errc <- nil
			return
		}

		if killDelay > 0 {
			timer := time.NewTimer(killDelay)
			select {
			// Report ctx.Err() as the reason we interrupted the process...
			case errc <- ctx.Err():
				timer.Stop()
				return
			// ...but after killDelay has elapsed, fall back to a stronger signal.
			case <-timer.C:
			}

			// Wait still hasn't returned.
			// Kill the process harder to make sure that it exits.
			//
			// Ignore any error: if cmd.Process has already terminated, we still
			// want to send ctx.Err() (or the error from the Interrupt call)
			// to properly attribute the signal that may have terminated it.
			_ = p.OsProcess().Kill()
		}

		errc <- err
	}()

	waitErr := p.Wait()
	if interruptErr := <-errc; interruptErr != nil {
		return interruptErr
	}
	return waitErr
}

// stopSignal returns the appropriate signal to use to request that a process
// stop execution.
//
// Copied from Go core:
// https://github.com/golang/go/blob/8981092d71aee273d27b0e11cf932a34d4d365c1/src/cmd/go/script_test.go#L1119-L1129
func stopSignal() os.Signal {
	if runtime.GOOS == "windows" {
		// Per https://golang.org/pkg/os/#Signal, “Interrupt is not implemented on
		// Windows; using it with os.Process.Signal will return an error.”
		// Fall back to Kill instead.
		return os.Kill
	}
	return os.Interrupt
}
