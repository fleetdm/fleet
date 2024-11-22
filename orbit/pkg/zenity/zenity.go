package zenity

import (
	"bytes"
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/execuser"
	"github.com/fleetdm/fleet/v4/orbit/pkg/platform"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/rs/zerolog/log"
)

const zenityProcessName = "zenity"

type Zenity struct {
	// cmdWithOutput can be set in tests to mock execution of the dialog.
	cmdWithOutput func(ctx context.Context, args ...string) ([]byte, int, error)
	// cmdWithWait can be set in tests to mock execution of the dialog.
	cmdWithWait func(ctx context.Context, args ...string) error
	// killZenityFunc can be set in tests to mock killing the zenity process.
	killZenityFunc func()
}

// New creates a new Zenity dialog instance for zenity v4 on Linux.
// Zenity implements the Dialog interface.
func New() *Zenity {
	return &Zenity{
		cmdWithOutput:  execCmdWithOutput,
		cmdWithWait:    execCmdWithWait,
		killZenityFunc: killZenityProcesses,
	}
}

// ShowEntry displays an dialog that accepts end user input. It returns the entered
// text or errors ErrCanceled, ErrTimeout, or ErrUnknown.
func (z *Zenity) ShowEntry(ctx context.Context, opts dialog.EntryOptions) ([]byte, error) {
	z.killZenityFunc()

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

	output, statusCode, err := z.cmdWithOutput(ctx, args...)
	if err != nil {
		switch statusCode {
		case 1:
			return nil, dialog.ErrCanceled
		case 5:
			return nil, dialog.ErrTimeout
		default:
			return nil, ctxerr.Wrap(ctx, dialog.ErrUnknown, err.Error())
		}
	}

	return output, nil
}

// ShowInfo displays an information dialog. It returns errors ErrTimeout or ErrUnknown.
func (z *Zenity) ShowInfo(ctx context.Context, opts dialog.InfoOptions) error {
	z.killZenityFunc()

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

	_, statusCode, err := z.cmdWithOutput(ctx, args...)
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

// ShowProgress starts a Zenity progress dialog with the given options.
// This function is designed to block until the provided context is canceled.
// It is intended to be used within a separate goroutine to avoid blocking
// the main execution flow.
//
// If the context is already canceled, the function will return immediately.
//
// Use this function for cases where a progress dialog is needed to run
// alongside other operations, with explicit cancellation or termination.
func (z *Zenity) ShowProgress(ctx context.Context, opts dialog.ProgressOptions) error {
	z.killZenityFunc()

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

	err := z.cmdWithWait(ctx, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, dialog.ErrUnknown, err.Error())
	}

	return nil
}

func execCmdWithOutput(ctx context.Context, args ...string) ([]byte, int, error) {
	var opts []execuser.Option
	for _, arg := range args {
		opts = append(opts, execuser.WithArg(arg, "")) // Using empty value for positional args
	}

	output, exitCode, err := execuser.RunWithOutput(zenityProcessName, opts...)

	// Trim the newline from zenity output
	output = bytes.TrimSuffix(output, []byte("\n"))

	return output, exitCode, err
}

func execCmdWithWait(ctx context.Context, args ...string) error {
	var opts []execuser.Option
	for _, arg := range args {
		opts = append(opts, execuser.WithArg(arg, "")) // Using empty value for positional args
	}

	_, err := execuser.Run(zenityProcessName, opts...)
	return err
}

func killZenityProcesses() {
	_, err := platform.KillAllProcessByName(zenityProcessName)
	if err != nil {
		log.Warn().Err(err).Msg("failed to kill zenity process")
	}
}
