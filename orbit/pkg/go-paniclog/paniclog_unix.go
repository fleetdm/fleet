// Log the panic under unix to the log file

// +build !windows,!solaris,!plan9

package paniclog

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func redirectStderr(f *os.File) (UndoFunction, error) {

	stderrFd := int(os.Stderr.Fd())
	oldfd, err := unix.Dup(stderrFd)
	if err != nil {
		return nil, errors.New("Failed to redirect stderr to file: " + err.Error())
	}

	err = unix.Dup2(int(f.Fd()), stderrFd)
	if err != nil {
		return nil, errors.New("Failed to redirect stderr to file: " + err.Error())
	}

	undo := func() error {
		undoErr := unix.Dup2(oldfd, stderrFd)
		unix.Close(oldfd)

		if undoErr != nil {
			return errors.New("Failed to reverse stderr redirection: " + err.Error())
		}

		return nil
	}

	return undo, nil
}
