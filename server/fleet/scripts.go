package fleet

import (
	"bufio"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/pkg/scripts"
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
	// ScriptContentID is the ID of the script contents, which are stored separately from the Script.
	ScriptContentID uint `json:"-" db:"script_content_id"`
}

func (s Script) AuthzType() string {
	return "script"
}

func (s *Script) ValidateNewScript() error {
	if s.Name == "" {
		return errors.New("The file name must not be empty.")
	}
	if filepath.Ext(s.Name) != ".sh" && filepath.Ext(s.Name) != ".ps1" {
		return errors.New("File type not supported. Only .sh and .ps1 file type is allowed.")
	}

	// validate the script contents as if it were already a saved script
	if err := ValidateHostScriptContents(s.ScriptContents, true); err != nil {
		return err
	}

	return nil
}

// HostScriptDetail represents the details of a script that applies to a specific host.
type HostScriptDetail struct {
	// HostID is the ID of the host.
	HostID uint `json:"-"`
	// ScriptID is the ID of the script.
	ScriptID uint `json:"script_id"`
	// Name is the name of the script.
	Name string `json:"name"`
	// LastExecution is the most recent execution of the script on the host. It is nil if the script
	// has never executed on the host.
	LastExecution *HostScriptExecution `json:"last_execution"`
}

// NewHostScriptDetail creates a new HostScriptDetail and sets its LastExecution field based on the
// provided details.
func NewHostScriptDetail(hostID, scriptID uint, name string, executionID *string, executedAt *time.Time, exitCode *int64, hsrID *uint) *HostScriptDetail {
	hs := HostScriptDetail{
		HostID:   hostID,
		ScriptID: scriptID,
		Name:     name,
	}
	hs.setLastExecution(executionID, executedAt, exitCode, hsrID)
	return &hs
}

// HostScriptExecution represents a single execution of a script on a host.
type HostScriptExecution struct {
	// HostID is the ID of the host.
	HostID uint `json:"-"`
	// ScriptID is the ID of the script.
	ScriptID uint `json:"-"`
	// HSRID is the unique row identifier of the host_script_results table for this execution.
	HSRID uint `json:"-"`
	// ExecutionID is a unique identifier for a single execution of the script.
	ExecutionID string `json:"execution_id"`
	// ExecutedAt represents the time that the script was executed on the host. It should correspond to
	// the created_at field of the host_script_results table for the associated HSRID.
	ExecutedAt time.Time `json:"executed_at"`
	// Status is the status of the script execution. It is one of "pending", "ran", or "error". It
	// is derived from the exit_code field of the host_script_results table for the associated HSRID.
	Status string `json:"status"`
}

// SetLastExecution updates the LastExecution field of the HostScriptDetail if the provided details
// are more recent than the current LastExecution. It returns true if the LastExecution was updated.
func (hs *HostScriptDetail) setLastExecution(executionID *string, executedAt *time.Time, exitCode *int64, hsrID *uint) bool {
	if hsrID == nil || executionID == nil || executedAt == nil {
		// no new execution, nothing to do
		return false
	}

	newHSE := &HostScriptExecution{
		HSRID:       *hsrID,
		ExecutionID: *executionID,
		ExecutedAt:  *executedAt,
	}
	switch {
	case exitCode == nil:
		newHSE.Status = "pending"
	case *exitCode == 0:
		newHSE.Status = "ran"
	default:
		newHSE.Status = "error"
	}

	if hs.LastExecution == nil {
		// no previous execution, use the new one
		hs.LastExecution = newHSE
		return true
	}
	if newHSE.ExecutedAt.After(hs.LastExecution.ExecutedAt) {
		// new execution is more recent, use it
		hs.LastExecution = newHSE
		return true
	}
	if newHSE.ExecutedAt == hs.LastExecution.ExecutedAt && newHSE.HSRID > hs.LastExecution.HSRID {
		// same execution time, but new execution has a higher ID, use it
		hs.LastExecution = newHSE
		return true
	}

	return false
}

type HostScriptRequestPayload struct {
	HostID          uint   `json:"host_id"`
	ScriptID        *uint  `json:"script_id"`
	PolicyID        *uint  `json:"policy_id"`
	ScriptContents  string `json:"script_contents"`
	ScriptContentID uint   `json:"-"`
	ScriptName      string `json:"script_name"`
	TeamID          uint   `json:"team_id,omitempty"`
	// UserID is filled automatically from the context's user (the authenticated
	// user that made the API request).
	UserID *uint `json:"-"`
	// SyncRequest is filled automatically based on the endpoint used to create
	// the execution request (synchronous or asynchronous).
	SyncRequest bool `json:"-"`
	// SetupExperienceScriptID is the ID of the setup experience script related to this request
	// payload, if such a script exists.
	SetupExperienceScriptID *uint `json:"-"`
}

