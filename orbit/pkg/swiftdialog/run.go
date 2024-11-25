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
	"time"
)

// SwiftDialog really wants the command file to be mode 666 for some reason
// https://github.com/swiftDialog/swiftDialog/wiki/Gotchas
var CommandFilePerms = fs.FileMode(0o666)

var (
	ErrKilled       = errors.New("process killed")
	ErrWindowClosed = errors.New("window closed")
)

type SwiftDialog struct {
	cancel      context.CancelCauseFunc
	cmd         *exec.Cmd
	commandFile *os.File
	context     context.Context
	output      *bytes.Buffer
	exitCode    ExitCode
	exitErr     error
	done        chan struct{}
	closed      bool
	binPath     string
}

type SwiftDialogExit struct {
	ExitCode ExitCode
	Output   map[string]any
}

type ExitCode int

const (
	ExitButton1               ExitCode = 0
	ExitButton2               ExitCode = 2
	ExitInfoButton            ExitCode = 3
	ExitTimer                 ExitCode = 4
	ExitQuitCommand           ExitCode = 5
	ExitQuitKey               ExitCode = 10
	ExitKeyAuthFailed         ExitCode = 30
	ExitImageResourceNotFound ExitCode = 201
	ExitFileNotFound          ExitCode = 202
)

func Create(ctx context.Context, swiftDialogBin string) (*SwiftDialog, error) {
	commandFile, err := os.CreateTemp("", "swiftDialogCommand")
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancelCause(ctx)

	if err := commandFile.Chmod(CommandFilePerms); err != nil {
		commandFile.Close()
		os.Remove(commandFile.Name())
		cancel(errors.New("could not create command file"))
		return nil, err
	}

	sd := &SwiftDialog{
		cancel:      cancel,
		commandFile: commandFile,
		context:     ctx,
		done:        make(chan struct{}),
		binPath:     swiftDialogBin,
	}

	return sd, nil
}

func (s *SwiftDialog) Start(ctx context.Context, opts *SwiftDialogOptions) error {
	jsonBytes, err := json.Marshal(opts)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext( //nolint:gosec
		ctx,
		s.binPath,
		"--jsonstring", string(jsonBytes),
		"--commandfile", s.commandFile.Name(),
		"--json",
	)

	s.cmd = cmd

	outBuf := &bytes.Buffer{}
	cmd.Stdout = outBuf

	s.output = outBuf

	err = cmd.Start()
	if err != nil {
		s.cancel(errors.New("could not start swiftDialog"))
		return err
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			errExit := &exec.ExitError{}
			if errors.As(err, &errExit) && strings.Contains(errExit.Error(), "exit status") {
				s.exitCode = ExitCode(errExit.ExitCode())
			} else {
				s.exitErr = fmt.Errorf("waiting for swiftDialog: %w", err)
			}
		}
		s.closed = true
		close(s.done)
		s.cancel(ErrWindowClosed)
	}()

	// This sleep makes sure that SD is fully up and running and has access to the command file.
	// We've found that if we start sending commands to the command file without this sleep, the
	// commands may be lost.
	time.Sleep(500 * time.Millisecond)

	return nil
}

func (s *SwiftDialog) finished() {
	<-s.done
}

func (s *SwiftDialog) Kill() error {
	s.cancel(ErrKilled)
	s.finished()
	if err := s.cleanup(); err != nil {
		return fmt.Errorf("Close cleaning up after swiftDialog: %w", err)
	}

	return nil
}

