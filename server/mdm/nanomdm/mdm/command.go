package mdm

import (
	"errors"

	"github.com/groob/plist"
)

var (
	ErrInvalidCommandResult = errors.New("invalid command result")
	ErrInvalidCommand       = errors.New("invalid command")
)

// ErrorChain represents errors that occured on the client executing an MDM command.
type ErrorChain struct {
	ErrorCode            int
	ErrorDomain          string
	LocalizedDescription string
	USEnglishDescription string
}

// CommandResults represents a 'command and report results' request.
// See https://developer.apple.com/documentation/devicemanagement/implementing_device_management/sending_mdm_commands_to_a_device
type CommandResults struct {
	Enrollment
	CommandUUID string `plist:",omitempty"`
	Status      string
	ErrorChain  []ErrorChain `plist:",omitempty"`
	Raw         []byte       `plist:"-"` // Original command result XML plist
}

// DecodeCheckin unmarshals rawMessage into results
func DecodeCommandResults(rawResults []byte) (results *CommandResults, err error) {
	results = new(CommandResults)
	err = plist.Unmarshal(rawResults, results)
	if err != nil {
		return
	}
	results.Raw = rawResults
	if results.Status == "" {
		err = ErrInvalidCommandResult
	}
	return
}

// Command represents a generic MDM command without command-specific fields.
type Command struct {
	CommandUUID string
	Command     struct {
		RequestType string
	}
	Raw []byte `plist:"-"` // Original command XML plist
}

// DecodeCommand unmarshals rawCommand into command
func DecodeCommand(rawCommand []byte) (command *Command, err error) {
	command = new(Command)
	err = plist.Unmarshal(rawCommand, command)
	if err != nil {
		return
	}
	command.Raw = rawCommand
	if command.CommandUUID == "" || command.Command.RequestType == "" {
		err = ErrInvalidCommand
	}
	return
}
