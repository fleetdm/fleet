package secure

import (
	"os"
	"syscall"

	"github.com/pkg/errors"
)

func MkdirAll(path string, perm os.FileMode) error {
	dir, err := os.Stat(path)
	if err == nil {
		if dir.IsDir() {
			if dir.Mode() != perm {
				return errors.Errorf(
					"Path %s already exists with mode %o instead of the expected %o", path, dir.Mode(), perm)
			}
			return nil
		}
		return &os.PathError{Op: "mkdir", Path: path, Err: syscall.ENOTDIR}
	}
	return os.MkdirAll(path, perm)
}

func OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	if f, err := os.Stat(name); !errors.Is(err, os.ErrNotExist) && f.Mode() != perm {
		return nil, errors.Errorf(
			"File %s already exists with mode %o instead of the expected %o", name, f.Mode(), perm)
	}
	return os.OpenFile(name, flag, perm)
}
