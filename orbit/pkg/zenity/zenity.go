package zenity

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

type Zenity struct {
	// CommandContext defined here for testing purposes.
	CommandContext func(ctx context.Context, name string, arg ...string) *exec.Cmd
}

type ExitStatus int

const (
	Success ExitStatus = iota
	Canceled
	Timeout
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
		CommandContext: exec.CommandContext,
	}
}

// ShowEntry displays an entry dialog.
func (z *Zenity) ShowEntry(ctx context.Context, opts EntryOptions) {
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
	z.CommandContext(ctx, "zenity", args...)
}
