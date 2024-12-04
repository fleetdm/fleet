package kdialog

import (
	"context"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/execuser"

	"github.com/godbus/dbus/v5"
)

const kdialogProcessName = "kdialog"

type KDialog struct {
	cmdWithOutput  func(ctx context.Context, args ...string) ([]byte, int, error)
	cmdWithContext func(ctx context.Context, args ...string) error
}

func New() *KDialog {
	return &KDialog{
		cmdWithOutput:  cmdWithOutput,
		cmdWithContext: cmdWithContext,
	}
}

func (k *KDialog) ShowEntry(opts dialog.EntryOptions) ([]byte, error) {
	args := []string{"--password"}
	if opts.Text != "" {
		args = append(args, opts.Text)
	}
	if opts.Title != "" {
		args = append(args, "--title", opts.Title)
	}

	output, _, err := k.cmdWithOutput(context.Background(), args...)
	if err != nil {
		return nil, errors.Join(dialog.ErrUnknown, err)
	}

	return output, nil
}

type ProgressBar struct {
	serviceName string
	objectPath  dbus.ObjectPath
	conn        *dbus.Conn
}

// Update sets the progress value of the progress bar.
func (p *ProgressBar) Update(value int) error {
	obj := p.conn.Object(p.serviceName, p.objectPath)
	call := obj.Call("org.freedesktop.DBus.Properties.Set", 0,
		"org.kde.kdialog.ProgressDialog", "value", dbus.MakeVariant(value))
	if call.Err != nil {
		return fmt.Errorf("error updating progress: %w", call.Err)
	}
	return nil
}

// Close closes the progress bar.
func (p *ProgressBar) Close() error {
	obj := p.conn.Object(p.serviceName, p.objectPath)
	call := obj.Call("org.kde.kdialog.ProgressDialog.close", 0)
	if call.Err != nil {
		return fmt.Errorf("error closing progress bar: %w", call.Err)
	}
	return p.conn.Close()
}

func (k *KDialog) ShowProgress(opts dialog.ProgressOptions) (func() error, error) {
	args := []string{"--msgbox"}
	if opts.Text != "" {
		args = append(args, opts.Text)
	}
	if opts.Title != "" {
		args = append(args, "--title", opts.Title)
	}

	ctx, cancel := context.WithCancel(context.Background())

	err := k.cmdWithContext(ctx, args...)
	if err != nil {
		cancel()
		return nil, errors.Join(dialog.ErrUnknown, err)
	}

	return func() error {
		cancel()
		return nil
	}, nil
}

func (k *KDialog) ShowInfo(opts dialog.InfoOptions) error {
	args := []string{"--msgbox"}
	if opts.Text != "" {
		args = append(args, opts.Text)
	}
	if opts.Title != "" {
		args = append(args, "--title", opts.Title)
	}

	// Create a context with timeout but do not block on it
	ctx, cancel := context.WithTimeout(context.Background(), opts.TimeOut)

	// Start the command
	err := k.cmdWithContext(ctx, args...)
	if err != nil {
		cancel() // Clean up the context if command fails immediately
		return errors.Join(dialog.ErrUnknown, err)
	}

	// Run a goroutine to ensure the process is killed after timeout
	go func() {
		<-ctx.Done() // Wait for timeout or explicit cancellation
		cancel()     // Clean up context resources
	}()

	return nil
}

func cmdWithOutput(ctx context.Context, args ...string) ([]byte, int, error) {
	var opts []execuser.Option
	for _, arg := range args {
		opts = append(opts, execuser.WithArg(arg, "")) // using empty value for positional args
	}

	output, exitCode, err := execuser.RunWithOutput(ctx, kdialogProcessName, opts...)
	if err != nil {
		return nil, exitCode, err
	}

	return output, exitCode, nil
}

func cmdWithContext(ctx context.Context, args ...string) error {
	var opts []execuser.Option
	for _, arg := range args {
		opts = append(opts, execuser.WithArg(arg, "")) // using empty value for positional args
	}

	return execuser.RunWithContext(ctx, kdialogProcessName, opts...)
}
