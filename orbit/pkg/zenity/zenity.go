package zenity

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/execuser"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

type Zenity struct {
	// execCmdFn can be set in tests to mock execution of the dialog.
	execCmdFn func(ctx context.Context, args ...string) ([]byte, int, error)
}

// NewZenity creates a new Zenity dialog instance for zenity v4 on Linux.
// Zenity implements the Dialog interface.
func New() *Zenity {
	return &Zenity{
		execCmdFn: execCmd,
	}
}

// ShowEntry displays an dialog that accepts end user input. It returns the entered
// text or errors ErrCanceled, ErrTimeout, or ErrUnknown.
func (z *Zenity) ShowEntry(ctx context.Context, opts dialog.EntryOptions) ([]byte, error) {
	args := []string{"--entry"}
	if opts.Title != "" {
		args = append(args, fmt.Sprintf("--title=%s", opts.Title))
	}
	if opts.Text != "" {
		args = append(args, fmt.Sprintf("--text=%s", opts.Text))
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
			return nil, ctxerr.Wrap(ctx, dialog.ErrCanceled)
		case 5:
			return nil, ctxerr.Wrap(ctx, dialog.ErrTimeout)
		default:
			return nil, ctxerr.Wrap(ctx, dialog.ErrUnknown, err.Error())
		}
	}

	return output, nil
}

// ShowInfo displays an information dialog. It returns errors ErrTimeout or ErrUnknown.
func (z *Zenity) ShowInfo(ctx context.Context, opts dialog.InfoOptions) error {
	args := []string{"--info"}
	if opts.Title != "" {
		args = append(args, fmt.Sprintf("--title=%s", opts.Title))
	}
	if opts.Text != "" {
		args = append(args, fmt.Sprintf("--text=%s", opts.Text))
	}
	if opts.TimeOut > 0 {
		args = append(args, fmt.Sprintf("--timeout=%d", int(opts.TimeOut.Seconds())))
	}

	_, statusCode, err := z.execCmdFn(ctx, args...)
	if err != nil {
		switch statusCode {
		case 5:
			return ctxerr.Wrap(ctx, dialog.ErrTimeout)
		default:
			return ctxerr.Wrap(ctx, dialog.ErrUnknown, err.Error())
		}
	}

	return nil
}

func execCmd(ctx context.Context, args ...string) ([]byte, int, error) {
	var opts []execuser.Option
	for _, arg := range args {
		opts = append(opts, execuser.WithArg(arg, "")) // Using empty value for positional args
	}

	return execuser.RunWithOutput("zenity", opts...)
}