func (r HostScriptRequestPayload) ValidateParams(waitForResult time.Duration) error {
	if r.ScriptContents == "" && r.ScriptID == nil && r.ScriptName == "" {
		return NewInvalidArgumentError("script", `One of 'script_id', 'script_contents', or 'script_name' is required.`)
	}

	if r.ScriptID != nil {
		switch {
		case r.ScriptContents != "":
			return NewInvalidArgumentError("script_id", `Only one of 'script_id' or 'script_contents' is allowed.`)
		case r.ScriptName != "":
			return NewInvalidArgumentError("script_id", `Only one of 'script_id' or 'script_name' is allowed.`)
		case r.TeamID > 0:
			return NewInvalidArgumentError("script_id", `Only one of 'script_id' or 'team_id' is allowed.`)
		}
	}
	if r.ScriptContents != "" {
		switch {
		case r.ScriptName != "":
			return NewInvalidArgumentError("script_contents", `Only one of 'script_contents' or 'script_name' is allowed.`)
		case r.TeamID > 0:
			return NewInvalidArgumentError("script_contents", `"Only one of 'script_contents' or 'team_id' is allowed.`)
		}
	}

	return nil
}

type HostScriptResultPayload struct {
	HostID      uint   `json:"host_id"`
	ExecutionID string `json:"execution_id"`
	Output      string `json:"output"`
	Runtime     int    `json:"runtime"`
	ExitCode    int    `json:"exit_code"`
	Timeout     int    `json:"timeout"`
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
	// Timeout is the maximum time in seconds that the script was allowed to run
	// at the time of execution.
	Timeout *int `json:"timeout" db:"timeout"`
	// CreatedAt is the creation timestamp of the script execution request. It is
	// not returned as part of the payloads, but is used to determine if the script
	// is too old to still expect a response from the host.
	CreatedAt time.Time `json:"-" db:"created_at"`
	// ScriptID is the id of the saved script to execute, or nil if this was an
	// anonymous script execution.
	ScriptID *uint `json:"script_id" db:"script_id"`
	// PolicyID is the id of the policy that triggered the script execution, or
	// nil if the execution was not triggered by a policy failure
	PolicyID *uint `json:"policy_id" db:"policy_id"`
	// UserID is the id of the user that requested execution. It is not part of
	// the rendered JSON as it is only returned by the
	// /hosts/:id/activities/upcoming endpoint which doesn't use this struct as
	// return type.
	UserID *uint `json:"-" db:"user_id"`
	// SyncRequest is used to determine when creating the script ran activity if
	// the request was synchronous or asynchronous. It is otherwise not returned
	// as part of any API endpoint.
	SyncRequest bool `json:"-" db:"sync_request"`

	// TeamID is only used for authorization, it must be set to the team id of
	// the host when checking authorization and is otherwise not set.
	TeamID *uint `json:"team_id" db:"-"` // TODO: should we omit this from the json result?

	// Message is the UserMessage associated with a response from an execution.
	// It may be set by the endpoint and included in the resulting JSON but it is
	// not otherwise part of the host_script_results table.
	Message string `json:"message" db:"-"`

	// Hostname can be set by the endpoint as extra information to make available
	// when generating the UserMessage associated with a response from an
	// execution. It is otherwise not part of the host_script_results table and
	// not returned as part of the resulting JSON.
	Hostname string `json:"-" db:"-"`

	// HostDeletedAt indicates if the results are associated with a deleted host.
	// This supports the soft-delete feature for script results so that the
	// results can still be returned to see activity details after the host got
	// deleted.
	HostDeletedAt *time.Time `json:"-" db:"host_deleted_at"`

	// SetupExperienceScriptID is the ID of the setup experience script, if this script execution
	// was part of setup experience.
	SetupExperienceScriptID *uint `json:"-" db:"setup_experience_script_id"`
}

func (hsr HostScriptResult) AuthzType() string {
	return "host_script_result"
}