func (s *SwiftDialog) cleanup() error {
	s.cancel(nil)
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

func (s *SwiftDialog) Wait() (*SwiftDialogExit, error) {
	s.finished()

	parsed := map[string]any{}
	if s.output.Len() != 0 {
		if err := json.Unmarshal(s.output.Bytes(), &parsed); err != nil {
			return nil, fmt.Errorf("parsing swiftDialog output: %w", err)
		}
	}

	if err := s.cleanup(); err != nil {
		return nil, fmt.Errorf("Wait cleaning up after swiftDialog: %w", err)
	}

	return &SwiftDialogExit{
		ExitCode: s.exitCode,
		Output:   parsed,
	}, s.exitErr
}

func (s *SwiftDialog) Closed() bool {
	return s.closed
}

func (s *SwiftDialog) sendCommand(command, arg string) error {
	if err := s.context.Err(); err != nil {
		return fmt.Errorf("could not send command: %w", context.Cause(s.context))
	}

	fullCommand := fmt.Sprintf("%s: %s", command, arg)

	return s.writeCommand(fullCommand)
}

func (s *SwiftDialog) sendMultiCommand(commands ...string) error {
	multiCommands := strings.Join(commands, "\n")
	return s.writeCommand(multiCommands)
}

func (s *SwiftDialog) writeCommand(fullCommand string) error {
	// For some reason swiftDialog needs us to open and close the file
	// to detect a new command, just writing to the file doesn't cause
	// a change

	commandFile, err := os.OpenFile(s.commandFile.Name(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, CommandFilePerms)
	if err != nil {
		return fmt.Errorf("opening command file for writing: %w", err)
	}

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

///////////
// Title //
///////////

// Updates the dialog title
func (s *SwiftDialog) UpdateTitle(title string) error {
	return s.sendCommand("title", title)
}

// Hides the title area
func (s *SwiftDialog) HideTitle() error {
	return s.sendCommand("title", "none")
}

/////////////
// Message //
/////////////

// Set the dialog messsage
func (s *SwiftDialog) SetMessage(text string) error {
	return s.sendCommand("message", sanitize(text))
}

// Append to the dialog message
func (s *SwiftDialog) AppendMessage(text string) error {
	return s.sendCommand("message", fmt.Sprintf("+ %s", sanitize(text)))
}

// SetMessageKeepListItems sets the message to the given string while preserving the current list items.
func (s *SwiftDialog) SetMessageKeepListItems(message string) error {
	return s.sendMultiCommand(fmt.Sprintf("message: %s", sanitize(message)), "list: show")
}

///////////
// Image //
///////////

// Displays the selected image
func (s *SwiftDialog) Image(pathOrUrl string) error {
	return s.sendCommand("image", pathOrUrl)
}

// Displays the specified text underneath any displayed image
func (s *SwiftDialog) SetImageCaption(caption string) error {
	return s.sendCommand("imagecaption", caption)
}

//////////////
// Progress //
//////////////

// When Dialog is initiated with the Progress option, this will update the progress value
func (s *SwiftDialog) UpdateProgress(progress uint) error {
	return s.sendCommand("progress", fmt.Sprintf("%d", progress))
}

// Increments the progress by one
func (s *SwiftDialog) IncrementProgress() error {
	return s.sendCommand("progress", "increment")
}

// Resets the progress bar to 0
func (s *SwiftDialog) ResetProgress() error {
	return s.sendCommand("progress", "reset")
}

// Maxes out the progress bar
func (s *SwiftDialog) CompleteProgress() error {
	return s.sendCommand("progress", "complete")
}

// Hide the progress bar
func (s *SwiftDialog) HideProgress() error {
	return s.sendCommand("progress", "hide")
}

// Show the progress bar
func (s *SwiftDialog) ShowProgress() error {
	return s.sendCommand("progress", "show")
}

// Will update the label associated with the progress bar
func (s *SwiftDialog) UpdateProgressText(text string) error {
	return s.sendCommand("progresstext", text)
}

///////////
// Lists //
///////////

// Create a list
func (s *SwiftDialog) SetList(items []string) error {
	return s.sendCommand("list", strings.Join(items, ","))
}

// Clears the list and removes it from display
func (s *SwiftDialog) ClearList() error {
	return s.sendCommand("list", "clear")
}

// Add a new item to the end of the current list
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

// Delete an item by name
func (s *SwiftDialog) DeleteListItemByTitle(title string) error {
	return s.sendCommand("listitem", fmt.Sprintf("delete, title: %s", title))
}

// Delete an item by index number (starting at 0)
func (s *SwiftDialog) DeleteListItemByIndex(index uint) error {
	return s.sendCommand("listitem", fmt.Sprintf("delete, index: %d", index))
}

// Update a list item by name
func (s *SwiftDialog) UpdateListItemByTitle(title, statusText string, status Status, progressPercent ...uint) error {
	argStatus := string(status)
	if len(progressPercent) == 1 && status == StatusProgress {
		argStatus = fmt.Sprintf("progress, progress: %d", progressPercent[0])
	}
	arg := fmt.Sprintf("title: %s, status: %s, statustext: %s", title, argStatus, statusText)
	return s.sendCommand("listitem", arg)
}

// Update a list item by index number (starting at 0)
func (s *SwiftDialog) UpdateListItemByIndex(index uint, statusText string, status Status, progressPercent ...uint) error {
	argStatus := string(status)
	if len(progressPercent) == 1 && status == StatusProgress {
		argStatus = fmt.Sprintf("progress, progress: %d", progressPercent[0])
	}
	arg := fmt.Sprintf("index: %d, status: %s, statustext: %s", index, argStatus, statusText)
	return s.sendCommand("listitem", arg)
}

// ShowList forces the list to render.
func (s *SwiftDialog) ShowList() error {
	return s.sendCommand("list", "show")
}

/////////////
// Buttons //
/////////////

// Enable or disable button 1
func (s *SwiftDialog) EnableButton1(enable bool) error {
	arg := "disable"
	if enable {
		arg = "enable"
	}
	return s.sendCommand("button1", arg)
}

// Enable or disable button 2
func (s *SwiftDialog) EnableButton2(enable bool) error {
	arg := "disable"
	if enable {
		arg = "enable"
	}
	return s.sendCommand("button2", arg)
}

// Changes the button 1 label
func (s *SwiftDialog) SetButton1Text(text string) error {
	return s.sendCommand("button1text", text)
}

// Changes the button 2 label
func (s *SwiftDialog) SetButton2Text(text string) error {
	return s.sendCommand("button2text", text)
}

// Changes the info button label
func (s *SwiftDialog) SetInfoButtonText(text string) error {
	return s.sendCommand("infobuttontext", text)
}

//////////////
// Info box //
//////////////

// Update the content in the info box
func (s *SwiftDialog) SetInfoBoxText(text string) error {
	return s.sendCommand("infobox", sanitize(text))
}

// Append to the conteit in the info box
func (s *SwiftDialog) AppendInfoBoxText(text string) error {
	return s.sendCommand("infobox", fmt.Sprintf("+ %s", sanitize(text)))
}

//////////
// Icon //
//////////

// Changes the displayed icon
// See https://github.com/swiftDialog/swiftDialog/wiki/Customising-the-Icon
func (s *SwiftDialog) SetIconLocation(location string) error {
	return s.sendCommand("icon", location)
}

// Moves the icon being shown
func (s *SwiftDialog) SetIconAlignment(alignment Alignment) error {
	return s.sendCommand("icon", string(alignment))
}

// Hide the icon
func (s *SwiftDialog) HideIcon() error {
	return s.sendCommand("icon", "hide")
}

// Changes the size of the displayed icon
func (s *SwiftDialog) SetIconSize(size uint) error {
	return s.sendCommand("icon", fmt.Sprintf("size: %d", size))
}

////////////
// Window //
////////////

// Changes the width of the window maintaining the current position
func (s *SwiftDialog) SetWindowWidth(width uint) error {
	return s.sendCommand("width", fmt.Sprintf("%d", width))
}

// Changes the height of the window maintaining the current position
func (s *SwiftDialog) SetWindowHeight(width uint) error {
	return s.sendCommand("height", fmt.Sprintf("%d", width))
}

// Changes the window position
func (s *SwiftDialog) SetWindowPosition(position FullPosition) error {
	return s.sendCommand("position", string(position))
}

// Display content from the specified URL
func (s *SwiftDialog) SetWebContent(url string) error {
	return s.sendCommand("webcontent", url)
}

// Hide web content
func (s *SwiftDialog) HideWebContent() error {
	return s.sendCommand("webcontent", "none")
}

// Display a video from the specified path or URL
func (s *SwiftDialog) SetVideo(location string) error {
	return s.sendCommand("video", location)
}

// Enables or disables the blur window layer
func (s *SwiftDialog) BlurScreen(enable bool) error {
	blur := "disable"
	if enable {
		blur = "enable"
	}
	return s.sendCommand("blurscreen", blur)
}

// Activates the dialog window and brings it to the forground
func (s *SwiftDialog) Activate() error {
	return s.sendCommand("activate", "")
}

// Quits dialog with exit code 5 (ExitQuitCommand)
func (s *SwiftDialog) Quit() error {
	return s.sendCommand("quit", "")
}

func sanitize(text string) string {
	return strings.ReplaceAll(text, "\n", "\\n")
}
