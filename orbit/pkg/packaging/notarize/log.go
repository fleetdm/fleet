package notarize

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-hclog"
)

// Log Retrieves notarization log for a single completed submission
type Log struct {
	JobId           string             `json:"jobId"`
	Status          string             `json:"status"`
	StatusSummary   string             `json:"statusSummary"`
	StatusCode      int                `json:"statusCode"`
	ArchiveFilename string             `json:"archiveFilename"`
	UploadDate      string             `json:"uploadDate"`
	SHA256          string             `json:"sha256"`
	Issues          []LogIssue         `json:"issues"`
	TicketContents  []LogTicketContent `json:"ticketContents"`
}

// LogIssue is a single issue that may have occurred during notarization.
type LogIssue struct {
	Severity string `json:"severity"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

// LogTicketContent is an entry that was noted as being within the archive.
type LogTicketContent struct {
	Path            string `json:"path"`
	DigestAlgorithm string `json:"digestAlgorithm"`
	CDHash          string `json:"cdhash"`
	Arch            string `json:"arch"`
}

// log requests the information about a notarization and returns
// the updated information.
func log(ctx context.Context, uuid string, opts *Options) (*Log, error) {
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
		"log",
		uuid,
		"--apple-id", opts.DeveloperId,
		"--password", opts.Password,
		"--team-id", opts.Provider,
	}

	// We store all output in out for logging and in case there is an error
	var out, combined bytes.Buffer
	cmd.Stdout = io.MultiWriter(&out, &combined)
	cmd.Stderr = &combined

	// Log what we're going to execute
	logger.Info("requesting notarization log",
		"uuid", uuid,
		"command_path", cmd.Path,
		"command_args", cmd.Args,
	)

	// Execute
	err := cmd.Run()

	// Log the result
	logger.Info("notarization log command finished",
		"output", out.String(),
		"err", err,
	)

	// If we have any output, try to decode that since even in the case of
	// an error it will output some information.
	var result Log
	// return &result, json.NewDecoder().Decode(&result)
	if out.Len() > 0 {
		if derr := json.Unmarshal(out.Bytes(), &result); derr != nil {
			return nil, fmt.Errorf("failed to decode notarization submission output: %w", derr)

		}
	}

	// Now we check the error for actually running the process
	if err != nil {
		return nil, fmt.Errorf("error checking on notarization status:\n\n%s", combined.String())
	}

	logger.Info("notarization log", "uuid", uuid, "info", result)
	return &result, nil
}
