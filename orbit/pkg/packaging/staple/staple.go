// Package staple staples a notarization ticket to a file, allowing it
// to be validated offline. This only works for files of type "app", "dmg",
// or "pkg".
package staple

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-hclog"
)

// Options are the options for creating the zip archive.
type Options struct {
	// File to staple. It is stapled in-place.
	File string

	// Logger is the logger to use. If this is nil then no logging will be done.
	Logger hclog.Logger

	// BaseCmd is the base command for executing the codesign binary. This is
	// used for tests to overwrite where the codesign binary is.
	BaseCmd *exec.Cmd
}

// Staple staples the notarization ticket to a file.
func Staple(ctx context.Context, opts *Options) error {
	logger := opts.Logger
	if logger == nil {
		logger = hclog.NewNullLogger()
	}

	// Build our command
	var cmd exec.Cmd
	if opts.BaseCmd != nil {
		cmd = *opts.BaseCmd
	}

	// We only set the path if it isn't set. This lets the options set the
	// path to the codesigning binary that we use.
	if cmd.Path == "" {
		path, err := exec.LookPath("xcrun")
		if err != nil {
			return err
		}
		cmd.Path = path
	}

	cmd.Args = []string{
		filepath.Base(cmd.Path),
		"stapler",
		"staple",
		opts.File,
	}

	// We store all output in out for logging and in case there is an error
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = cmd.Stdout

	// Log what we're going to execute
	logger.Info("executing stapler",
		"file", opts.File,
		"command_path", cmd.Path,
		"command_args", cmd.Args,
	)

	// Execute
	if err := cmd.Run(); err != nil {
		logger.Error("error stapling", "err", err, "output", out.String())
		return fmt.Errorf("error stapling:\n\n%s", out.String())
	}

	logger.Info("stapling complete", "file", opts.File)
	return nil
}