// UserMessage returns the user-friendly message to associate with the current
// state of the HostScriptResult. This is returned as part of the API endpoints
// for running a script synchronously (so that fleetctl can display it) and to
// get the script results for an execution ID (e.g. when looking at the details
// screen of a script execution activity in the website).
func (hsr HostScriptResult) UserMessage(hostTimeout bool, hostTimeoutValue *int) string {
	if hostTimeout {
		return RunScriptHostTimeoutErrMsg
	}

	if hsr.ExitCode == nil {
		if hsr.HostTimeout(scripts.MaxServerWaitTime) {
			return RunScriptHostTimeoutErrMsg
		}

		if !hsr.SyncRequest {
			return RunScriptAsyncScriptEnqueuedMsg
		}

		return RunScriptAlreadyRunningErrMsg
	}

	switch *hsr.ExitCode {
	case -1:
		return HostScriptTimeoutMessage(hostTimeoutValue)
	case -2:
		return RunScriptDisabledErrMsg
	default:
		return ""
	}
}

func HostScriptTimeoutMessage(seconds *int) string {
	var timeout int
	if seconds == nil || *seconds == 0 {
		timeout = int(scripts.MaxHostExecutionTime.Seconds())
	} else {
		timeout = *seconds
	}
	return fmt.Sprintf("Timeout. Fleet stopped the script after %d seconds to protect host performance.", timeout)
}

func (hsr HostScriptResult) HostTimeout(waitForResultTime time.Duration) bool {
	return hsr.SyncRequest && hsr.ExitCode == nil && time.Now().After(hsr.CreatedAt.Add(waitForResultTime))
}

const (
	SavedScriptMaxRuneLen   = 500000
	UnsavedScriptMaxRuneLen = 10000
)

// anchored, so that it matches to the end of the line
var (
	scriptHashbangValidation  = regexp.MustCompile(`^#!\s*(:?/usr)?/bin/z?sh(?:\s*|\s+.*)$`)
	ErrUnsupportedInterpreter = errors.New(`Interpreter not supported. Shell scripts must run in "#!/bin/sh" or "#!/bin/zsh."`)
)

// ValidateShebang validates if we support a script, and whether we
// can execute it directly, or need to pass it to a shell interpreter.
func ValidateShebang(s string) (directExecute bool, err error) {
	if strings.HasPrefix(s, "#!") {
		// read the first line in a portable way
		s := bufio.NewScanner(strings.NewReader(s))
		// if a hashbang is present, it can only be `/bin/sh` or `(/usr)/bin/zsh` for now
		if s.Scan() && !scriptHashbangValidation.MatchString(s.Text()) {
			return false, ErrUnsupportedInterpreter
		}
		return true, nil
	}
	return false, nil
}

func ValidateHostScriptContents(s string, isSavedScript bool) error {
	if s == "" {
		return errors.New("Script contents must not be empty.")
	}

	maxLen := SavedScriptMaxRuneLen
	maxLenErrMsg := RunScripSavedMaxLenErrMsg
	if !isSavedScript {
		maxLen = UnsavedScriptMaxRuneLen
		maxLenErrMsg = RunScripUnsavedMaxLenErrMsg
	}

	// look for the script length in bytes first, as rune counting a huge string
	// can be expensive.
	if len(s) > utf8.UTFMax*maxLen {
		return errors.New(maxLenErrMsg)
	}

	// now that we know that the script is at most 4*maxScriptRuneLen bytes long,
	// we can safely count the runes for a precise check.
	if utf8.RuneCountInString(s) > maxLen {
		return errors.New(maxLenErrMsg)
	}

	// script must be a "text file", but that's not so simple to validate, so we
	// assume that if it is valid utf8 encoding, it is a text file (binary files
	// will often have invalid utf8 byte sequences).
	if !utf8.ValidString(s) {
		return errors.New("Wrong data format. Only plain text allowed.")
	}

	if _, err := ValidateShebang(s); err != nil {
		return err
	}

	return nil
}

type ScriptPayload struct {
	Name           string `json:"name"`
	ScriptContents []byte `json:"script_contents"`
}

type SoftwareInstallerPayload struct {
	URL                string   `json:"url"`
	PreInstallQuery    string   `json:"pre_install_query"`
	InstallScript      string   `json:"install_script"`
	UninstallScript    string   `json:"uninstall_script"`
	PostInstallScript  string   `json:"post_install_script"`
	SelfService        bool     `json:"self_service"`
	FleetMaintained    bool     `json:"-"`
	Filename           string   `json:"-"`
	InstallDuringSetup *bool    `json:"install_during_setup"` // if nil, do not change saved value, otherwise set it
	LabelsIncludeAny   []string `json:"labels_include_any"`
	LabelsExcludeAny   []string `json:"labels_exclude_any"`

	// Those resolved fields are only filled once the string-version of the
	// labels have been validated in the batch-set endpoint and known to exist.
	ResolvedLabelsIncludeAny []*SoftwareScopeLabel `json:"-"`
	ResolvedLabelsExcludeAny []*SoftwareScopeLabel `json:"-"`
}

