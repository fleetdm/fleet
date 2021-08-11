//+build !windows

package platform

import (
	"os"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/pkg/errors"
)

// ChmodExecutableDirectory sets the appropriate permissions on an executable
// file. On POSIX this is a normal chmod call.
func ChmodExecutableDirectory(path string) error {
	if err := os.Chmod(path, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "chmod executable directory")
	}
	return nil
}

// ChmodExecutable sets the appropriate permissions on the parent directory of
// an executable file. On POSIX this is a regular chmod call.
func ChmodExecutable(path string) error {
	if err := os.Chmod(path, constant.DefaultExecutableMode); err != nil {
		return errors.Wrap(err, "chmod executable")
	}
	return nil
}
