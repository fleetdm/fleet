package swiftdialog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
)

// SwiftDialog really wants the command file to be mode 666 for some reason
var CommandFilePerms = fs.FileMode(0666)

type SwiftDialog struct {
	cancel      context.CancelFunc
	cmd         *exec.Cmd
	commandFile *os.File
	context     context.Context
	output      *bytes.Buffer
}

// Exit codes
const (
	ExitButton1               = 0
	ExitButton2               = 2
	ExitInfoButton            = 3
	ExitTimer                 = 4
	ExitQuitCommand           = 5
	ExitQuitKey               = 10
	ExitKeyAuthFailed         = 30
	ExitImageResourceNotFound = 201
	ExitFileNotFound          = 202
)

func Run(ctx context.Context, swiftDialogBin string, options *SwiftDialogOptions) (*SwiftDialog, error) {
	commandFile, err := os.CreateTemp("", "swiftDialogCommand")
	if err != nil {
		return nil, err
	}

	jsonBytes, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	if err := commandFile.Chmod(CommandFilePerms); err != nil {
		commandFile.Close()
		os.Remove(commandFile.Name())
		cancel()
		return nil, err
	}

	cmd := exec.CommandContext(
		ctx,
		swiftDialogBin,
		"--jsonstring", string(jsonBytes),
		"--commandfile", commandFile.Name(),
		"--json",
	)

	outBuf := &bytes.Buffer{}
	cmd.Stdout = outBuf

	err = cmd.Start()
	if err != nil {
		cancel()
		return nil, err
	}

	return &SwiftDialog{
		cancel:      cancel,
		cmd:         cmd,
		commandFile: commandFile,
		context:     ctx,
		output:      outBuf,
	}, nil
}

func (s *SwiftDialog) Close() error {
	s.cancel()
	if err := s.cmd.Wait(); err != nil {
		return fmt.Errorf("waiting for swiftDialog: %w", err)
	}
	if err := s.cleanup(); err != nil {
		return fmt.Errorf("Close cleaning up after swiftDialog: %w", err)
	}

	return nil
}

func (s *SwiftDialog) cleanup() error {
	s.cancel()
	cmdFileName := s.commandFile.Name()
	err := s.commandFile.Close()
	if err != nil {
		return fmt.Errorf("closing swiftDialog command file: %w", err)
	}
	err = os.Remove(cmdFileName)
	if err != nil {
		return fmt.Errorf("removing swiftDialog command file: %w", err)
	}

	return nil
}

type SwiftDialogExit struct {
	ExitCode int
	Output   map[string]any
}

func (s *SwiftDialog) Wait() (*SwiftDialogExit, error) {
	var exitCode int
	errExit := &exec.ExitError{}
	err := s.cmd.Wait()
	if err != nil {
		if errors.As(err, &errExit) {
			exitCode = errExit.ExitCode()
		} else {
			return nil, fmt.Errorf("waiting for swiftDialog: %w", err)
		}
	}

	parsed := map[string]any{}
	if s.output.Len() != 0 {
		err = json.Unmarshal(s.output.Bytes(), &parsed)
		if err != nil {
			return nil, fmt.Errorf("parsing swiftDialog output: %w", err)
		}
	}

	if err := s.cleanup(); err != nil {
		return nil, fmt.Errorf("Wait cleaning up after swiftDialog: %w", err)
	}

	return &SwiftDialogExit{
		ExitCode: exitCode,
		Output:   parsed,
	}, nil
}

