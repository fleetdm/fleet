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

func NewZenity() *Zenity {
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

func execCmd(ctx context.Context, args ...string) ([]byte, int, error) {
	cmd := exec.CommandContext(ctx, "zenity", args...)
	err := cmd.Run()
	if err != nil {
		return nil, cmd.ProcessState.ExitCode(), err
	}

	return nil, cmd.ProcessState.ExitCode(), nil
}
