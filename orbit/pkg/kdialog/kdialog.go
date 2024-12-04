package kdialog

import (
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/execuser"
	"github.com/fleetdm/fleet/v4/orbit/pkg/platform"

	"github.com/godbus/dbus/v5"
)

const kdialogProcessName = "kdialog"

type KDialog struct {
	cmdWithOutput func(timeout time.Duration, args ...string) ([]byte, int, error)
	cmdWithCancel func(args ...string) (cancelFunc func() error, err error)
}

func New() *KDialog {
	return &KDialog{
		cmdWithOutput: execCmdWithOutput,
		cmdWithCancel: execCmdWithCancel,
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

	output, statusCode, err := k.cmdWithOutput(opts.TimeOut, args...)
	if err != nil {
		switch statusCode {
		case 1:
			return nil, dialog.ErrCanceled
		case 124:
			return nil, dialog.ErrTimeout
		default:
			return nil, errors.Join(dialog.ErrUnknown, err)
		}
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

	cancel, err := k.cmdWithCancel(args...)
	if err != nil {
		return nil, err
	}

	return cancel, nil
}

func (k *KDialog) ShowInfo(opts dialog.InfoOptions) error {
	args := []string{"--msgbox"}
	if opts.Text != "" {
		args = append(args, opts.Text)
	}
	if opts.Title != "" {
		args = append(args, "--title", opts.Title)
	}

	_, statusCode, err := k.cmdWithOutput(opts.TimeOut, args...)
	if err != nil {
		switch statusCode {
		case 124:
			return dialog.ErrTimeout
		default:
			return errors.Join(dialog.ErrUnknown, err)
		}
	}

	return nil
}

func execCmdWithOutput(timeout time.Duration, args ...string) ([]byte, int, error) {
	var opts []execuser.Option
	for _, arg := range args {
		opts = append(opts, execuser.WithArg(arg, "")) // using empty value for positional args
	}

	if timeout > 0 {
		opts = append(opts, execuser.WithTimeout(timeout))
	}

	output, exitCode, err := execuser.RunWithOutput(kdialogProcessName, opts...)
	if err != nil {
		return nil, exitCode, err
	}

	return output, exitCode, nil
}

func execCmdWithCancel(args ...string) (func() error, error) {
	var opts []execuser.Option
	for _, arg := range args {
		opts = append(opts, execuser.WithArg(arg, "")) // using empty value for positional args
	}

	_, err := execuser.Run(kdialogProcessName, opts...)
	if err != nil {
		return nil, err
	}

	killFunc := func() error {
		if _, err := platform.KillAllProcessByName(kdialogProcessName); err != nil {
			return err
		}
		return nil
	}

	return killFunc, nil
}