func (s *SwiftDialog) sendCommand(command, arg string) error {
	// For some reason swiftDialog needs us to open and close the file
	// to detect a new command, just writing to the file doesn't cause
	// a change
	commandFile, err := os.OpenFile(s.commandFile.Name(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, CommandFilePerms)
	if err != nil {
		return fmt.Errorf("opening command file for writing: %w", err)
	}

	fullCommand := fmt.Sprintf("%s: %s", command, arg)

	_, err = fmt.Fprintf(commandFile, "%s\n", fullCommand)
	if err != nil {
		return fmt.Errorf("writing command to file: %w", err)
	}

	err = commandFile.Close()
	if err != nil {
		return fmt.Errorf("closing command file: %w", err)
	}

	return nil
}

// Title

func (s *SwiftDialog) UpdateTitle(title string) error {
	return s.sendCommand("title", title)
}

func (s *SwiftDialog) HideTitle() error {
	return s.sendCommand("title", "none")
}

// Message

func (s *SwiftDialog) UpdateMessage(text string) error {
	return s.sendCommand("message", text)
}

// Image

func (s *SwiftDialog) Image(pathOrUrl string) error {
	return s.sendCommand("image", pathOrUrl)
}

func (s *SwiftDialog) ImageCaption(caption string) error {
	return s.sendCommand("imagecaption", caption)
}

// Progress

func (s *SwiftDialog) UpdateProgress(progress uint) error {
	return s.sendCommand("progress", fmt.Sprintf("%d", progress))
}

func (s *SwiftDialog) IncrementProgress() error {
	return s.sendCommand("progress", "increment")
}

func (s *SwiftDialog) ResetProgress() error {
	return s.sendCommand("progress", "reset")
}

func (s *SwiftDialog) CompleteProgress() error {
	return s.sendCommand("progress", "complete")
}

func (s *SwiftDialog) HideProgress() error {
	return s.sendCommand("progress", "hide")
}

func (s *SwiftDialog) ShowProgress() error {
	return s.sendCommand("progress", "show")
}

func (s *SwiftDialog) UpdateProgressTest(text string) error {
	return s.sendCommand("progresstext", text)
}

// Lists

func (s *SwiftDialog) SetList(items []string) error {
	return s.sendCommand("list", strings.Join(items, ","))
}

func (s *SwiftDialog) ClearList() error {
	return s.sendCommand("list", "clear")
}

func (s *SwiftDialog) AddListItem(item ListItem) error {
	arg := fmt.Sprintf("add, title: %s", item.Title)
	if item.Status != "" {
		arg = fmt.Sprintf("%s, status: %s", arg, item.Status)
	}
	if item.StatusText != "" {
		arg = fmt.Sprintf("%s, statustext: %s", arg, item.StatusText)
	}
	return s.sendCommand("listitem", arg)
}

func (s *SwiftDialog) DeleteListItemByTitle(title string) error {
	return s.sendCommand("listitem", fmt.Sprintf("delete, title: %s", title))
}

func (s *SwiftDialog) DeleteListItemByIndex(index uint) error {
	return s.sendCommand("listitem", fmt.Sprintf("delete, index: %d", index))
}

func (s *SwiftDialog) UpdateListItemByTitle(title, statusText string, status Status) error {
	arg := fmt.Sprintf("title: %s, status: %s, statustext: %s", title, status, statusText)
	return s.sendCommand("listitem", arg)
}

func (s *SwiftDialog) UpdateListItemByIndex(index uint, statusText string, status Status) error {
	arg := fmt.Sprintf("index: %d, status: %s, statustext: %s", index, status, statusText)
	return s.sendCommand("listitem", arg)
}

// Buttons

func (s *SwiftDialog) EnableButton1(enable bool) error {
	arg := "disable"
	if enable {
		arg = "enable"
	}
	return s.sendCommand("button1", arg)
}

func (s *SwiftDialog) EnableButton2(enable bool) error {
	arg := "disable"
	if enable {
		arg = "enable"
	}
	return s.sendCommand("button2", arg)
}

func (s *SwiftDialog) SetButton1Text(text string) error {
	return s.sendCommand("button1text", text)
}

func (s *SwiftDialog) SetButton2Text(text string) error {
	return s.sendCommand("button2text", text)
}

func (s *SwiftDialog) SetInfoButtonText(text string) error {
	return s.sendCommand("infobuttontext", text)
}

// TODO remainder of updates
