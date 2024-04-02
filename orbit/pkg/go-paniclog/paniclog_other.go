// Log the panic to the log file - for oses which can't do this

//go:build !windows && !darwin && !dragonfly && !freebsd && !linux && !nacl && !netbsd && !openbsd
// +build !windows,!darwin,!dragonfly,!freebsd,!linux,!nacl,!netbsd,!openbsd

package paniclog

import (
	"errors"
	"os"
)

func redirectStderr(f *os.File) (UndoFunction, error) {
	return nil, errors.New("Can't redirect stderr to file")
}
