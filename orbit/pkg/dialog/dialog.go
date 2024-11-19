package dialog

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrCanceled is returned when the dialog is canceled by the cancel button.
	ErrCanceled = errors.New("dialog canceled")
	// ErrTimeout is returned when the dialog is automatically closed due to a timeout.
	ErrTimeout = errors.New("dialog timed out")
	// ErrUnknown is returned when an unknown error occurs.
	ErrUnknown = errors.New("unknown error")
)

// Dialog represents a UI dialog that can be displayed to the end user
// on a host
type Dialog interface {
	// ShowEntry displays a dialog that accepts end user input. It returns the entered
	// text or errors ErrCanceled, ErrTimeout, or ErrUnknown.
	ShowEntry(ctx context.Context, opts EntryOptions) ([]byte, error)
	// ShowInfo displays a dialog that displays information. It returns an error if the dialog
	// could not be displayed.
	ShowInfo(ctx context.Context, opts InfoOptions) error
	// Progress displays a dialog that shows progress. It waits until the
	// context is cancelled.
	ShowProgress(ctx context.Context, opts ProgressOptions) error
}

// EntryOptions represents options for a dialog that accepts end user input.
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

// InfoOptions represents options for a dialog that displays information.
type InfoOptions struct {
	// Title sets the title of the dialog.
	Title string

	// Text sets the text of the dialog.
	Text string

	// Timeout sets the time in seconds before the dialog is automatically closed.
	TimeOut time.Duration
}

// ProgressOptions represents options for a dialog that shows progress.
type ProgressOptions struct {
	// Title sets the title of the dialog.
	Title string

	// Text sets the text of the dialog.
	Text string
}
