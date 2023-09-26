package fleet

import (
	"bufio"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Script represents a saved script that can be executed on a host.
type Script struct {
	ID     uint   `json:"id" db:"id"`
	TeamID *uint  `json:"team_id" db:"team_id"`
	Name   string `json:"name" db:"name"`
	// ScriptContents is not returned in payloads nor is it returned
	// from reading from the database, it is only used as payload to
	// create the script. This is so that we minimize the number of
	// times this potentially large field is transferred.
	ScriptContents string    `json:"-"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	// UpdatedAt serves as the "uploaded at" timestamp, since it is updated each
	// time the script record gets updated.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func (s Script) AuthzType() string {
	return "script"
}

type HostScriptRequestPayload struct {
	HostID         uint   `json:"host_id"`
	ScriptContents string `json:"script_contents"`
}

type HostScriptResultPayload struct {
	HostID      uint   `json:"host_id"`
	ExecutionID string `json:"execution_id"`
	Output      string `json:"output"`
	Runtime     int    `json:"runtime"`
	ExitCode    int    `json:"exit_code"`
}

// HostScriptResult represents a script result that was requested to execute on
// a specific host. If no result was received yet for a script, the ExitCode
// field is null and the output is empty.
type HostScriptResult struct {
	// ID is the unique row identifier of the host script result.
	ID uint `json:"-" db:"id"`
	// HostID is the host on which the script was executed.
	HostID uint `json:"host_id" db:"host_id"`
	// ExecutionID is a unique identifier for a single execution of the script.
	ExecutionID string `json:"execution_id" db:"execution_id"`
	// ScriptContents is the content of the script to execute.
	ScriptContents string `json:"script_contents" db:"script_contents"`
	// Output is the combined stdout/stderr output of the script. It is empty
	// if no result was received yet.
	Output string `json:"output" db:"output"`
	// Runtime is the running time of the script in seconds, rounded.
	Runtime int `json:"runtime" db:"runtime"`
	// ExitCode is null if script execution result was never received from the
	// host. It is -1 if it was received but the script did not terminate
	// normally (same as how Go handles this: https://pkg.go.dev/os#ProcessState.ExitCode)
	ExitCode *int64 `json:"exit_code" db:"exit_code"`
	// CreatedAt is the creation timestamp of the script execution request. It is
	// not returned as part of the payloads, but is used to determine if the script
	// is too old to still expect a response from the host.
	CreatedAt time.Time `json:"-" db:"created_at"`

	// TeamID is only used for authorization, it must be set to the team id of
	// the host when checking authorization and is otherwise not set.
	TeamID *uint `json:"team_id" db:"-"`

	// Message is the UserMessage associated with a response from an execution.
	// It may be set by the endpoint and included in the resulting JSON but it is
	// not otherwise part of the host_script_results table.
	Message string `json:"message" db:"-"`

	// Hostname can be set by the endpoint as extra information to make available
	// when generating the UserMessage associated with a response from an
	// execution. It is otherwise not part of the host_script_results table and
	// not returned as part of the resulting JSON.
	Hostname string `json:"-" db:"-"`
}

func (hsr HostScriptResult) AuthzType() string {
	return "host_script_result"
}

// UserMessage returns the user-friendly message to associate with the current
// state of the HostScriptResult. This is returned as part of the API endpoints
// for running a script synchronously (so that fleetctl can display it) and to
// get the script results for an execution ID (e.g. when looking at the details
// screen of a script execution activity in the website).
func (hsr HostScriptResult) UserMessage(hostTimeout bool) string {
	if hostTimeout {
		return RunScriptHostTimeoutErrMsg
	}

	if hsr.ExitCode == nil {
		if hsr.HostTimeout(1 * time.Minute) {
			return RunScriptHostTimeoutErrMsg
		}
		return RunScriptAlreadyRunningErrMsg
	}

	switch *hsr.ExitCode {
	case -1:
		return "Timeout. Fleet stopped the script after 30 seconds to protect host performance."
	case -2:
		return "Scripts are disabled for this host. To run scripts, deploy a Fleet installer with scripts enabled."
	default:
		return ""
	}
}

func (hsr HostScriptResult) HostTimeout(waitForResultTime time.Duration) bool {
	return time.Now().After(hsr.CreatedAt.Add(waitForResultTime))
}

const MaxScriptRuneLen = 10000

var (
	// anchored, so that it matches to the end of the line
	scriptHashbangValidation = regexp.MustCompile(`^#!\s*/bin/sh\s*$`)
)

func ValidateHostScriptContents(s string) error {
	if s == "" {
		return errors.New("Script contents must not be empty.")
	}

	// look for the script length in bytes first, as rune counting a huge string
	// can be expensive.
	if len(s) > utf8.UTFMax*MaxScriptRuneLen {
		return errors.New("Script is too large. It's limited to 10,000 characters (approximately 125 lines).")
	}

	// now that we know that the script is at most 4*maxScriptRuneLen bytes long,
	// we can safely count the runes for a precise check.
	if utf8.RuneCountInString(s) > MaxScriptRuneLen {
		return errors.New("Script is too large. It's limited to 10,000 characters (approximately 125 lines).")
	}

	// script must be a "text file", but that's not so simple to validate, so we
	// assume that if it is valid utf8 encoding, it is a text file (binary files
	// will often have invalid utf8 byte sequences).
	if !utf8.ValidString(s) {
		return errors.New("Wrong data format. Only plain text allowed.")
	}

	if strings.HasPrefix(s, "#!") {
		// read the first line in a portable way
		s := bufio.NewScanner(strings.NewReader(s))
		// if a hashbang is present, it can only be `/bin/sh` for now
		if s.Scan() && !scriptHashbangValidation.MatchString(s.Text()) {
			return errors.New(`Interpreter not supported. Bash scripts must run in "#!/bin/sh”.`)
		}
	}

	return nil
}
