package swiftdialog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type SwiftDialog struct {
	cancel      context.CancelFunc
	cmd         *exec.Cmd
	commandFile *os.File
}

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

	cmd := exec.CommandContext(
		ctx,
		swiftDialogBin,
		"--jsonstring", string(jsonBytes),
		"--comamndfile", commandFile.Name(),
		"--json",
	)

	err = cmd.Start()
	if err != nil {
		cancel()
		return nil, err
	}

	return &SwiftDialog{
		cancel:      cancel,
		cmd:         cmd,
		commandFile: commandFile,
	}, nil
}

func (s *SwiftDialog) Close() error {
	s.cancel()
	err := s.cmd.Wait()
	cmdFileName := s.commandFile.Name()
	s.commandFile.Close()
	os.Remove(cmdFileName)
	return err
}

func (s *SwiftDialog) sendCommand(command, arg string) error {
	_, err := fmt.Fprintf(s.commandFile, "%s: %s", command, arg)
	return err
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
