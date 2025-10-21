package notarize

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-hclog"
	"howett.net/plist"
)

// Info is the information structure for the state of a notarization request.
//
// All fields should be checked against their zero value since certain values
// only become available at different states of the notarization process. If
// we were only able to submit a notarization request and not check the status
// once, only RequestUUID will be set.
type Info struct {
	// RequestUUID is the UUID provided by Apple after submitting the
	// notarization request. This can be used to look up notarization information
	// using the Apple tooling.
	RequestUUID string `plist:"id"`

	// Date is the date and time of submission
	Date string `plist:"createdDate"`

	// Name is th file uploaded for submission.
	Name string `plist:"name"`

	// Status the status of the notarization.
	Status string `plist:"status"`

	// StatusMessage is a human-friendly message associated with a status.
	StatusMessage string `plist:"message"`
}

// info requests the information about a notarization and returns
// the updated information.
func info(ctx context.Context, uuid string, opts *Options) (*Info, error) {
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
			return nil, err
		}
		cmd.Path = path
	}

	cmd.Args = []string{
		filepath.Base(cmd.Path),
		"notarytool",
		"info",
		uuid,
		"--apple-id", opts.DeveloperId,
		"--password", opts.Password,
		"--team-id", opts.Provider,
		"--output-format", "plist",
	}

	// We store all output in out for logging and in case there is an error
	var out, combined bytes.Buffer
	cmd.Stdout = io.MultiWriter(&out, &combined)
	cmd.Stderr = &combined

	// Log what we're going to execute
	logger.Info("requesting notarization info",
		"uuid", uuid,
		"command_path", cmd.Path,
		"command_args", cmd.Args,
	)

	// Execute
	err := cmd.Run()

	// Log the result
	logger.Info("notarization info command finished",
		"output", out.String(),
		"err", err,
	)

	// If we have any output, try to decode that since even in the case of
	// an error it will output some information.
	var result Info
	if out.Len() > 0 {
		if _, perr := plist.Unmarshal(out.Bytes(), &result); perr != nil {
			return nil, fmt.Errorf("failed to decode notarization submission output: %w", perr)
		}
	}

	// Now we check the error for actually running the process
	if err != nil {
		return nil, fmt.Errorf("error checking on notarization status:\n\n%s", combined.String())
	}

	logger.Info("notarization info", "uuid", uuid, "info", result)
	return &result, nil
}
