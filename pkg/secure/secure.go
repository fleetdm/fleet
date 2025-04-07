//go:build !windows
// +build !windows

package secure

import (
	"errors"
	"fmt"
	"os"
	"path"
	"syscall"
)

func isMorePermissive(currentMode, newMode os.FileMode) bool {
	currentGroup := currentMode & 070
	newGroup := newMode & 070
	currentAll := currentMode & 07
	newAll := newMode & 07

	return newGroup > currentGroup || newAll > currentAll
}

func checkPermPath(path string, perm os.FileMode) error {
	if !perm.IsDir() {
		perm ^= os.ModeDir
	}

	dir, err := os.Stat(path)
	if err == nil {
		if dir.IsDir() {
			if isMorePermissive(dir.Mode(), perm) {
				return fmt.Errorf(
					"Path %s already exists with mode %o instead of the expected %o", path, dir.Mode(), perm)
			}
			return nil
		}
		return &os.PathError{Op: "mkdir", Path: path, Err: syscall.ENOTDIR}
	}

	// Look for the parent directory in the path and then check the permissions in that
	// This is based on the logic from os.MkdirAll
	i := len(path)
	// This first loop skips over trailing path separators. Eg:
	// /some/path//////
	//          ^ i will end up here
	for i > 0 && os.IsPathSeparator(path[i-1]) { // Skip trailing path separator.
		i--
	}

	j := i
	// This loop starts from where i left off and finds the previous path separator
	// /some/path//////
	//      ^ j will end up here
	for j > 0 && !os.IsPathSeparator(path[j-1]) { // Scan backward over element.
		j--
	}

	// if we are pointing at a path separator that is not the root separator, then check for the path before it
	// /some
	if j > 1 {
		return checkPermPath(path[:j-1], perm)
	}

	return nil
}

func checkPermFile(filePath string, perm os.FileMode) error {
	if f, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) && f != nil && f.Mode() != perm {
		return fmt.Errorf(
			"File %s already exists with mode %o instead of the expected %o", filePath, f.Mode(), perm)
	}
	if err := checkPermPath(path.Dir(filePath), perm); err != nil {
		return err
	}

	return nil
}

func MkdirAll(path string, perm os.FileMode) error {
	if err := checkPermPath(path, perm); err != nil {
		return err
	}
	return os.MkdirAll(path, perm)
}

func OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	if err := checkPermFile(name, perm); err != nil {
		return nil, err
	}
	return os.OpenFile(name, flag, perm)
}
