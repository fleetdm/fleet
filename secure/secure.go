package secure

import (
	"os"
	"path"
	"syscall"

	"github.com/pkg/errors"
)

func isMorePermissive(currentMode, newMode os.FileMode) bool {
	currentGroup := currentMode >> 1 & 07
	newGroup := newMode >> 1 & 07
	currentAll := currentMode & 0x7
	newAll := newMode & 0x7

	return newGroup > currentGroup || newAll > currentAll
}

func checkPermPath(path string, perm os.FileMode) error {
	if !perm.IsDir() {
		perm = perm ^ os.ModeDir
	}

	dir, err := os.Stat(path)
	if err == nil {
		if dir.IsDir() {
			if isMorePermissive(dir.Mode(), perm) {
				return errors.Errorf(
					"Path %s already exists with mode %o instead of the expected %o", path, dir.Mode(), perm)
			}
			return nil
		} else {
			return &os.PathError{Op: "mkdir", Path: path, Err: syscall.ENOTDIR}
		}
	}

	i := len(path)
	for i > 0 && os.IsPathSeparator(path[i-1]) { // Skip trailing path separator.
		i--
	}

	j := i
	for j > 0 && !os.IsPathSeparator(path[j-1]) { // Scan backward over element.
		j--
	}

	if j > 1 {
		return checkPermPath(path[:j-1], perm)
	}

	return nil
}

func checkPermFile(filePath string, perm os.FileMode) error {
	if f, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) && f != nil && f.Mode() != perm {
		return errors.Errorf(
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
