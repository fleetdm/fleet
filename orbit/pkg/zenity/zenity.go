package zenity

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/execuser"
)

const zenityProcessName = "zenity"

type Zenity struct {
	// cmdWithOutput can be set in tests to mock execution of the dialog.
	cmdWithOutput func(ctx context.Context, args ...string) ([]byte, int, error)
	// cmdWithWait can be set in tests to mock execution of the dialog.
	cmdWithCancel func(args ...string) (func() error, error)
}

// New creates a new Zenity dialog instance for zenity v4 on Linux.
// Zenity implements the Dialog interface.
func New() *Zenity {
	return &Zenity{
		cmdWithOutput: execCmdWithOutput,
		cmdWithCancel: execCmdWithCancel,
	}
}

// ShowEntry displays an dialog that accepts end user input. It returns the entered
// text or errors ErrCanceled, ErrTimeout, or ErrUnknown.
func (z *Zenity) ShowEntry(opts dialog.EntryOptions) ([]byte, error) {
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

	output, statusCode, err := z.cmdWithOutput(context.Background(), args...)
	if err != nil {
		switch statusCode {
		case 1:
			return nil, dialog.ErrCanceled
		case 5:
			return nil, dialog.ErrTimeout
		default:
			return nil, errors.Join(dialog.ErrUnknown, err)
		}
	}

	return output, nil
}

// ShowInfo displays an information dialog. It returns errors ErrTimeout or ErrUnknown.
func (z *Zenity) ShowInfo(opts dialog.InfoOptions) error {
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

	_, statusCode, err := z.cmdWithOutput(context.Background(), args...)
	if err != nil {
		switch statusCode {
		case 5:
			return dialog.ErrTimeout
		default:
			return errors.Join(dialog.ErrUnknown, err)
		}
	}

	return nil
}

// ShowProgress starts a Zenity pulsating progress dialog with the given options.
// It returns a cancel function that can be used to cancel the dialog.
func (z *Zenity) ShowProgress(opts dialog.ProgressOptions) (func() error, error) {
	args := []string{"--progress"}
	if opts.Title != "" {
		args = append(args, fmt.Sprintf("--title=%s", opts.Title))
	}
	if opts.Text != "" {
		args = append(args, fmt.Sprintf("--text=%s", opts.Text))
	}

	// --pulsate shows a pulsating progress bar
	args = append(args, "--pulsate")

	// --no-cancel disables the cancel button
	args = append(args, "--no-cancel")

	// --auto-close automatically closes the dialog when stdin is closed
	args = append(args, "--auto-close")

	cancel, err := z.cmdWithCancel(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to start progress dialog: %w", err)
	}

	return cancel, nil
}

func execCmdWithOutput(ctx context.Context, args ...string) ([]byte, int, error) {
	var opts []execuser.Option
	for _, arg := range args {
		opts = append(opts, execuser.WithArg(arg, "")) // Using empty value for positional args
	}

	output, exitCode, err := execuser.RunWithOutput(ctx, zenityProcessName, opts...)

	// Trim the newline from zenity output
	output = bytes.TrimSuffix(output, []byte("\n"))

	return output, exitCode, err
}

func execCmdWithCancel(args ...string) (func() error, error) {
	var opts []execuser.Option
	for _, arg := range args {
		opts = append(opts, execuser.WithArg(arg, "")) // Using empty value for positional args
	}

	stdin, err := execuser.RunWithStdin(zenityProcessName, opts...)
	if err != nil {
		return nil, err
	}

	return stdin.Close, err
}