type HostLockWipeStatus struct {
	// HostFleetPlatform is the fleet-normalized platform of the host, i.e. the
	// result of host.FleetPlatform().
	HostFleetPlatform string

	// macOS hosts use an MDM command to lock
	LockMDMCommand       *MDMCommand
	LockMDMCommandResult *MDMCommandResult

	// windows and linux hosts use a script to lock
	LockScript *HostScriptResult

	// macOS hosts must manually unlock using a secret PIN, which is stored here
	// when the lock request is sent.
	UnlockPIN string
	// macOS records the timestamp of the unlock request in the "unlock_ref",
	// which is then stored here.
	UnlockRequestedAt time.Time
	// windows and linux hosts use a script to unlock
	UnlockScript *HostScriptResult

	// macOS and Windows use MDM commands for Wipe
	WipeMDMCommand       *MDMCommand
	WipeMDMCommandResult *MDMCommandResult

	// Linux uses a script for Wipe
	WipeScript *HostScriptResult
}

// ScriptResponse is the response type used when applying scripts by batch.
type ScriptResponse struct {
	// TeamID is the id of the team.
	// A value of nil means it is scoped to hosts that are assigned to "No team".
	TeamID *uint `json:"team_id" db:"team_id"`
	// ID is the id of the script
	ID uint `json:"id" db:"id"`
	// Name is the name of the script
	Name string `json:"name" db:"name"`
}

func (s *HostLockWipeStatus) IsPendingLock() bool {
	if s.HostFleetPlatform == "darwin" || s.HostFleetPlatform == "ios" || s.HostFleetPlatform == "ipados" {
		// pending lock if an MDM command is queued but no result received yet
		return s.LockMDMCommand != nil && s.LockMDMCommandResult == nil
	}
	// pending lock if script execution request is queued but no result yet
	return s.LockScript != nil && s.LockScript.ExitCode == nil
}

func (s HostLockWipeStatus) IsPendingUnlock() bool {
	if s.HostFleetPlatform == "darwin" || s.HostFleetPlatform == "ios" || s.HostFleetPlatform == "ipados" {
		// Apple MDM does not have a concept of pending unlock.
		return false
	}
	// pending unlock if script execution request is queued but no result yet
	return s.UnlockScript != nil && s.UnlockScript.ExitCode == nil
}

func (s HostLockWipeStatus) IsPendingWipe() bool {
	if s.HostFleetPlatform == "linux" {
		// pending wipe if script execution request is queued but no result yet
		return s.WipeScript != nil && s.WipeScript.ExitCode == nil
	}
	// pending wipe if an MDM command is queued but no result received yet
	return s.WipeMDMCommand != nil && s.WipeMDMCommandResult == nil
}

func (s HostLockWipeStatus) IsLocked() bool {
	// this state is regardless of pending unlock/wipe (it reports whether the
	// host is locked *now*).

	if s.HostFleetPlatform == "darwin" || s.HostFleetPlatform == "ios" || s.HostFleetPlatform == "ipados" {
		// locked if an MDM command was sent and succeeded
		return s.LockMDMCommand != nil && s.LockMDMCommandResult != nil &&
			s.LockMDMCommandResult.Status == MDMAppleStatusAcknowledged
	}
	// locked if a script was sent and succeeded
	return s.LockScript != nil && s.LockScript.ExitCode != nil &&
		*s.LockScript.ExitCode == 0
}

func (s HostLockWipeStatus) IsUnlocked() bool {
	// this state is regardless of pending lock/unlock/wipe (it reports whether
	// the host is unlocked *now*).
	return !s.IsLocked() && !s.IsWiped()
}

func (s HostLockWipeStatus) IsWiped() bool {
	switch s.HostFleetPlatform {
	case "linux":
		// wiped if script was sent and succeeded
		return s.WipeScript != nil && s.WipeScript.ExitCode != nil &&
			*s.WipeScript.ExitCode == 0
	case "windows":
		// wiped if an MDM command was sent and succeeded
		return s.WipeMDMCommand != nil && s.WipeMDMCommandResult != nil &&
			strings.HasPrefix(s.WipeMDMCommandResult.Status, "2")
	case "darwin", "ios", "ipados":
		// wiped if an MDM command was sent and succeeded
		return s.WipeMDMCommand != nil && s.WipeMDMCommandResult != nil &&
			s.WipeMDMCommandResult.Status == MDMAppleStatusAcknowledged
	default:
		return false
	}
}
