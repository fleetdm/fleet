package zenity

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

type Zenity struct {
	// execCmdFn can be set in tests to mock execution of the dialog.
	execCmdFn func(ctx context.Context, args ...string) ([]byte, int, error)
}

var (
	ErrCanceled = errors.New("dialog canceled")
	ErrTimeout  = errors.New("dialog timed out")
	ErrUnknown  = errors.New("unknown error")
)

// EntryOptions represents options for the Entry dialog.
type EntryOptions struct {
	// Title sets the title of the dialog.
	Title string

	// Text sets the text of the dialog.
	Text string

	// HideText hides the text entered by the user.
	HideText bool

	// TimeOut sets the time in seconds before the dialog is automatically closed.
	TimeOut time.Duration
}

// InfoOptions represents options for the Info dialog.
type InfoOptions struct {
	// Title sets the title of the dialog.
	Title string

	// Text sets the text of the dialog.
	Text string

	// Timeout sets the time in seconds before the dialog is automatically closed.
	TimeOut time.Duration
}

// NewZenity creates a new Zenity dialog instance for zenity v4 on Linux.
func New() *Zenity {
	return &Zenity{
		execCmdFn: execCmd,
	}
}

// ShowEntry displays an dialog that accepts end user input. It returns the entered
// text or errors ErrCanceled, ErrTimeout, or ErrUnknown.
func (z *Zenity) ShowEntry(ctx context.Context, opts EntryOptions) ([]byte, error) {
	args := []string{"--entry"}
	if opts.Title != "" {
		args = append(args, fmt.Sprintf(`--title="%s"`, opts.Title))
	}
	if opts.Text != "" {
		args = append(args, fmt.Sprintf(`--text="%s"`, opts.Text))
	}
	if opts.HideText {
		args = append(args, "--hide-text")
	}
	if opts.TimeOut > 0 {
		args = append(args, fmt.Sprintf("--timeout=%d", int(opts.TimeOut.Seconds())))
	}

	output, statusCode, err := z.execCmdFn(ctx, args...)
	if err != nil {
		switch statusCode {
		case 1:
			return nil, ctxerr.Wrap(ctx, ErrCanceled)
		case 5:
			return nil, ctxerr.Wrap(ctx, ErrTimeout)
		default:
			return nil, ctxerr.Wrap(ctx, ErrUnknown, err.Error())
		}
	}

	return output, nil
}

// ShowInfo displays an information dialog. It returns errors ErrTimeout or ErrUnknown.
func (z *Zenity) ShowInfo(ctx context.Context, opts InfoOptions) error {
	args := []string{"--info"}
	if opts.Title != "" {
		args = append(args, fmt.Sprintf(`--title="%s"`, opts.Title))
	}
	if opts.Text != "" {
		args = append(args, fmt.Sprintf(`--text="%s"`, opts.Text))
	}
	if opts.TimeOut > 0 {
		args = append(args, fmt.Sprintf("--timeout=%d", int(opts.TimeOut.Seconds())))
	}

	_, statusCode, err := z.execCmdFn(ctx, args...)
	if err != nil {
		switch statusCode {
		case 5:
			return ctxerr.Wrap(ctx, ErrTimeout)
		default:
			return ctxerr.Wrap(ctx, ErrUnknown, err.Error())
		}
	}

	return nil
}

func execCmd(ctx context.Context, args ...string) ([]byte, int, error) {
	cmd := exec.CommandContext(ctx, "zenity", args...)
	err := cmd.Run()
	if err != nil {
		var exitCode int // exit 0 ok here if zenity returns an error
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		return nil, exitCode, err
	}

	return nil, cmd.ProcessState.ExitCode(), nil
}
